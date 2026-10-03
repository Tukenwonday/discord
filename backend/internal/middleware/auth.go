package middleware

import (
	"strings"

	"github.com/cordis/backend/internal/auth"
	"github.com/cordis/backend/internal/httpx"
	"github.com/gin-gonic/gin"
)

// ContextUserID is the gin context key holding the authenticated user id.
const ContextUserID = "user_id"

// ContextClaims is the gin context key holding the decoded access claims.
const ContextClaims = "access_claims"

// BearerToken extracts the token from the Authorization header, falling back
// to the token query parameter used by the WebSocket handshake.
func BearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(c.Query("token"))
}

// AuthRequired validates the bearer access token and rejects the request with
// the 401 envelope when it is missing or invalid.
func AuthRequired(tokens *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := BearerToken(c)
		if token == "" {
			httpx.RespondError(c, httpx.NewUnauthorized("authentication required"))
			return
		}
		claims, err := tokens.ParseAccess(token)
		if err != nil {
			httpx.RespondError(c, httpx.NewUnauthorized("invalid or expired token"))
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextClaims, claims)
		c.Next()
	}
}

// UserID returns the authenticated user id, empty when the route is public.
func UserID(c *gin.Context) string {
	if v, ok := c.Get(ContextUserID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Claims returns the decoded access claims for the current request.
func Claims(c *gin.Context) *auth.AccessClaims {
	if v, ok := c.Get(ContextClaims); ok {
		if cl, ok := v.(*auth.AccessClaims); ok {
			return cl
		}
	}
	return nil
}