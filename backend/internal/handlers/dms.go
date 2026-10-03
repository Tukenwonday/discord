package handlers

import (
	"net/http"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// DMs serves the direct message endpoints of section 5.
type DMs struct {
	dms      *service.DM
	messages *service.Message
}

// NewDMs builds the direct message handler.
func NewDMs(deps service.Deps) *DMs {
	return &DMs{dms: service.NewDM(deps), messages: service.NewMessage(deps)}
}

// openDMRequest is the body of POST /api/dms.
type openDMRequest struct {
	RecipientID string `json:"recipientId"`
}

// List returns the conversations of the caller.
func (h *DMs) List(c *gin.Context) {
	page, err := h.dms.List(middleware.UserID(c), pageOptions(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, page)
}

// Open returns the existing conversation with a user or creates it. The endpoint
// is idempotent, so it answers 200 rather than 201.
func (h *DMs) Open(c *gin.Context) {
	var req openDMRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	dm, err := h.dms.Open(middleware.UserID(c), req.RecipientID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, dm)
}

// Get returns one conversation with its recipients and last message.
func (h *DMs) Get(c *gin.Context) {
	dm, err := h.dms.Get(c.Param("id"), middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, dm)
}

// ListMessages returns a page of conversation messages, oldest to newest.
func (h *DMs) ListMessages(c *gin.Context) {
	page, err := h.messages.ListDM(c.Param("id"), middleware.UserID(c), pageOptions(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, page)
}

// CreateMessage stores a message in a conversation and answers 201.
func (h *DMs) CreateMessage(c *gin.Context) {
	var req createMessageRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}
	message, err := h.messages.CreateDM(c.Param("id"), middleware.UserID(c), service.CreateMessageInput{
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

// Leave removes the caller from a conversation.
func (h *DMs) Leave(c *gin.Context) {
	if err := h.dms.Leave(c.Param("id"), middleware.UserID(c)); err != nil {
		httpx.RespondError(c, err)
		return
	}
	middleware.NoContent(c)
}

// Recipients lists the participants of a conversation as a bare array.
func (h *DMs) Recipients(c *gin.Context) {
	users, err := h.dms.Recipients(c.Param("id"), middleware.UserID(c))
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	httpx.RespondJSON(c, http.StatusOK, users)
}