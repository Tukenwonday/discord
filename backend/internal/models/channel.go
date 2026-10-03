package models

import (
	"time"

	"gorm.io/gorm"
)

// Channel is a text, voice, video or category destination inside a server.
type Channel struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	ServerID  string    `gorm:"type:uuid;not null;index" json:"serverId"`
	ParentID  *string   `gorm:"type:uuid;index" json:"parentId"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Topic     string    `gorm:"size:256;not null;default:''" json:"topic"`
	Type      string    `gorm:"size:16;not null;default:'text'" json:"type"`
	Position  int       `gorm:"not null;default:0" json:"position"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Messages []Message `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Channel) TableName() string { return "channels" }

// BeforeCreate fills the generated columns before the first insert.
func (c *Channel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = newID()
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	if c.Type == "" {
		c.Type = ChannelTypeText
	}
	return nil
}

// IsVoiceBacked reports whether the channel carries audio or video.
func (c *Channel) IsVoiceBacked() bool {
	return c.Type == ChannelTypeVoice || c.Type == ChannelTypeVideo
}