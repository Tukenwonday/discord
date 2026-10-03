package ws

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/gorilla/websocket"
)

// Socket tuning values mandated by section 6 of the contract.
const (
	// ReadLimit is the maximum size of one inbound frame.
	ReadLimit = 64 << 10
	// WriteWait is the deadline applied to every write.
	WriteWait = 10 * time.Second
	// PongWait is how long the server tolerates silence before assuming the peer
	// is gone.
	PongWait = 60 * time.Second
	// PingPeriod must stay below PongWait so a ping is always sent in time.
	PingPeriod = 25 * time.Second
	// sendBufferSize is the depth of the per socket outbox.
	sendBufferSize = 256
)

// client is one authenticated WebSocket connection. Sends go through a buffered
// channel drained by writePump, so a slow reader can never block a publisher.
type client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID string
	send   chan []byte
	// sessionID identifies this socket in the ready event.
	sessionID string
	closeOnce chan struct{}
}

// newClient wires a socket to its hub.
func newClient(hub *Hub, conn *websocket.Conn, userID, sessionID string) *client {
	return &client{
		hub:       hub,
		conn:      conn,
		userID:    userID,
		sessionID: sessionID,
		send:      make(chan []byte, sendBufferSize),
		closeOnce: make(chan struct{}),
	}
}

// enqueue hands a frame to the write pump, dropping the socket when its outbox
// is full so a stalled peer cannot exhaust memory.
func (c *client) enqueue(frame []byte) {
	select {
	case <-c.closeOnce:
		return
	default:
	}
	select {
	case c.send <- frame:
	default:
		c.hub.log.Warn().Str("user_id", c.userID).Msg("client send buffer full, closing socket")
		c.close()
	}
}

// close shuts the socket down exactly once.
func (c *client) close() {
	select {
	case <-c.closeOnce:
		return
	default:
		close(c.closeOnce)
	}
	_ = c.conn.Close()
}

// sendFrame queues one envelope for delivery.
func (c *client) sendFrame(event string, data any) {
	c.enqueue(marshalEnvelope(event, data))
}

// sendError answers a bad frame with the error event, echoing the nonce so the
// client can correlate it with the request that failed.
func (c *client) sendError(nonce string, err error) {
	payload := ErrorPayload{Code: "internal_error", Message: "internal server error", Nonce: nonce}
	var apiErr *httpx.APIError
	if errors.As(err, &apiErr) {
		payload.Code = apiErr.Code
		payload.Message = apiErr.Message
	}
	c.enqueue(marshalEnvelope(EventError, payload))
}

// readPump consumes inbound frames and hands them to the dispatcher. It returns
// when the peer closes or a frame violates the protocol, and it never replies on
// its own so error rendering stays in the dispatcher.
func (c *client) readPump() {
	defer func() {
		c.hub.Unregister(c)
	}()

	c.conn.SetReadLimit(ReadLimit)
	_ = c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(PongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				c.hub.log.Debug().Str("user_id", c.userID).Msg("websocket read ended")
			}
			return
		}

		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			c.sendError("", httpx.NewValidation("frame is not a valid envelope"))
			continue
		}
		if env.Event == "" {
			c.sendError(env.Nonce, httpx.NewValidation("event name is required"))
			continue
		}

		// A handler failure produces an error event rather than dropping the
		// connection, so one bad frame never costs the client its session.
		if err := c.hub.dispatch(c, env); err != nil {
			c.sendError(env.Nonce, err)
		}
	}
}

// writePump owns every write to the socket: queued frames, the 25 second ping
// and the close frame on shutdown.
func (c *client) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.Unregister(c)
	}()

	for {
		select {
		case frame, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, frame); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closeOnce:
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			_ = c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

// upgrader turns an HTTP request into a WebSocket, rejecting any origin that is
// not part of CORS_ORIGINS.
func upgrader(origins []string) *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return middleware.AllowedOrigin(origins, r.Header.Get("Origin"))
		},
	}
}