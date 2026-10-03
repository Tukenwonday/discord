package middleware

import (
	"net/http"
	"strconv"

	"github.com/cordis/backend/internal/cache"
	"github.com/cordis/backend/internal/httpx"
	"github.com/gin-gonic/gin"
)

// Rate limit scopes, one per contract section 5 route group.
const (
	ScopeLogin    = "login"
	ScopeRegister = "register"
	ScopeMessage  = "message"
)

// Per scope limits applied inside a 60 second window.
const (
	LoginLimit    = 10
	RegisterLimit = 5
	MessageLimit  = 60
)

// RateLimit counts requests per scope and client in Redis, answering with the
// 429 envelope once the budget is exhausted.
func RateLimit(rdb *cache.Client, scope string, limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := clientID(c)
		allowed, remaining, err := rdb.Allow(c.Request.Context(), scope, id, limit)
		if err != nil {
			// A Redis outage must not block authentication traffic.
			c.Next()
			return
		}
		if !allowed {
			httpx.RespondError(c, httpx.NewRateLimited("too many requests, slow down"))
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Next()
	}
}

// RateLimitByIP builds a limiter keyed by the client address, used for the
// login and register routes where no user id exists yet.
func RateLimitByIP(rdb *cache.Client, scope string, limit int) gin.HandlerFunc {
	return RateLimit(rdb, scope, limit)
}

// clientID prefers the authenticated user id and falls back to the remote
// address so anonymous traffic is still throttled.
func clientID(c *gin.Context) string {
	if uid := UserID(c); uid != "" {
		return uid
	}
	return c.ClientIP()
}

// NoContent is a tiny helper used by 204 handlers.
func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }