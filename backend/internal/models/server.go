package models

import (
	"time"

	"gorm.io/gorm"
)

// Server is a guild like workspace owned by a single user.
type Server struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Description string    `gorm:"size:512;not null;default:''" json:"description"`
	IconURL     *string   `json:"iconUrl"`
	BannerURL   *string   `json:"bannerUrl"`
	OwnerID     string    `gorm:"type:uuid;not null;index" json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	Owner    User      `gorm:"foreignKey:OwnerID;references:ID" json:"-"`
	Members  []Member  `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE" json:"-"`
	Channels []Channel `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE" json:"-"`
	Roles    []Role    `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Server) TableName() string { return "servers" }

// BeforeCreate fills the generated columns before the first insert.
func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = newID()
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	return nil
}