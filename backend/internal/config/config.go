// Package config parses and validates every environment variable described in
// section 2 of the Cordis contract.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// UploadDriverLocal stores attachments on the local filesystem.
const UploadDriverLocal = "local"

// UploadDriverS3 stores attachments in an S3 compatible object store.
const UploadDriverS3 = "s3"

// Config is the fully resolved runtime configuration.
type Config struct {
	AppEnv           string
	Port             string
	GinMode          string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	CORSOrigins      []string
	PublicURL        string
	RedisChannel     string
	TrustedProxies   []string
	MaxMessageLen    int
	SelfHostedDomain string
	LiveKit          LiveKit
	Upload           Upload
}

// LiveKit holds the credentials used to mint video access tokens.
type LiveKit struct {
	URL       string
	APIKey    string
	APISecret string
}

// Upload holds the attachment storage configuration.
type Upload struct {
	Driver     string
	Dir        string
	PublicBase string
	MaxMB      int
	MaxBytes   int64
	S3         S3
}

// S3 holds the MinIO / S3 compatible object storage settings.
type S3 struct {
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
}

// IsProduction reports whether the process runs with APP_ENV=production.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// Addr returns the listen address for the HTTP server.
func (c *Config) Addr() string { return ":" + c.Port }

// TrustedProxyList returns the proxy CIDRs, defaulting to none.
func (c *Config) TrustedProxyList() []string {
	if len(c.TrustedProxies) == 0 {
		return nil
	}
	return c.TrustedProxies
}

// Load reads the configuration from the process environment.
func Load() (*Config, error) {
	accessTTL, err := lookupDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := lookupDuration("REFRESH_TOKEN_TTL", 720*time.Hour)
	if err != nil {
		return nil, err
	}
	maxUploadMB, err := lookupInt("MAX_UPLOAD_MB", 25)
	if err != nil {
		return nil, err
	}
	if maxUploadMB <= 0 {
		return nil, fmt.Errorf("MAX_UPLOAD_MB must be greater than zero")
	}
	maxMessageLen, err := lookupInt("MAX_MESSAGE_LEN", 4000)
	if err != nil {
		return nil, err
	}
	if maxMessageLen <= 0 {
		return nil, fmt.Errorf("MAX_MESSAGE_LEN must be greater than zero")
	}

	cfg := &Config{
		AppEnv:           lookup("APP_ENV", "development"),
		Port:             lookup("PORT", "8080"),
		GinMode:          lookup("GIN_MODE", "debug"),
		DatabaseURL:      lookup("DATABASE_URL", "postgres://cordis:cordis@localhost:5432/cordis?sslmode=disable"),
		RedisURL:         lookup("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:        lookup("JWT_SECRET", "change-me-in-production-min-32-chars"),
		AccessTokenTTL:   accessTTL,
		RefreshTokenTTL:  refreshTTL,
		CORSOrigins:      lookupList("CORS_ORIGINS", []string{"http://localhost:1420", "http://localhost:5173"}),
		PublicURL:        strings.TrimRight(lookup("PUBLIC_URL", "http://localhost:8080"), "/"),
		RedisChannel:     lookup("REDIS_CHANNEL", "cordis:events"),
		TrustedProxies:   lookupList("TRUSTED_PROXIES", nil),
		MaxMessageLen:    maxMessageLen,
		SelfHostedDomain: strings.TrimRight(lookup("SELF_HOSTED_DOMAIN", "http://localhost:5173"), "/"),
		LiveKit: LiveKit{
			URL:       lookup("LIVEKIT_URL", "ws://localhost:7880"),
			APIKey:    lookup("LIVEKIT_API_KEY", "devkey"),
			APISecret: lookup("LIVEKIT_API_SECRET", "secret"),
		},
		Upload: Upload{
			Driver:     strings.ToLower(lookup("UPLOAD_DRIVER", UploadDriverLocal)),
			Dir:        lookup("UPLOAD_DIR", "./data/uploads"),
			PublicBase: strings.TrimRight(lookup("UPLOAD_PUBLIC_BASE", "http://localhost:8080/uploads"), "/"),
			MaxMB:      maxUploadMB,
			MaxBytes:   int64(maxUploadMB) * 1024 * 1024,
			S3: S3{
				Endpoint:       lookup("S3_ENDPOINT", ""),
				Region:         lookup("S3_REGION", "us-east-1"),
				Bucket:         lookup("S3_BUCKET", "cordis"),
				AccessKey:      lookup("S3_ACCESS_KEY", ""),
				SecretKey:      lookup("S3_SECRET_KEY", ""),
				ForcePathStyle: lookupBool("S3_FORCE_PATH_STYLE", true),
			},
		},
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT must not be empty")
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("PORT must be numeric: %w", err)
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be positive")
	}
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET must not be empty")
	}
	if c.IsProduction() && len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if len(c.CORSOrigins) == 0 {
		return fmt.Errorf("CORS_ORIGINS must contain at least one origin")
	}
	switch c.Upload.Driver {
	case UploadDriverLocal:
	case UploadDriverS3:
		if c.Upload.S3.Endpoint == "" || c.Upload.S3.AccessKey == "" || c.Upload.S3.SecretKey == "" {
			return fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY are required when UPLOAD_DRIVER=s3")
		}
	default:
		return fmt.Errorf("UPLOAD_DRIVER must be %q or %q", UploadDriverLocal, UploadDriverS3)
	}
	if c.RedisChannel == "" {
		return fmt.Errorf("REDIS_CHANNEL must not be empty")
	}
	return nil
}

func lookup(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func lookupInt(key string, def int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return v, nil
}

func lookupBool(key string, def bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def
	}
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return def
	}
	return v
}

func lookupDuration(key string, def time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	v, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s must be a Go duration: %w", key, err)
	}
	return v, nil
}

func lookupList(key string, def []string) []string {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}