package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Reactions serves the reaction endpoints of section 5.
type Reactions struct {
	reactions *service.Reaction
}

// NewReactions builds the reactions handler.
func NewReactions(deps service.Deps) *Reactions {
	return &Reactions{reactions: service.NewReaction(deps)}
}

// toggleRequest is the body of POST /api/messages/:id/reactions.
type toggleRequest struct {
	Emoji string `json:"emoji"`
}

// Toggle adds a reaction and answers 201, or removes the caller's existing one
// and answers 204 because the endpoint toggles.
func (h *Reactions) Toggle(c *gin.Context) {
	var req toggleRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	reaction, created, err := h.reactions.Toggle(c.Param("id"), middleware.UserID(c), req.Emoji)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	if !created {
		middleware.NoContent(c)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, reaction)
}

// Remove deletes a reaction addressed by emoji, optionally targeting another
// user when the caller may moderate messages.
func (h *Reactions) Remove(c *gin.Context) {
	err := h.reactions.Remove(
		c.Param("id"),
		middleware.UserID(c),
		c.Query("emoji"),
		c.Query("userId"),
	)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}

// List returns every reaction of a message as a bare array.
func (h *Reactions) List(c *gin.Context) {
	rows, err := h.reactions.List(c.Param("id"), middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, rows)
}