package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Servers serves the server lifecycle endpoints of section 5.
type Servers struct {
	servers *service.Server
}

// NewServers builds the servers handler.
func NewServers(deps service.Deps) *Servers { return &Servers{servers: service.NewServer(deps)} }

// createServerRequest is the body of POST /api/servers.
type createServerRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// updateServerRequest is the body of PATCH /api/servers/:id.
type updateServerRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IconURL     *string `json:"iconUrl"`
	BannerURL   *string `json:"bannerUrl"`
}

// List returns the paginated servers of the caller.
func (h *Servers) List(c *gin.Context) {
	page, err := h.servers.List(middleware.UserID(c), pageOptions(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, page)
}

// Create provisions a server and answers 201 with its detail payload.
func (h *Servers) Create(c *gin.Context) {
	var req createServerRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	detail, err := h.servers.Create(middleware.UserID(c), service.CreateServerInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, detail)
}

// Get returns the composite server detail for a member.
func (h *Servers) Get(c *gin.Context) {
	detail, err := h.servers.Get(c.Param("id"), middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, detail)
}

// Update applies a partial server update.
func (h *Servers) Update(c *gin.Context) {
	var req updateServerRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	server, err := h.servers.Update(c.Param("id"), middleware.UserID(c), service.UpdateServerInput{
		Name:        req.Name,
		Description: req.Description,
		IconURL:     req.IconURL,
		BannerURL:   req.BannerURL,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, server)
}

// Delete removes a server; only the owner may do so.
func (h *Servers) Delete(c *gin.Context) {
	if err := h.servers.Delete(c.Param("id"), middleware.UserID(c)); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}