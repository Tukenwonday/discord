package models

import (
	"time"

	"gorm.io/gorm"
)

// Role groups a permission bitmask under a name inside a server.
type Role struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	ServerID    string    `gorm:"type:uuid;not null;index" json:"serverId"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Color       string    `gorm:"size:9;not null;default:'#5865f2'" json:"color"`
	Permissions int64     `gorm:"not null;default:0" json:"permissions"`
	Hoist       bool      `gorm:"not null;default:false" json:"hoist"`
	Position    int       `gorm:"not null;default:0" json:"position"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Role) TableName() string { return "roles" }

// BeforeCreate fills the generated columns before the first insert.
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = newID()
	}
	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}
	if r.Color == "" {
		r.Color = "#5865f2"
	}
	return nil
}

// EveryoneRoleName is the role every member of a server receives.
const EveryoneRoleName = "@everyone"