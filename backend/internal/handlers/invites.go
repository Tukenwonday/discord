package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Invites serves the invite endpoints of section 5.
type Invites struct {
	invites *service.Invite
}

// NewInvites builds the invites handler.
func NewInvites(deps service.Deps) *Invites {
	return &Invites{invites: service.NewInvite(deps)}
}

// createInviteRequest is the body of POST /api/servers/:id/invites.
type createInviteRequest struct {
	ChannelID      *string `json:"channelId"`
	MaxUses        int     `json:"maxUses"`
	ExpiresInHours int     `json:"expiresInHours"`
}

// List returns the invites of a server, which only the owner may read.
func (h *Invites) List(c *gin.Context) {
	rows, err := h.invites.List(c.Param("id"), middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, rows)
}

// Create mints an invite and answers 201.
func (h *Invites) Create(c *gin.Context) {
	var req createInviteRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	invite, err := h.invites.Create(c.Param("id"), middleware.UserID(c), service.CreateInviteInput{
		ChannelID:      req.ChannelID,
		MaxUses:        req.MaxUses,
		ExpiresInHours: req.ExpiresInHours,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, invite)
}

// Get previews an invite. The route is public so a link can be opened before
// signing in.
func (h *Invites) Get(c *gin.Context) {
	invite, err := h.invites.Get(c.Param("code"))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, invite)
}

// Join adds the caller to the server behind an invite.
func (h *Invites) Join(c *gin.Context) {
	detail, err := h.invites.Join(c.Param("code"), middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, detail)
}

// Revoke deletes an invite; only the server owner may do so.
func (h *Invites) Revoke(c *gin.Context) {
	if err := h.invites.Revoke(c.Param("code"), middleware.UserID(c)); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}