package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Channels serves the channel endpoints of section 5.
type Channels struct {
	channels *service.Channel
}

// NewChannels builds the channels handler.
func NewChannels(deps service.Deps) *Channels {
	return &Channels{channels: service.NewChannel(deps)}
}

// createChannelRequest is the body of POST /api/servers/:id/channels.
type createChannelRequest struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Topic    string  `json:"topic"`
	ParentID *string `json:"parentId"`
}

// updateChannelRequest is the body of PATCH /api/channels/:id.
type updateChannelRequest struct {
	Name     *string `json:"name"`
	Topic    *string `json:"topic"`
	Position *int    `json:"position"`
	ParentID *string `json:"parentId"`
}

// Create adds a channel to a server and answers 201.
func (h *Channels) Create(c *gin.Context) {
	var req createChannelRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	channel, err := h.channels.Create(c.Param("id"), middleware.UserID(c), service.CreateChannelInput{
		Name:     req.Name,
		Type:     req.Type,
		Topic:    req.Topic,
		ParentID: req.ParentID,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, channel)
}

// Update applies a partial channel update.
func (h *Channels) Update(c *gin.Context) {
	var req updateChannelRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	channel, err := h.channels.Update(c.Param("id"), middleware.UserID(c), service.UpdateChannelInput{
		Name:     req.Name,
		Topic:    req.Topic,
		Position: req.Position,
		ParentID: req.ParentID,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, channel)
}

// Delete removes a channel.
func (h *Channels) Delete(c *gin.Context) {
	if err := h.channels.Delete(c.Param("id"), middleware.UserID(c)); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}