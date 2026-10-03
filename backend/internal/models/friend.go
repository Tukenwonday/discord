package models

import (
	"time"

	"gorm.io/gorm"
)

// Friend is a directional relationship row from requester to addressee. The
// JSON payload always carries the *other* participant in the user field.
type Friend struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      string    `gorm:"type:uuid;not null;uniqueIndex:idx_friend_pair;index" json:"-"`
	AddresseeID string    `gorm:"type:uuid;not null;uniqueIndex:idx_friend_pair;index" json:"-"`
	Status      string    `gorm:"size:16;not null;default:'pending'" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"-"`

	// Other is the counterpart rendered for the API consumer.
	Other *User `gorm:"-" json:"user"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Friend) TableName() string { return "friends" }

// BeforeCreate fills the generated columns before the first insert.
func (f *Friend) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = newID()
	}
	now := time.Now().UTC()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = now
	}
	if f.Status == "" {
		f.Status = FriendPending
	}
	return nil
}

// FriendView renders a friend row from the perspective of viewerID.
type FriendView struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	User      *User     `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
}

// View projects the row with the other participant resolved.
func (f *Friend) View(other *User) FriendView {
	return FriendView{ID: f.ID, Status: f.Status, User: other, CreatedAt: f.CreatedAt}
}