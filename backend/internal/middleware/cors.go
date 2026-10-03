package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSAllowedMethods are the methods advertised to the browser.
var CORSAllowedMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPatch,
	http.MethodPut, http.MethodDelete, http.MethodOptions,
}

// CORSAllowedHeaders are the request headers the client may send.
var CORSAllowedHeaders = []string{"Authorization", "Content-Type"}

// CORSExposedHeaders are the response headers the browser may read.
var CORSExposedHeaders = []string{"Content-Length"}

// CORSMaxAge is the preflight cache duration in seconds.
const CORSMaxAge = 86400

// CORS answers preflight requests and reflects the configured origins. An
// unknown origin is left without CORS headers so the browser blocks it.
func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	wildcard := false
	for _, o := range origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "*" {
			wildcard = true
		}
		if o != "" {
			allowed[o] = struct{}{}
		}
	}
	methods := strings.Join(CORSAllowedMethods, ",")
	headers := strings.Join(CORSAllowedHeaders, ",")
	exposed := strings.Join(CORSExposedHeaders, ",")

	return func(c *gin.Context) {
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if origin != "" {
			_, ok := allowed[origin]
			if wildcard || ok {
				if wildcard {
					c.Header("Access-Control-Allow-Origin", "*")
				} else {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin")
				}
				c.Header("Access-Control-Allow-Methods", methods)
				c.Header("Access-Control-Allow-Headers", headers)
				c.Header("Access-Control-Expose-Headers", exposed)
				c.Header("Access-Control-Max-Age", strconv.Itoa(CORSMaxAge))
				if origin != "null" {
					c.Header("Access-Control-Allow-Credentials", "true")
				}
			}
		}
		if c.Request.Method == http.MethodOptions {
			if origin == "" || wildcard {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			if _, ok := allowed[origin]; !ok {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AllowedOrigin reports whether origin is part of the configured list. The
// WebSocket upgrader uses it for its CheckOrigin hook.
func AllowedOrigin(origins []string, origin string) bool {
	if origin == "" {
		return true
	}
	origin = strings.TrimRight(origin, "/")
	for _, o := range origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}