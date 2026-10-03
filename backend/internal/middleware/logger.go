package middleware

import (
	"time"

	"github.com/cordis/backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// Logger emits one structured line per request with the request id, method,
// path, status, latency and the authenticated user id when present.
func Logger() gin.HandlerFunc {
	log := logger.New()
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()
		fields := []any{
			int64(status),
			c.Request.Method,
			path,
			time.Since(start).Milliseconds(),
			c.ClientIP(),
			c.Writer.Size(),
			logger.GinRequestID(c),
		}
		if query != "" {
			fields = append(fields, query)
		}
		if uid := UserID(c); uid != "" {
			fields = append(fields, uid)
		}
		if len(c.Errors) > 0 {
			fields = append(fields, c.Errors.String())
		}
		switch {
		case status >= 500:
			log.Error().Fields(fields).Msg("request failed")
		case status >= 400:
			log.Warn().Fields(fields).Msg("request rejected")
		default:
			log.Info().Fields(fields).Msg("request")
		}
	}
}