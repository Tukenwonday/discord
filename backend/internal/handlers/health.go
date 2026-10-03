package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/cordis/backend/internal/cache"
	"github.com/cordis/backend/internal/database"
	"github.com/cordis/backend/internal/httpx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Health serves the liveness and readiness probes of section 5.
type Health struct {
	startedAt time.Time
	db        *gorm.DB
	cache     *cache.Client
	version   string
}

// healthzResponse is the body of GET /healthz.
type healthzResponse struct {
	Status        string `json:"status"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
	Version       string `json:"version"`
}

// readyzResponse is the body of GET /readyz.
type readyzResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// NewHealth builds the probe handler, stamping the process start time.
func NewHealth(db *gorm.DB, rdb *cache.Client, version string) *Health {
	return &Health{startedAt: time.Now(), db: db, cache: rdb, version: version}
}

// Live answers 200 while the process is up.
func (h *Health) Live(c *gin.Context) {
	httpx.RespondJSON(c, http.StatusOK, healthzResponse{
		Status:        "ok",
		UptimeSeconds: int64(time.Since(h.startedAt).Seconds()),
		Version:       h.version,
	})
}

// Ready answers 200 only when both Postgres and Redis ping successfully and
// otherwise reports the failing dependency with 503.
func (h *Health) Ready(c *gin.Context) {
	checks := map[string]string{}
	healthy := true

	if err := database.Ping(h.db); err != nil {
		checks["postgres"] = "down"
		healthy = false
	} else {
		checks["postgres"] = "ok"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := h.cache.Ping(ctx); err != nil {
		checks["redis"] = "down"
		healthy = false
	} else {
		checks["redis"] = "ok"
	}

	if !healthy {
		httpx.RespondJSON(c, http.StatusServiceUnavailable, readyzResponse{
			Status: "degraded",
			Checks: checks,
		})
		return
	}
	httpx.RespondJSON(c, http.StatusOK, readyzResponse{Status: "ok", Checks: checks})
}