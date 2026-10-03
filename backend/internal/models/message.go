package models

import (
	"time"

	"gorm.io/gorm"
)

// Message is a single chat entry. ChannelID points either at a server channel
// or at a direct message conversation, which keeps one table for both.
type Message struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID string     `gorm:"type:uuid;not null;index" json:"channelId"`
	AuthorID  string     `gorm:"type:uuid;not null;index" json:"-"`
	Content   string     `gorm:"type:text;not null;default:''" json:"content"`
	Type      string     `gorm:"size:16;not null;default:'text'" json:"type"`
	ReplyToID *string    `gorm:"type:uuid;index" json:"-"`
	EditedAt  *time.Time `json:"editedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"-"`

	Author      User         `gorm:"foreignKey:AuthorID;references:ID" json:"author"`
	ReplyTo     *Message     `gorm:"foreignKey:ReplyToID;references:ID" json:"replyTo"`
	Attachments []Attachment `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"attachments"`
	Reactions   []Reaction   `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"reactions"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Message) TableName() string { return "messages" }

// BeforeCreate fills the generated columns before the first insert.
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = newID()
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	if m.Type == "" {
		m.Type = MessageTypeText
	}
	return nil
}

// ChannelIDJSON exposes the channel id under the contract key.
func (m *Message) ChannelIDJSON() string { return m.ChannelID }