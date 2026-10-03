package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Friends serves the friendship endpoints of section 5.
type Friends struct {
	friends *service.Friend
}

// NewFriends builds the friends handler.
func NewFriends(deps service.Deps) *Friends {
	return &Friends{friends: service.NewFriend(deps)}
}

// requestFriendRequest is the body of POST /api/friends.
type requestFriendRequest struct {
	UserID string `json:"userId"`
}

// acceptFriendRequest is the body of PATCH /api/friends/:id.
type acceptFriendRequest struct {
	Status string `json:"status"`
}

// Lists returns accepted, incoming and outgoing friendships.
func (h *Friends) Lists(c *gin.Context) {
	lists, err := h.friends.Lists(middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, lists)
}

// Request creates a pending friendship. An existing row answers 200 instead of
// 201 so repeating the call is harmless.
func (h *Friends) Request(c *gin.Context) {
	var req requestFriendRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	view, created, err := h.friends.Request(middleware.UserID(c), req.UserID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	if !created {
		httpx.RespondJSON(c, http.StatusOK, view)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, view)
}

// Accept marks an incoming request as accepted and mirrors it.
func (h *Friends) Accept(c *gin.Context) {
	var req acceptFriendRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	if req.Status != "accepted" {
		httpx.RespondValidation(c, map[string]string{"status": "must be accepted"})
		return
	}
	view, err := h.friends.Accept(middleware.UserID(c), c.Param("id"))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, view)
}

// Remove deletes a friendship in both directions.
func (h *Friends) Remove(c *gin.Context) {
	if err := h.friends.Remove(middleware.UserID(c), c.Param("id")); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}