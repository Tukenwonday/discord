package models

import (
	"time"

	"gorm.io/gorm"
)

// Attachment links an uploaded file to a message.
type Attachment struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID   string    `gorm:"type:uuid;index" json:"-"`
	URL         string    `gorm:"size:1024;not null" json:"url"`
	Filename    string    `gorm:"size:255;not null" json:"filename"`
	Size        int64     `gorm:"not null" json:"size"`
	ContentType string    `gorm:"size:128;not null" json:"contentType"`
	Width       *int      `json:"width"`
	Height      *int      `json:"height"`
	CreatedAt   time.Time `json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Attachment) TableName() string { return "attachments" }

// BeforeCreate fills the generated columns before the first insert.
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = newID()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return nil
}

// AttachmentInput is the payload a client sends when attaching an already
// uploaded file to a message.
type AttachmentInput struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
}