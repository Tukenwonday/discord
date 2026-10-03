package models

import (
	"time"

	"gorm.io/gorm"
)

// Reaction is an emoji placed on a message by a user. The unique index on
// (message_id, user_id, emoji) makes the toggle operation atomic.
type Reaction struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID string    `gorm:"type:uuid;not null;uniqueIndex:idx_reaction_unique" json:"messageId"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_reaction_unique;index" json:"userId"`
	Emoji     string    `gorm:"size:64;not null;uniqueIndex:idx_reaction_unique" json:"emoji"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Reaction) TableName() string { return "reactions" }

// BeforeCreate fills the generated columns before the first insert.
func (r *Reaction) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = newID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	return nil
}