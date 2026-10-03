package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/storage"
	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
)

// Upload stores an attachment through the configured driver.
type Upload struct {
	driver  storage.Driver
	maxMB   int
	maxByte int64
}

// NewUpload builds the upload handler.
func NewUpload(driver storage.Driver, maxMB int) *Upload {
	return &Upload{driver: driver, maxMB: maxMB, maxByte: int64(maxMB) * 1024 * 1024}
}

// multipartContentType is the media type a multipart request must declare.
const multipartContentType = "multipart/form-data"

// File accepts a multipart upload and answers 201 with the attachment. A non
// multipart request is refused with 415 and an oversized one with 413.
func (h *Upload) File(c *gin.Context) {
	contentType := c.ContentType()
	if !strings.HasPrefix(strings.ToLower(contentType), multipartContentType) {
		httpx.RespondError(c, httpx.NewUnsupportedMediaType("content type must be multipart/form-data"))
		return
	}

	// The ceiling is enforced while parsing so an oversized body never reaches
	// the disk or the object store.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxByte+(1<<20))

	header, err := c.FormFile("file")
	if err != nil {
		httpx.RespondError(c, httpx.NewField("file", "is required"))
		return
	}
	if header.Size <= 0 {
		httpx.RespondError(c, httpx.NewField("file", "file is empty"))
		return
	}
	if header.Size > h.maxByte {
		httpx.RespondError(c, httpx.NewPayloadTooLarge(
			"file exceeds the maximum upload size of "+strconv.Itoa(h.maxMB)+" MB"))
		return
	}

	detected := header.Header.Get("Content-Type")
	if err := storage.ValidateContentType(detected); err != nil {
		httpx.RespondError(c, storage.MapError(err))
		return
	}

	attachment, err := h.driver.Save(c.Request.Context(), header, detected)
	if err != nil {
		httpx.RespondError(c, storage.MapError(err))
		return
	}

	// Optional pixel dimensions let the client lay out an image without
	// decoding it again.
	if width, ok := optionalInt(c.PostForm("width")); ok {
		attachment.Width = &width
	}
	if height, ok := optionalInt(c.PostForm("height")); ok {
		attachment.Height = &height
	}

	httpx.RespondJSON(c, http.StatusCreated, sanitizeAttachment(attachment))
}

// optionalInt parses an optional numeric form field.
func optionalInt(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// sanitizeAttachment returns the attachment with an identifier assigned so the
// client can reference it from a later message.
func sanitizeAttachment(a models.Attachment) models.Attachment {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	a.Filename = storage.SafeFilename(a.Filename)
	return a
}