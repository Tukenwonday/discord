package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/permissions"
	"gorm.io/gorm"
)

// channelNameRe is the channel name pattern from section 5 of the contract.
var channelNameRe = regexp.MustCompile(`^[a-z0-9\-_]{1,64}$`)

// CreateChannelInput is the validated channel creation payload.
type CreateChannelInput struct {
	Name     string
	Type     string
	Topic    string
	ParentID *string
}

// UpdateChannelInput is the partial channel update payload.
type UpdateChannelInput struct {
	Name     *string
	Topic    *string
	Position *int
	ParentID *string
}

// Channel implements channel lifecycle operations.
type Channel struct {
	deps Deps
}

// NewChannel builds the channel service.
func NewChannel(deps Deps) *Channel { return &Channel{deps: deps} }

// Create adds a channel to a server, requiring ManageChannels.
func (s *Channel) Create(serverID, userID string, in CreateChannelInput) (*models.Channel, error) {
	if _, err := requireMember(s.deps.DB, serverID, userID); err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, serverID, userID, permissions.ManageChannels); err != nil {
		return nil, err
	}

	fields := map[string]string{}
	name := strings.ToLower(strings.TrimSpace(in.Name))
	switch {
	case name == "":
		fields["name"] = "required"
	case !channelNameRe.MatchString(name):
		fields["name"] = "must match ^[a-z0-9-_]{1,64}$"
	}
	channelType := strings.ToLower(strings.TrimSpace(in.Type))
	if channelType == "" {
		channelType = models.ChannelTypeText
	}
	if !models.ValidChannelType(channelType) {
		fields["type"] = "must be one of text, voice, video, category"
	}
	topic := strings.TrimSpace(in.Topic)
	if utf8.RuneCountInString(topic) > 256 {
		fields["topic"] = "must be at most 256 characters"
	}
	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}

	if in.ParentID != nil && strings.TrimSpace(*in.ParentID) != "" {
		var parent models.Channel
		err := s.deps.DB.Where("id = ? AND server_id = ?", *in.ParentID, serverID).Take(&parent).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, httpx.NewField("parentId", "parent channel not found")
			}
			return nil, httpx.NewInternalFromError(fmt.Errorf("load parent channel: %w", err))
		}
		if parent.Type != models.ChannelTypeCategory {
			return nil, httpx.NewField("parentId", "parent must be a category channel")
		}
	}

	var position int
	if err := s.deps.DB.Model(&models.Channel{}).Where("server_id = ?", serverID).
		Select("COALESCE(MAX(position), -1) + 1").Scan(&position).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("compute position: %w", err))
	}

	channel := &models.Channel{
		ServerID: serverID,
		Name:     name,
		Topic:    topic,
		Type:     channelType,
		Position: position,
		ParentID: optionalID(in.ParentID),
	}
	if err := s.deps.DB.Create(channel).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("create channel: %w", err))
	}
	return channel, nil
}

// Update applies a partial channel update, requiring ManageChannels.
func (s *Channel) Update(channelID, userID string, in UpdateChannelInput) (*models.Channel, error) {
	channel, err := s.Get(channelID)
	if err != nil {
		return nil, err
	}
	if _, err := requireMember(s.deps.DB, channel.ServerID, userID); err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, channel.ServerID, userID, permissions.ManageChannels); err != nil {
		return nil, err
	}

	fields := map[string]string{}
	updates := map[string]any{}
	if in.Name != nil {
		name := strings.ToLower(strings.TrimSpace(*in.Name))
		switch {
		case name == "":
			fields["name"] = "required"
		case !channelNameRe.MatchString(name):
			fields["name"] = "must match ^[a-z0-9-_]{1,64}$"
		default:
			updates["name"] = name
		}
	}
	if in.Topic != nil {
		topic := strings.TrimSpace(*in.Topic)
		if utf8.RuneCountInString(topic) > 256 {
			fields["topic"] = "must be at most 256 characters"
		} else {
			updates["topic"] = topic
		}
	}
	if in.Position != nil {
		if *in.Position < 0 {
			fields["position"] = "must be zero or greater"
		} else {
			updates["position"] = *in.Position
		}
	}
	if in.ParentID != nil {
		if strings.TrimSpace(*in.ParentID) == "" {
			updates["parent_id"] = nil
		} else {
			if *in.ParentID == channelID {
				return nil, httpx.NewField("parentId", "a channel cannot be its own parent")
			}
			var parent models.Channel
			err := s.deps.DB.Where("id = ? AND server_id = ?", *in.ParentID, channel.ServerID).Take(&parent).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, httpx.NewField("parentId", "parent channel not found")
				}
				return nil, httpx.NewInternalFromError(fmt.Errorf("load parent channel: %w", err))
			}
			if parent.Type != models.ChannelTypeCategory {
				return nil, httpx.NewField("parentId", "parent must be a category channel")
			}
			updates["parent_id"] = *in.ParentID
		}
	}
	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}
	if len(updates) == 0 {
		return channel, nil
	}
	if err := s.deps.DB.Model(&models.Channel{}).Where("id = ?", channelID).Updates(updates).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("update channel: %w", err))
	}
	return s.Get(channelID)
}

// Delete removes a channel and its messages, requiring ManageChannels.
func (s *Channel) Delete(channelID, userID string) error {
	channel, err := s.Get(channelID)
	if err != nil {
		return err
	}
	if _, err := requireMember(s.deps.DB, channel.ServerID, userID); err != nil {
		return err
	}
	if err := RequirePermission(s.deps.DB, channel.ServerID, userID, permissions.ManageChannels); err != nil {
		return err
	}
	if err := s.deps.DB.Where("id = ?", channelID).Delete(&models.Channel{}).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("delete channel: %w", err))
	}
	return nil
}

// Get loads a channel by id.
func (s *Channel) Get(channelID string) (*models.Channel, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, httpx.NewField("id", "required")
	}
	var channel models.Channel
	if err := s.deps.DB.Where("id = ?", channelID).Take(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("channel not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load channel: %w", err))
	}
	return &channel, nil
}

// GetForMember loads a channel and verifies ViewChannel for the user.
func (s *Channel) GetForMember(channelID, userID string) (*models.Channel, error) {
	channel, err := s.Get(channelID)
	if err != nil {
		return nil, err
	}
	if _, err := requireMember(s.deps.DB, channel.ServerID, userID); err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, channel.ServerID, userID, permissions.ViewChannel); err != nil {
		return nil, err
	}
	return channel, nil
}

// ListByServer returns the channels of a server ordered by position.
func (s *Channel) ListByServer(serverID, userID string) ([]models.Channel, error) {
	if _, err := requireMember(s.deps.DB, serverID, userID); err != nil {
		return nil, err
	}
	var rows []models.Channel
	if err := s.deps.DB.Where("server_id = ?", serverID).
		Order("position ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load channels: %w", err))
	}
	if rows == nil {
		rows = []models.Channel{}
	}
	return rows, nil
}

// optionalID normalises an empty parent id to a NULL column.
func optionalID(id *string) *string {
	if id == nil || strings.TrimSpace(*id) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*id)
	return &trimmed
}