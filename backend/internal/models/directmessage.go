package models

import (
	"time"

	"gorm.io/gorm"
)

// DirectMessage is a private conversation between two or more users. The id
// doubles as the channel id used by the message table.
type DirectMessage struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Recipients []User        `gorm:"many2many:dm_recipients;joinForeignKey:DMID;joinReferences:UserID" json:"recipients"`
	LastMessage *Message     `gorm:"-" json:"lastMessage"`
	MemberRows  []DMRecipient `gorm:"foreignKey:DMID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (DirectMessage) TableName() string { return "direct_messages" }

// BeforeCreate fills the generated columns before the first insert.
func (d *DirectMessage) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = newID()
	}
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}
	return nil
}

// DMRecipient is the join row describing who belongs to a direct message.
type DMRecipient struct {
	DMID      string    `gorm:"type:uuid;primaryKey;autoIncrement:false" json:"-"`
	UserID    string    `gorm:"type:uuid;primaryKey;autoIncrement:false;index" json:"-"`
	CreatedAt time.Time `json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (DMRecipient) TableName() string { return "dm_recipients" }

// BeforeCreate fills the generated columns before the first insert.
func (r *DMRecipient) BeforeCreate(tx *gorm.DB) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	return nil
}