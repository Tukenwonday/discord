package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// Recovery converts a panic into a 500 error envelope and never lets a single
// request take the process down.
func Recovery() gin.HandlerFunc {
	log := logger.New()
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Interface("panic", rec).
					Str("request_id", logger.GinRequestID(c)).
					Str("path", c.Request.URL.Path).
					Bytes("stack", debug.Stack()).
					Msg("recovered panic")
				if !c.Writer.Written() {
					httpx.RespondError(c, httpx.NewInternal("internal server error"))
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}

// NoRoute renders the contract error envelope for unknown paths.
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.RespondError(c, httpx.NewNotFound("route not found"))
	}
}

// MethodNotAllowed renders the contract error envelope for wrong methods.
func MethodNotAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.RespondError(c, &httpx.APIError{
			Code:       httpx.CodeNotFound,
			Message:    "method not allowed",
			HTTPStatus: http.StatusMethodNotAllowed,
		})
	}
}