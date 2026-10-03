// Package models holds the GORM entities and their JSON representations as
// defined in section 4 of the Cordis contract.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Presence status values.
const (
	StatusOnline    = "online"
	StatusIdle      = "idle"
	StatusDND       = "dnd"
	StatusOffline   = "offline"
	StatusInvisible = "invisible"
)

// ValidStatus reports whether s is a status the backend accepts.
func ValidStatus(s string) bool {
	switch s {
	case StatusOnline, StatusIdle, StatusDND, StatusOffline, StatusInvisible:
		return true
	default:
		return false
	}
}

// ChannelType values.
const (
	ChannelTypeText     = "text"
	ChannelTypeVoice    = "voice"
	ChannelTypeVideo    = "video"
	ChannelTypeCategory = "category"
)

// ValidChannelType reports whether s is a supported channel type.
func ValidChannelType(s string) bool {
	switch s {
	case ChannelTypeText, ChannelTypeVoice, ChannelTypeVideo, ChannelTypeCategory:
		return true
	default:
		return false
	}
}

// MessageType values.
const (
	MessageTypeText   = "text"
	MessageTypeImage  = "image"
	MessageTypeFile   = "file"
	MessageTypeSystem = "system"
)

// FriendStatus values.
const (
	FriendPending  = "pending"
	FriendAccepted = "accepted"
	FriendBlocked  = "blocked"
)

// newID returns a fresh UUIDv4 primary key.
func newID() string { return uuid.NewString() }

// Base carries the columns every entity shares.
type Base struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate assigns the identifier and creation timestamp.
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = newID()
	}
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = now
	}
	return nil
}