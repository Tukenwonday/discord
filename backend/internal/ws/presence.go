package ws

import (
	"context"
	"time"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// PresenceHeartbeatPeriod is how often the online marker of every connected
// user is refreshed, chosen well inside the 5 minute TTL of section 8.
const PresenceHeartbeatPeriod = 45 * time.Second

// Handle upgrades an authenticated request to a WebSocket and starts the socket
// pumps. An auth failure is refused before the upgrade so the client receives
// the contract error envelope instead of an opaque socket close.
func (h *Hub) Handle(c *gin.Context) {
	token := middleware.BearerToken(c)
	if token == "" {
		httpx.RespondError(c, httpx.NewUnauthorized("authentication required"))
		return
	}
	claims, err := h.deps.Tokens.ParseAccess(token)
	if err != nil {
		httpx.RespondError(c, httpx.NewUnauthorized("invalid or expired token"))
		return
	}

	up := upgrader(h.cfg.CORSOrigins)
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade already wrote its own HTTP error response.
		h.log.Warn().Err(err).Str("user_id", claims.UserID).Msg("websocket upgrade failed")
		return
	}

	socket := newClient(h, conn, claims.UserID, uuid.NewString())
	h.Register(socket)

	// The ready event is queued immediately so the client learns its session id
	// before it starts waiting on the outbox.
	socket.enqueue(marshalEnvelope(EventReady, ReadyPayload{
		UserID:     claims.UserID,
		SessionID:  socket.sessionID,
		ServerTime: time.Now().UTC(),
	}))

	h.connectPresence(claims.UserID)
}

// connectPresence marks the user online in Redis and broadcasts the change to
// the people who share a server with them.
func (h *Hub) connectPresence(userID string) {
	store := service.NewPresenceStore(h.deps)
	if err := store.Connect(context.Background(), userID, models.StatusOnline, ""); err != nil {
		h.log.Warn().Err(err).Str("user_id", userID).Msg("mark online failed")
	}
}

// StartPresenceHeartbeat runs until ctx is cancelled and refreshes the online
// marker of every locally connected user, which is what keeps the 5 minute
// presence TTL of section 8 alive during a long session.
func (h *Hub) StartPresenceHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(PresenceHeartbeatPeriod)
	defer ticker.Stop()

	store := service.NewPresenceStore(h.deps)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, userID := range h.reg.onlineUsers() {
				if err := store.Heartbeat(userID); err != nil {
					h.log.Warn().Err(err).Str("user_id", userID).Msg("presence heartbeat failed")
				}
			}
		}
	}
}

// Upgrader exposes the origin checked upgrader so a handshake can be driven in a
// test without the gin plumbing.
func (h *Hub) Upgrader() *websocket.Upgrader { return upgrader(h.cfg.CORSOrigins) }