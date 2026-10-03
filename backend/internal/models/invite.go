package models

import (
	"crypto/rand"
	"math/big"
	"time"

	"gorm.io/gorm"
)

// Invite is a shareable join link for a server.
type Invite struct {
	Code      string     `gorm:"size:10;primaryKey" json:"code"`
	ServerID  string     `gorm:"type:uuid;not null;index" json:"-"`
	ChannelID *string    `gorm:"type:uuid" json:"channelId"`
	InviterID string     `gorm:"type:uuid;not null" json:"-"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Uses      int        `gorm:"not null;default:0" json:"uses"`
	MaxUses   int        `gorm:"not null;default:0" json:"maxUses"`
	CreatedAt time.Time  `json:"createdAt"`

	Server  Server `gorm:"foreignKey:ServerID;references:ID" json:"server"`
	Inviter User   `gorm:"foreignKey:InviterID;references:ID" json:"inviter"`
}

// inviteAlphabet is the 36 character alphabet used for invite codes; the
// contract requires 10 lowercase alphanumeric characters.
const inviteAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// inviteCodeLength is the fixed length of a generated invite code.
const inviteCodeLength = 10

// newInviteCode returns a cryptographically random invite code.
func newInviteCode() string {
	out := make([]byte, inviteCodeLength)
	max := big.NewInt(int64(len(inviteAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = inviteAlphabet[i%len(inviteAlphabet)]
			continue
		}
		out[i] = inviteAlphabet[n.Int64()]
	}
	return string(out)
}

// TableName pins the table name so AutoMigrate is deterministic.
func (Invite) TableName() string { return "invites" }

// BeforeCreate fills the generated columns before the first insert.
func (i *Invite) BeforeCreate(tx *gorm.DB) error {
	if i.Code == "" {
		i.Code = newInviteCode()
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now().UTC()
	}
	return nil
}

// IsExpired reports whether the invite may no longer be used.
func (i *Invite) IsExpired(now time.Time) bool {
	return i.ExpiresAt != nil && !i.ExpiresAt.After(now)
}

// IsExhausted reports whether the invite reached its usage cap. A MaxUses of
// zero means unlimited.
func (i *Invite) IsExhausted() bool { return i.MaxUses > 0 && i.Uses >= i.MaxUses }