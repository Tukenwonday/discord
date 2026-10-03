// Command server boots the Cordis backend: configuration, structured logger,
// Postgres and Redis connections with retry, the WebSocket hub, the service
// layer and the HTTP router, followed by a graceful shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cordis/backend/internal/auth"
	"github.com/cordis/backend/internal/cache"
	"github.com/cordis/backend/internal/config"
	"github.com/cordis/backend/internal/database"
	"github.com/cordis/backend/internal/handlers"
	"github.com/cordis/backend/internal/logger"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/cordis/backend/internal/storage"
	"github.com/cordis/backend/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// Version is reported by the liveness probe and stamped at build time.
var Version = "dev"

// shutdownTimeout bounds how long in flight requests may finish.
const shutdownTimeout = 15 * time.Second

// redisRetryAttempts is how many times the process tries to reach Redis before
// giving up, which covers a slow starting container.
const redisRetryAttempts = 10

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

// run wires every component and serves until a termination signal arrives.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger.Init(cfg.AppEnv)
	log := logger.New()
	gin.SetMode(cfg.GinMode)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	if err := database.AutoMigrate(db); err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("obtain sql handle: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	rdb, err := connectRedis(rootCtx, cfg.RedisURL, log)
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()

	driver, err := buildDriver(rootCtx, cfg)
	if err != nil {
		return err
	}

	tokens := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// The hub is the single Publisher. It is created with a deps value whose Pub
	// field is itself, which closes the cycle the hub needs in order to resolve
	// channel audiences while publishing.
	deps := service.Deps{
		DB:     db,
		Cache:  rdb,
		Tokens: tokens,
		Config: cfg,
		Log:    log,
	}
	hub := ws.NewHub(cfg, rdb, deps, log)
	deps.Pub = hub

	stopRedis := hub.Subscribe(rootCtx)
	defer stopRedis()
	go hub.StartPresenceHeartbeat(rootCtx)

	router, err := buildRouter(cfg, deps, db, rdb, driver, hub)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", cfg.Addr()).Str("version", Version).Msg("http server listening")
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case serveErr := <-errCh:
		if serveErr != nil {
			return fmt.Errorf("http server: %w", serveErr)
		}
		return nil
	case <-rootCtx.Done():
		log.Info().Msg("shutdown signal received")
	}

	// The hub closes its sockets first so clients reconnect to another replica
	// instead of hanging on a half closed connection.
	hub.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info().Msg("shutdown complete")
	return nil
}

// connectRedis dials Redis, retrying with a backoff so the process can start
// alongside its Redis container.
func connectRedis(ctx context.Context, redisURL string, log zerolog.Logger) (*cache.Client, error) {
	backoff := time.Second
	var lastErr error
	for attempt := 1; attempt <= redisRetryAttempts; attempt++ {
		client, err := cache.New(ctx, redisURL)
		if err == nil {
			return client, nil
		}
		lastErr = err
		log.Warn().Err(err).Int("attempt", attempt).Msg("waiting for redis")
		if attempt == redisRetryAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 10*time.Second {
			backoff *= 2
		}
	}
	return nil, fmt.Errorf("connect redis: %w", lastErr)
}

// buildDriver selects the attachment driver named by UPLOAD_DRIVER.
func buildDriver(ctx context.Context, cfg *config.Config) (storage.Driver, error) {
	if cfg.Upload.Driver == config.UploadDriverS3 {
		return storage.NewS3(ctx, storage.S3Options{
			Endpoint:       cfg.Upload.S3.Endpoint,
			Region:         cfg.Upload.S3.Region,
			Bucket:         cfg.Upload.S3.Bucket,
			AccessKey:      cfg.Upload.S3.AccessKey,
			SecretKey:      cfg.Upload.S3.SecretKey,
			ForcePathStyle: cfg.Upload.S3.ForcePathStyle,
			PublicBase:     cfg.Upload.PublicBase,
			MaxBytes:       cfg.Upload.MaxBytes,
		})
	}
	local, err := storage.NewLocal(cfg.Upload.Dir, cfg.Upload.PublicBase, cfg.Upload.MaxBytes)
	if err != nil {
		return nil, err
	}
	return local, nil
}
// buildRouter registers every endpoint of section 5 behind the middleware chain
// and returns the HTTP handler.
func buildRouter(
	cfg *config.Config,
	deps service.Deps,
	db *gorm.DB,
	rdb *cache.Client,
	driver storage.Driver,
	hub *ws.Hub,
) (*gin.Engine, error) {
	router := gin.New()
	// Without an explicit list the client address must not be trusted, so rate
	// limiting keys on the real socket peer rather than a spoofable header.
	if err := router.SetTrustedProxies(cfg.TrustedProxyList()); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}

	router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.CORS(cfg.CORSOrigins),
	)
	router.NoRoute(middleware.NoRoute())
	router.NoMethod(middleware.MethodNotAllowed())

	authRequired := middleware.AuthRequired(deps.Tokens)

	health := handlers.NewHealth(db, rdb, Version)
	router.GET("/healthz", health.Live)
	router.GET("/readyz", health.Ready)

	if local, ok := driver.(*storage.Local); ok {
		router.GET("/uploads/*filepath", serveFiles(local.Dir()))
	}

	// The WebSocket endpoint is served both at the root and under the API base,
	// as required by section 6.
	router.GET("/ws", hub.Handle)
	router.GET("/api/ws", hub.Handle)

	api := router.Group("/api")
	{
		authHandler := handlers.NewAuth(deps)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register",
				middleware.RateLimitByIP(rdb, middleware.ScopeRegister, middleware.RegisterLimit),
				authHandler.Register)
			authGroup.POST("/login",
				middleware.RateLimitByIP(rdb, middleware.ScopeLogin, middleware.LoginLimit),
				authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
			authGroup.POST("/logout", authHandler.Logout)
		}

		invites := handlers.NewInvites(deps)
		// Previewing an invite is public so a link can be opened before signing in.
		api.GET("/invites/:code", invites.Get)

		private := api.Group("")
		private.Use(authRequired)
		{
			users := handlers.NewUsers(deps)
			private.GET("/users", users.Search)
			private.GET("/users/me", users.Me)
			private.PATCH("/users/me", users.UpdateMe)
			private.GET("/users/:id", users.Get)

			servers := handlers.NewServers(deps)
			private.GET("/servers", servers.List)
			private.POST("/servers", servers.Create)
			private.GET("/servers/:id", servers.Get)
			private.PATCH("/servers/:id", servers.Update)
			private.DELETE("/servers/:id", servers.Delete)

			channels := handlers.NewChannels(deps)
			private.POST("/servers/:id/channels", channels.Create)
			private.PATCH("/channels/:id", channels.Update)
			private.DELETE("/channels/:id", channels.Delete)

			messages := handlers.NewMessages(deps)
			private.GET("/channels/:id/messages", messages.List)
			private.POST("/channels/:id/messages",
				middleware.RateLimit(rdb, middleware.ScopeMessage, middleware.MessageLimit),
				messages.Create)
			private.PATCH("/messages/:id", messages.Update)
			private.DELETE("/messages/:id", messages.Delete)

			reactions := handlers.NewReactions(deps)
			private.POST("/messages/:id/reactions", reactions.Toggle)
			private.DELETE("/messages/:id/reactions", reactions.Remove)
			private.GET("/messages/:id/reactions", reactions.List)

			dms := handlers.NewDMs(deps)
			private.GET("/dms", dms.List)
			private.POST("/dms", dms.Open)
			private.GET("/dms/:id", dms.Get)
			private.DELETE("/dms/:id", dms.Leave)
			private.GET("/dms/:id/messages", dms.ListMessages)
			private.POST("/dms/:id/messages",
				middleware.RateLimit(rdb, middleware.ScopeMessage, middleware.MessageLimit),
				dms.CreateMessage)
			private.GET("/dms/:id/recipients", dms.Recipients)

			friends := handlers.NewFriends(deps)
			private.GET("/friends", friends.Lists)
			private.POST("/friends", friends.Request)
			private.PATCH("/friends/:id", friends.Accept)
			private.DELETE("/friends/:id", friends.Remove)

			private.GET("/servers/:id/invites", invites.List)
			private.POST("/servers/:id/invites", invites.Create)
			private.POST("/invites/:code/join", invites.Join)
			private.DELETE("/invites/:code", invites.Revoke)

			private.POST("/livekit/token", handlers.NewLiveKit(deps).Token)
			private.POST("/upload", handlers.NewUpload(driver, cfg.Upload.MaxMB).File)
		}
	}

	return router, nil
}

// serveFiles serves stored uploads with an immutable cache policy, since a file
// is written once under a uuid name and never mutated afterwards.
func serveFiles(root string) gin.HandlerFunc {
	fileServer := http.FileServer(http.Dir(root))
	return func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}