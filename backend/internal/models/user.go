package models

import (
	"time"

	"gorm.io/gorm"
)

// User is an account that can authenticate, own servers and send messages.
type User struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string    `gorm:"size:32;uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"size:254;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	DisplayName  string    `gorm:"size:64;not null" json:"displayName"`
	AvatarURL    *string   `json:"avatarUrl"`
	BannerURL    *string   `json:"bannerUrl"`
	Bio          string    `gorm:"size:512;not null;default:''" json:"bio"`
	Status       string    `gorm:"size:16;not null;default:'offline'" json:"status"`
	CustomStatus string    `gorm:"size:128;not null;default:''" json:"customStatus"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`

	Servers []Member `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (User) TableName() string { return "users" }

// BeforeCreate fills the generated columns before the first insert.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = newID()
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	if u.Status == "" {
		u.Status = StatusOffline
	}
	return nil
}

// IsVisibleStatus reports whether the status should be shown to other users.
func (u *User) IsVisibleStatus() bool { return u.Status != StatusOffline }

// PublicFields returns the user with the password hash already excluded by
// the json tags, which keeps the single representation authoritative.
func (u *User) PublicFields() *User { return u }