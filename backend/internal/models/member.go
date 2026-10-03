package models

import (
	"time"

	"gorm.io/gorm"
)

// Member links a user to a server and carries the per server profile.
type Member struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	ServerID  string    `gorm:"type:uuid;not null;uniqueIndex:idx_member_pair;index" json:"serverId"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_member_pair;index" json:"-"`
	Nickname  string    `gorm:"size:64;not null;default:''" json:"nickname"`
	JoinedAt  time.Time `json:"joinedAt"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	User    User         `gorm:"foreignKey:UserID;references:ID" json:"user"`
	RoleIDs []string     `gorm:"-" json:"roleIds"`
	Roles   []MemberRole `gorm:"foreignKey:MemberID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Member) TableName() string { return "members" }

// BeforeCreate fills the generated columns before the first insert.
func (m *Member) BeforeCreate(tx *gorm.DB) error {
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
	if m.JoinedAt.IsZero() {
		m.JoinedAt = now
	}
	if m.RoleIDs == nil {
		m.RoleIDs = []string{}
	}
	return nil
}

// MemberRole is the join row assigning a role to a member.
type MemberRole struct {
	MemberID string    `gorm:"type:uuid;primaryKey;autoIncrement:false" json:"-"`
	RoleID   string    `gorm:"type:uuid;primaryKey;autoIncrement:false;index" json:"-"`
	CreatedAt time.Time `json:"-"`
}

// TableName pins the table name so AutoMigrate is deterministic.
func (MemberRole) TableName() string { return "member_roles" }

// BeforeCreate fills the generated columns before the first insert.
func (mr *MemberRole) BeforeCreate(tx *gorm.DB) error {
	if mr.CreatedAt.IsZero() {
		mr.CreatedAt = time.Now().UTC()
	}
	return nil
}