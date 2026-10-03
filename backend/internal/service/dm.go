package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"gorm.io/gorm"
)

// DM implements direct message conversations. The conversation id doubles as
// the channel id of its messages, so the WebSocket registry needs no mapping.
type DM struct {
	deps Deps
}

// NewDM builds the direct message service.
func NewDM(deps Deps) *DM { return &DM{deps: deps} }

// List returns the conversations of a user, most recently active first.
func (s *DM) List(userID string, opts PageOptions) (*Paginated[models.DirectMessage], error) {
	opts = opts.Normalize()
	ids := []string{}
	if err := s.deps.DB.Model(&models.DMRecipient{}).Where("user_id = ?", userID).
		Order("created_at DESC").Pluck("dm_id", &ids).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load dm ids: %w", err))
	}
	rows := []models.DirectMessage{}
	if len(ids) > 0 {
		if err := s.deps.DB.Where("id IN ?", ids).
			Order("updated_at DESC").Find(&rows).Error; err != nil {
			return nil, httpx.NewInternalFromError(fmt.Errorf("load dms: %w", err))
		}
	}
	rows, hasMore, next, err := s.cut(rows, opts)
	if err != nil {
		return nil, err
	}
	if err := s.hydrate(rows); err != nil {
		return nil, err
	}
	return &Paginated[models.DirectMessage]{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// Open returns the existing conversation between two users or creates it. The
// pair is searched deterministically so a duplicate request is idempotent.
func (s *DM) Open(userID, recipientID string) (*models.DirectMessage, error) {
	if strings.TrimSpace(recipientID) == "" {
		return nil, httpx.NewField("recipientId", "required")
	}
	if recipientID == userID {
		return nil, httpx.NewValidation("validation failed").
			WithFields(map[string]string{"recipientId": "you cannot message yourself"})
	}
	if _, err := s.loadUser(recipientID); err != nil {
		return nil, err
	}

	var existing models.DirectMessage
	err := s.deps.DB.Joins("JOIN dm_recipients ON dm_recipients.dm_id = direct_messages.id").
		Where("dm_recipients.user_id IN ?", []string{userID, recipientID}).
		Group("direct_messages.id").
		Having("COUNT(*) = 2").Take(&existing).Error
	if err == nil {
		return s.hydrateOne(&existing)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load dm: %w", err))
	}

	dm := &models.DirectMessage{}
	err = s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dm).Error; err != nil {
			return fmt.Errorf("create dm: %w", err)
		}
		rows := []models.DMRecipient{
			{DMID: dm.ID, UserID: userID},
			{DMID: dm.ID, UserID: recipientID},
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("create dm recipients: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	return s.hydrateOne(dm)
}

// Get returns one conversation with its recipients and last message.
func (s *DM) Get(dmID, userID string) (*models.DirectMessage, error) {
	if err := RequireDMRecipient(s.deps.DB, dmID, userID); err != nil {
		return nil, err
	}
	var dm models.DirectMessage
	if err := s.deps.DB.Where("id = ?", dmID).Take(&dm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("conversation not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load dm: %w", err))
	}
	return s.hydrateOne(&dm)
}

// Recipients lists the participants of a conversation.
func (s *DM) Recipients(dmID, userID string) ([]models.User, error) {
	if err := RequireDMRecipient(s.deps.DB, dmID, userID); err != nil {
		return nil, err
	}
	users := []models.User{}
	if err := s.deps.DB.Joins("JOIN dm_recipients ON dm_recipients.user_id = users.id").
		Where("dm_recipients.dm_id = ?", dmID).
		Order("users.created_at ASC").Find(&users).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load recipients: %w", err))
	}
	return users, nil
}

// Leave removes the caller from the conversation. The conversation row is
// deleted once nobody is left in it.
func (s *DM) Leave(dmID, userID string) error {
	ok, err := IsDMRecipient(s.deps.DB, dmID, userID)
	if err != nil {
		return httpx.NewInternalFromError(err)
	}
	if !ok {
		return httpx.NewNotFound("conversation not found")
	}
	err = s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("dm_id = ? AND user_id = ?", dmID, userID).
			Delete(&models.DMRecipient{}).Error; err != nil {
			return fmt.Errorf("delete dm recipient: %w", err)
		}
		var remaining int64
		if err := tx.Model(&models.DMRecipient{}).Where("dm_id = ?", dmID).
			Count(&remaining).Error; err != nil {
			return fmt.Errorf("count dm recipients: %w", err)
		}
		if remaining == 0 {
			if err := tx.Where("channel_id = ?", dmID).Delete(&models.Message{}).Error; err != nil {
				return fmt.Errorf("delete dm messages: %w", err)
			}
			if err := tx.Where("id = ?", dmID).Delete(&models.DirectMessage{}).Error; err != nil {
				return fmt.Errorf("delete dm: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return httpx.NewInternalFromError(err)
	}
	return nil
}

// cut applies the cursor window to a conversation list that was already
// ordered by recent activity, since conversations are not ordered by
// created_at the way messages are.
func (s *DM) cut(rows []models.DirectMessage, opts PageOptions) ([]models.DirectMessage, bool, *string, error) {
	opts = opts.Normalize()
	index := make(map[string]int, len(rows))
	for i := range rows {
		index[rows[i].ID] = i
	}
	start := 0
	if opts.BeforeID != "" {
		if i, ok := index[opts.BeforeID]; ok {
			start = i + 1
		}
	}
	if opts.AfterID != "" {
		if i, ok := index[opts.AfterID]; ok {
			// The list is newest first, so everything after the cursor keeps
			// only the entries that come before it in this ordering.
			rows = rows[:i]
		}
	}
	rows = rows[start:]
	hasMore := len(rows) > opts.Limit
	if hasMore {
		rows = rows[:opts.Limit]
	}
	var next *string
	if hasMore && len(rows) > 0 {
		id := rows[len(rows)-1].ID
		next = &id
	}
	return rows, hasMore, next, nil
}

// hydrate fills the recipients and last message of a list of conversations.
func (s *DM) hydrate(rows []models.DirectMessage) error {
	for i := range rows {
		if err := s.loadRecipients(&rows[i]); err != nil {
			return err
		}
		if err := s.loadLastMessage(&rows[i]); err != nil {
			return err
		}
	}
	return nil
}

// hydrateOne fills a single conversation.
func (s *DM) hydrateOne(dm *models.DirectMessage) (*models.DirectMessage, error) {
	if err := s.loadRecipients(dm); err != nil {
		return nil, err
	}
	if err := s.loadLastMessage(dm); err != nil {
		return nil, err
	}
	return dm, nil
}

// loadRecipients resolves the participant list of a conversation.
func (s *DM) loadRecipients(dm *models.DirectMessage) error {
	users := []models.User{}
	if err := s.deps.DB.Joins("JOIN dm_recipients ON dm_recipients.user_id = users.id").
		Where("dm_recipients.dm_id = ?", dm.ID).
		Order("users.created_at ASC").Find(&users).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("load recipients: %w", err))
	}
	dm.Recipients = users
	return nil
}

// loadLastMessage attaches the most recent message of the conversation.
func (s *DM) loadLastMessage(dm *models.DirectMessage) error {
	var msg models.Message
	err := s.deps.DB.Preload("Author").Preload("Attachments").Preload("Reactions").
		Where("channel_id = ?", dm.ID).Order("created_at DESC").Take(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dm.LastMessage = nil
			return nil
		}
		return httpx.NewInternalFromError(fmt.Errorf("load last message: %w", err))
	}
	dm.LastMessage = &msg
	return nil
}

// loadUser reads a participant of a conversation.
func (s *DM) loadUser(id string) (*models.User, error) {
	var user models.User
	if err := s.deps.DB.Where("id = ?", id).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("user not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load user: %w", err))
	}
	return &user, nil
}