package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/permissions"
	"gorm.io/gorm"
)

// Server to client reaction events.
const (
	EventReactionAdded   = "reaction.added"
	EventReactionRemoved = "reaction.removed"
)

// ReactionAddedPayload is the body of reaction.added.
type ReactionAddedPayload struct {
	Reaction  *models.Reaction `json:"reaction"`
	MessageID string           `json:"messageId"`
	ChannelID string           `json:"channelId"`
}

// ReactionRemovedPayload is the body of reaction.removed.
type ReactionRemovedPayload struct {
	ID        string `json:"id"`
	MessageID string `json:"messageId"`
	ChannelID string `json:"channelId"`
	Emoji     string `json:"emoji"`
	UserID    string `json:"userId"`
}
// Toggle adds a reaction or removes the caller's existing one, reporting
// whether a row was created so the handler can answer 201 or 204.
func (s *Reaction) Toggle(messageID, userID, emoji string) (*models.Reaction, bool, error) {
	clean, err := validateEmoji(emoji)
	if err != nil {
		return nil, false, err
	}
	msg, err := s.loadMessage(messageID, userID)
	if err != nil {
		return nil, false, err
	}
	if err := s.requireAddPermission(msg, userID); err != nil {
		return nil, false, err
	}

	var existing models.Reaction
	err = s.deps.DB.Where("message_id = ? AND user_id = ? AND emoji = ?", msg.ID, userID, clean).
		Take(&existing).Error
	switch {
	case err == nil:
		if err := s.deps.DB.Where("id = ?", existing.ID).Delete(&models.Reaction{}).Error; err != nil {
			return nil, false, httpx.NewInternalFromError(fmt.Errorf("delete reaction: %w", err))
		}
		s.deps.Pub.PublishToChannel(msg.ChannelID, EventReactionRemoved, ReactionRemovedPayload{
			ID:        existing.ID,
			MessageID: msg.ID,
			ChannelID: msg.ChannelID,
			Emoji:     clean,
			UserID:    userID,
		})
		return nil, false, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return nil, false, httpx.NewInternalFromError(fmt.Errorf("load reaction: %w", err))
	}

	reaction := &models.Reaction{MessageID: msg.ID, UserID: userID, Emoji: clean}
	if err := s.deps.DB.Create(reaction).Error; err != nil {
		return nil, false, httpx.NewInternalFromError(fmt.Errorf("create reaction: %w", err))
	}
	s.deps.Pub.PublishToChannel(msg.ChannelID, EventReactionAdded, ReactionAddedPayload{
		Reaction:  reaction,
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
	})
	return reaction, true, nil
}

// Remove deletes a reaction addressed by emoji alone or by emoji plus userId.
// Without a userId the caller may only delete their own reaction.
func (s *Reaction) Remove(messageID, userID, emoji, targetUserID string) error {
	clean, err := validateEmoji(emoji)
	if err != nil {
		return err
	}
	msg, err := s.loadMessage(messageID, userID)
	if err != nil {
		return err
	}

	target := userID
	if strings.TrimSpace(targetUserID) != "" {
		canManage, err := s.canManageReactions(msg, userID)
		if err != nil {
			return err
		}
		if !canManage {
			return httpx.NewForbidden("you do not have permission to remove other users reactions")
		}
		target = strings.TrimSpace(targetUserID)
	}

	if err := s.deps.DB.Where("message_id = ? AND user_id = ? AND emoji = ?", msg.ID, target, clean).
		Delete(&models.Reaction{}).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("delete reaction: %w", err))
	}
	s.deps.Pub.PublishToChannel(msg.ChannelID, EventReactionRemoved, ReactionRemovedPayload{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		Emoji:     clean,
		UserID:    target,
	})
	return nil
}

// List returns every reaction of a message ordered by creation time.
func (s *Reaction) List(messageID, userID string) ([]models.Reaction, error) {
	msg, err := s.loadMessage(messageID, userID)
	if err != nil {
		return nil, err
	}
	rows := []models.Reaction{}
	if err := s.deps.DB.Where("message_id = ?", msg.ID).
		Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("list reactions: %w", err))
	}
	return rows, nil
}
// loadMessage reads a message and verifies the caller may see it, accepting
// both server channels and direct message conversations.
func (s *Reaction) loadMessage(messageID, userID string) (*models.Message, error) {
	if strings.TrimSpace(messageID) == "" {
		return nil, httpx.NewField("id", "required")
	}
	var msg models.Message
	err := s.deps.DB.Where("id = ?", messageID).Take(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("message not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load message: %w", err))
	}
	if err := requireDestination(s.deps.DB, msg.ChannelID, userID); err != nil {
		return nil, err
	}
	return &msg, nil
}

// requireAddPermission enforces AddReactions for server channels; direct
// message participants may always react.
func (s *Reaction) requireAddPermission(msg *models.Message, userID string) error {
	serverID, err := serverOfDestination(s.deps.DB, msg.ChannelID)
	if err != nil {
		return err
	}
	if serverID == "" {
		return nil
	}
	return RequirePermission(s.deps.DB, serverID, userID, permissions.AddReactions)
}

// canManageReactions reports whether the caller may delete another user's
// reaction, which requires ManageMessages in the owning server.
func (s *Reaction) canManageReactions(msg *models.Message, userID string) (bool, error) {
	serverID, err := serverOfDestination(s.deps.DB, msg.ChannelID)
	if err != nil {
		return false, err
	}
	if serverID == "" {
		return true, nil
	}
	mask, isOwner, err := MemberPermissions(s.deps.DB, serverID, userID)
	if err != nil {
		return false, err
	}
	return permissions.Can(mask, permissions.ManageMessages, isOwner), nil
}

// validateEmoji enforces the 1 to 16 rune rule and rejects invalid UTF-8.
func validateEmoji(emoji string) (string, error) {
	if !utf8.ValidString(emoji) {
		return "", httpx.NewValidation("validation failed").
			WithFields(map[string]string{"emoji": "must be valid UTF-8"})
	}
	trimmed := strings.TrimSpace(emoji)
	n := utf8.RuneCountInString(trimmed)
	if n == 0 || n > 16 {
		return "", httpx.NewValidation("validation failed").
			WithFields(map[string]string{"emoji": "must be between 1 and 16 characters"})
	}
	return trimmed, nil
}

// Reaction implements the reaction toggle, removal and listing operations.
type Reaction struct {
	deps Deps
}

// NewReaction builds the reaction service.
func NewReaction(deps Deps) *Reaction { return &Reaction{deps: deps} }