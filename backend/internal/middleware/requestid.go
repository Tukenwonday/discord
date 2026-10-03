// Package middleware holds the gin middleware chain: request ids, structured
// logging, panic recovery, CORS, authentication and rate limiting.
package middleware

import (
	"github.com/cordis/backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextRequestID is the gin context key holding the request id.
const ContextRequestID = "request_id"

// RequestID echoes an incoming X-Request-ID or generates a new UUID and stores
// it on the context so logs and handlers can reference it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 128 {
			id = uuid.NewString()
		}
		c.Set(ContextRequestID, id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

// RequestIDFrom returns the request id stored on the gin context.
func RequestIDFrom(c *gin.Context) string { return logger.GinRequestID(c) }