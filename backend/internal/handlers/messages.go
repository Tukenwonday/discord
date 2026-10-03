package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Messages serves the channel message endpoints of section 5.
type Messages struct {
	messages *service.Message
}

// NewMessages builds the messages handler.
func NewMessages(deps service.Deps) *Messages {
	return &Messages{messages: service.NewMessage(deps)}
}

// createMessageRequest is the body of POST /api/channels/:id/messages.
type createMessageRequest struct {
	Content     string                   `json:"content"`
	ReplyToID   *string                  `json:"replyToId"`
	Attachments []models.AttachmentInput `json:"attachments"`
}

// updateMessageRequest is the body of PATCH /api/messages/:id.
type updateMessageRequest struct {
	Content string `json:"content"`
}

// List returns a page of channel messages, oldest to newest.
func (h *Messages) List(c *gin.Context) {
	page, err := h.messages.List(c.Param("id"), middleware.UserID(c), pageOptions(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, page)
}

// Create stores a message in a server channel and answers 201.
func (h *Messages) Create(c *gin.Context) {
	var req createMessageRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	message, err := h.messages.Create(c.Param("id"), middleware.UserID(c), service.CreateMessageInput{
		Content:     req.Content,
		ReplyToID:   req.ReplyToID,
		Attachments: req.Attachments,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusCreated, message)
}

// Update edits a message authored by the caller.
func (h *Messages) Update(c *gin.Context) {
	var req updateMessageRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	message, err := h.messages.Update(c.Param("id"), middleware.UserID(c), req.Content)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, message)
}

// Delete removes a message the caller authored or can moderate.
func (h *Messages) Delete(c *gin.Context) {
	if err := h.messages.Delete(c.Param("id"), middleware.UserID(c)); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}