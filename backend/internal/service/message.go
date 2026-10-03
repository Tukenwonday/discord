package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/permissions"
	"github.com/cordis/backend/internal/storage"
	"gorm.io/gorm"
)

// Server to client event names from section 6 of the contract.
const (
	EventMessageCreated = "message.created"
	EventMessageUpdated = "message.updated"
	EventMessageDeleted = "message.deleted"
	EventTypingStarted  = "typing.started"
	EventTypingStopped  = "typing.stopped"
	EventPresenceUpdate = "presence.updated"
)

// MessageCreatedPayload is the message.created event body.
type MessageCreatedPayload struct {
	Message   *models.Message `json:"message"`
	ChannelID string          `json:"channelId"`
	DMID      *string         `json:"dmId"`
}

// MessageUpdatedPayload is the message.updated event body.
type MessageUpdatedPayload struct {
	Message   *models.Message `json:"message"`
	ChannelID string          `json:"channelId"`
}

// MessageDeletedPayload is the message.deleted event body.
type MessageDeletedPayload struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channelId"`
	DeletedAt time.Time `json:"deletedAt"`
}

// CreateMessageInput is the validated message creation payload.
type CreateMessageInput struct {
	Content     string
	ReplyToID   *string
	Attachments []models.AttachmentInput
}

// Message implements message reads, creation, edits and deletions and emits
// the message WebSocket events.
type Message struct {
	deps Deps
}

// NewMessage builds the message service.
func NewMessage(deps Deps) *Message { return &Message{deps: deps} }

// List returns a page of channel messages, oldest to newest.
func (s *Message) List(channelID, userID string, opts PageOptions) (*Paginated[models.Message], error) {
	if _, err := s.channelAccess(channelID, userID); err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, s.serverIDOf(channelID), userID, permissions.ReadMessageHistory); err != nil {
		return nil, err
	}
	return s.page(channelID, opts)
}

// Create stores a message in a server channel and broadcasts message.created.
func (s *Message) Create(channelID, userID string, in CreateMessageInput) (*models.Message, error) {
	channel, err := s.channelAccess(channelID, userID)
	if err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, channel.ServerID, userID, permissions.SendMessages); err != nil {
		return nil, err
	}
	msg, err := s.store(channelID, userID, in, true)
	if err != nil {
		return nil, err
	}
	s.deps.Pub.PublishToChannel(channelID, EventMessageCreated, MessageCreatedPayload{
		Message:   msg,
		ChannelID: channelID,
		DMID:      nil,
	})
	return msg, nil
}

// CreateDM stores a message in a direct message conversation and broadcasts
// message.created with the conversation id in both channelId and dmId.
func (s *Message) CreateDM(dmID, userID string, in CreateMessageInput) (*models.Message, error) {
	if err := RequireDMRecipient(s.deps.DB, dmID, userID); err != nil {
		return nil, err
	}
	msg, err := s.store(dmID, userID, in, false)
	if err != nil {
		return nil, err
	}
	if err := touchDM(s.deps.DB, dmID); err != nil {
		s.deps.Log.Warn().Err(err).Msg("touch dm failed")
	}
	s.deps.Pub.PublishToChannel(dmID, EventMessageCreated, MessageCreatedPayload{
		Message:   msg,
		ChannelID: dmID,
		DMID:      &dmID,
	})
	return msg, nil
}

// Get loads a single message with its author, attachments and reactions.
func (s *Message) Get(messageID string) (*models.Message, error) {
	msg, err := s.load(messageID)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// ListDM returns a page of direct message messages, oldest to newest.
func (s *Message) ListDM(dmID, userID string, opts PageOptions) (*Paginated[models.Message], error) {
	if err := RequireDMRecipient(s.deps.DB, dmID, userID); err != nil {
		return nil, err
	}
	return s.page(dmID, opts)
}

// Update edits the content of a message. Only the author may edit, and the
// editedAt timestamp is always refreshed.
func (s *Message) Update(messageID, userID, content string) (*models.Message, error) {
	msg, err := s.load(messageID)
	if err != nil {
		return nil, err
	}
	if msg.AuthorID != userID {
		return nil, httpx.NewForbidden("only the author can edit this message")
	}

	trimmed := strings.TrimSpace(content)
	switch {
	case trimmed == "":
		return nil, httpx.NewValidation("validation failed").
			WithFields(map[string]string{"content": "required"})
	case utf8.RuneCountInString(trimmed) > s.deps.Config.MaxMessageLen:
		return nil, httpx.NewValidation("validation failed").
			WithFields(map[string]string{
				"content": fmt.Sprintf("must be at most %d characters", s.deps.Config.MaxMessageLen),
			})
	}

	now := time.Now().UTC()
	if err := s.deps.DB.Model(&models.Message{}).Where("id = ?", messageID).
		Updates(map[string]any{"content": trimmed, "edited_at": now}).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("update message: %w", err))
	}

	updated, err := s.load(messageID)
	if err != nil {
		return nil, err
	}
	s.deps.Pub.PublishToChannel(updated.ChannelID, EventMessageUpdated, MessageUpdatedPayload{
		Message:   updated,
		ChannelID: updated.ChannelID,
	})
	return updated, nil
}

// Delete removes a message. The author may always delete, otherwise the
// ManageMessages permission in the owning server is required.
func (s *Message) Delete(messageID, userID string) error {
	msg, err := s.load(messageID)
	if err != nil {
		return err
	}
	if msg.AuthorID != userID {
		serverID := s.serverIDOf(msg.ChannelID)
		if serverID == "" {
			return httpx.NewForbidden("only the author can delete this message")
		}
		if err := RequirePermission(s.deps.DB, serverID, userID, permissions.ManageMessages); err != nil {
			return err
		}
	}
	if err := s.deps.DB.Where("id = ?", messageID).Delete(&models.Message{}).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("delete message: %w", err))
	}
	s.deps.Pub.PublishToChannel(msg.ChannelID, EventMessageDeleted, MessageDeletedPayload{
		ID:        messageID,
		ChannelID: msg.ChannelID,
		DeletedAt: time.Now().UTC(),
	})
	return nil
}
// channelAccess loads a channel and verifies the user is a member of its
// server. Direct message destinations are not accepted here.
func (s *Message) channelAccess(channelID, userID string) (*models.Channel, error) {
	var channel models.Channel
	err := s.deps.DB.Where("id = ?", channelID).Take(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("channel not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load channel: %w", err))
	}
	if _, err := requireMember(s.deps.DB, channel.ServerID, userID); err != nil {
		return nil, err
	}
	return &channel, nil
}

// serverIDOf resolves the server of a channel, returning an empty string when
// the id belongs to a direct message conversation.
func (s *Message) serverIDOf(channelID string) string {
	var channel models.Channel
	if err := s.deps.DB.Select("id, server_id").Where("id = ?", channelID).Take(&channel).Error; err != nil {
		return ""
	}
	return channel.ServerID
}

// page runs the shared cursor pagination for any destination id.
func (s *Message) page(destinationID string, opts PageOptions) (*Paginated[models.Message], error) {
	opts = opts.Normalize()
	q := s.deps.DB.Model(&models.Message{}).
		Preload("Author").
		Preload("Attachments").
		Preload("Reactions").
		Preload("ReplyTo").
		Preload("ReplyTo.Author").
		Where("channel_id = ?", destinationID)
	rows, hasMore, next, err := paginate[models.Message](s.deps.DB, opts, "messages", q)
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	if rows == nil {
		rows = []models.Message{}
	}
	return &Paginated[models.Message]{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// load reads one message with every association the contract payload needs.
func (s *Message) load(messageID string) (*models.Message, error) {
	var msg models.Message
	err := s.deps.DB.
		Preload("Author").
		Preload("Attachments").
		Preload("Reactions").
		Preload("ReplyTo").
		Preload("ReplyTo.Author").
		Where("id = ?", messageID).Take(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("message not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load message: %w", err))
	}
	return &msg, nil
}

// store validates and persists a message plus its attachments. Direct message
// destinations skip the server context, which is why isChannel exists.
func (s *Message) store(destinationID, authorID string, in CreateMessageInput, isChannel bool) (*models.Message, error) {
	if !isChannel {
		if err := RequireDMRecipient(s.deps.DB, destinationID, authorID); err != nil {
			return nil, err
		}
	}
	content := strings.TrimSpace(in.Content)
	fields := map[string]string{}
	if utf8.RuneCountInString(content) > s.deps.Config.MaxMessageLen {
		fields["content"] = fmt.Sprintf("must be at most %d characters", s.deps.Config.MaxMessageLen)
	}
	if content == "" && len(in.Attachments) == 0 {
		fields["content"] = "required"
	}
	if len(in.Attachments) > 10 {
		fields["attachments"] = "must contain at most 10 items"
	}
	for i, a := range in.Attachments {
		key := fmt.Sprintf("attachments[%d]", i)
		switch {
		case strings.TrimSpace(a.URL) == "":
			fields[key+".url"] = "required"
		case strings.TrimSpace(a.Filename) == "":
			fields[key+".filename"] = "required"
		case a.Size < 0:
			fields[key+".size"] = "must be zero or greater"
		case a.ContentType != "" && !storage.AllowedContentTypes[strings.ToLower(a.ContentType)]:
			fields[key+".contentType"] = "content type is not allowed"
		}
	}
	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}

	if in.ReplyToID != nil && strings.TrimSpace(*in.ReplyToID) != "" {
		var parent models.Message
		err := s.deps.DB.Select("id, channel_id").Where("id = ?", *in.ReplyToID).Take(&parent).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, httpx.NewField("replyToId", "reply target not found")
			}
			return nil, httpx.NewInternalFromError(fmt.Errorf("load reply target: %w", err))
		}
		if parent.ChannelID != destinationID {
			return nil, httpx.NewField("replyToId", "reply target belongs to another destination")
		}
	}

	msgType := models.MessageTypeText
	if len(in.Attachments) > 0 {
		msgType = models.MessageTypeFile
		for _, a := range in.Attachments {
			if strings.HasPrefix(strings.ToLower(a.ContentType), "image/") {
				msgType = models.MessageTypeImage
				break
			}
		}
	}

	msg := &models.Message{
		ChannelID: destinationID,
		AuthorID:  authorID,
		Content:   content,
		Type:      msgType,
		ReplyToID: optionalID(in.ReplyToID),
	}

	err := s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return fmt.Errorf("create message: %w", err)
		}
		if len(in.Attachments) == 0 {
			return nil
		}
		rows := make([]models.Attachment, 0, len(in.Attachments))
		for _, a := range in.Attachments {
			rows = append(rows, models.Attachment{
				MessageID:   msg.ID,
				URL:         strings.TrimSpace(a.URL),
				Filename:    storage.SafeFilename(a.Filename),
				Size:        a.Size,
				ContentType: strings.TrimSpace(a.ContentType),
				Width:       a.Width,
				Height:      a.Height,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("create attachments: %w", err)
		}
		msg.Attachments = rows
		return nil
	})
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return nil, apiErr
		}
		return nil, httpx.NewInternalFromError(err)
	}
	return s.load(msg.ID)
}