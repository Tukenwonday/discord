// Package logger configures zerolog and carries the request id through the
// request context so every log line can be correlated with a client call.
package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type contextKey string

// requestIDKey is the private context key holding the current request id.
const requestIDKey contextKey = "cordis.request_id"

var root = newLogger("development")

// Init builds the global logger. Production emits JSON, development emits a
// human readable console stream.
func Init(appEnv string) {
	root = newLogger(appEnv)
}

// L returns the global logger.
func L() *zerolog.Logger { return &root }

// New returns an independent logger, used by background goroutines.
func New() zerolog.Logger { return root }

// WithRequestID stores the request id in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID returns the request id stored in the context, if any.
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// GinRequestID returns the request id attached to a gin context.
func GinRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func newLogger(appEnv string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	if strings.EqualFold(appEnv, "production") {
		return zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Str("service", "cordis-backend").Logger()
	}
	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		Level(zerolog.DebugLevel).
		With().Timestamp().Str("service", "cordis-backend").Logger()
}
