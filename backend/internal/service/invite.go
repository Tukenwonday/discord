package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/permissions"
	"gorm.io/gorm"
)

// inviteCodeLength is the fixed length of an invite code from section 5.
const inviteCodeLength = 10

// CreateInviteInput is the validated invite creation payload.
type CreateInviteInput struct {
	ChannelID      *string
	MaxUses        int
	ExpiresInHours int
}

// Invite implements invite creation, listing, joining and revocation.
type Invite struct {
	deps Deps
}

// NewInvite builds the invite service.
func NewInvite(deps Deps) *Invite { return &Invite{deps: deps} }

// Create mints an invite for a server, requiring the CreateInstantInvite
// permission unless the caller owns the server.
func (s *Invite) Create(serverID, userID string, in CreateInviteInput) (*models.Invite, error) {
	if _, err := requireMember(s.deps.DB, serverID, userID); err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, serverID, userID, permissions.CreateInstantInvite); err != nil {
		return nil, err
	}

	fields := map[string]string{}
	if in.MaxUses < 0 {
		fields["maxUses"] = "must be zero or greater"
	}
	if in.MaxUses > 10000 {
		fields["maxUses"] = "must be at most 10000"
	}
	if in.ExpiresInHours < 0 {
		fields["expiresInHours"] = "must be zero or greater"
	}
	if in.ExpiresInHours > 8760 {
		fields["expiresInHours"] = "must be at most 8760"
	}
	if in.ChannelID != nil && strings.TrimSpace(*in.ChannelID) != "" {
		var channel models.Channel
		err := s.deps.DB.Where("id = ? AND server_id = ?", *in.ChannelID, serverID).Take(&channel).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, httpx.NewInternalFromError(fmt.Errorf("load channel: %w", err))
			}
			fields["channelId"] = "channel not found in this server"
		}
	}
	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}

	invite := &models.Invite{
		ServerID:  serverID,
		InviterID: userID,
		ChannelID: optionalID(in.ChannelID),
		MaxUses:   in.MaxUses,
	}
	if in.ExpiresInHours > 0 {
		expires := time.Now().UTC().Add(time.Duration(in.ExpiresInHours) * time.Hour)
		invite.ExpiresAt = &expires
	}
	if err := s.deps.DB.Create(invite).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("create invite: %w", err))
	}
	return s.load(invite.Code)
}

// List returns the invites of a server, which only the owner may read.
func (s *Invite) List(serverID, userID string) ([]models.Invite, error) {
	if err := s.requireOwner(serverID, userID); err != nil {
		return nil, err
	}
	rows := []models.Invite{}
	if err := s.deps.DB.Preload("Server").Preload("Inviter").
		Where("server_id = ?", serverID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load invites: %w", err))
	}
	return rows, nil
}

// Get returns one invite by code. The endpoint is public so a link can be
// previewed before signing in.
func (s *Invite) Get(code string) (*models.Invite, error) {
	return s.load(strings.TrimSpace(code))
}

// Revoke deletes an invite. Only the server owner may do so.
func (s *Invite) Revoke(code, userID string) error {
	invite, err := s.load(strings.TrimSpace(code))
	if err != nil {
		return err
	}
	if err := s.requireOwner(invite.ServerID, userID); err != nil {
		return err
	}
	if err := s.deps.DB.Where("code = ?", invite.Code).Delete(&models.Invite{}).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("delete invite: %w", err))
	}
	return nil
}

// Join adds the user to the server of an invite and returns the server detail.
// An expired or exhausted invite is refused rather than silently ignored.
func (s *Invite) Join(code, userID string) (*ServerDetail, error) {
	invite, err := s.load(strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	if invite.IsExpired(time.Now().UTC()) || invite.IsExhausted() {
		return nil, httpx.NewForbidden("this invite is no longer valid")
	}

	_, memberErr := requireMember(s.deps.DB, invite.ServerID, userID)
	switch {
	case memberErr == nil:
		// Already a member: only the usage counter moves.
	case isForbidden(memberErr):
		if err := s.joinServer(invite.ServerID, userID); err != nil {
			return nil, err
		}
	default:
		return nil, memberErr
	}

	if err := s.consume(invite); err != nil {
		return nil, err
	}
	return NewServer(s.deps).Get(invite.ServerID, userID)
}

// consume increments the usage counter of an invite.
func (s *Invite) consume(invite *models.Invite) error {
	if err := s.deps.DB.Model(&models.Invite{}).Where("code = ?", invite.Code).
		Update("uses", gorm.Expr("uses + 1")).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("count invite use: %w", err))
	}
	return nil
}

// joinServer creates the membership row and assigns the @everyone role.
func (s *Invite) joinServer(serverID, userID string) error {
	var everyoneID string
	err := s.deps.DB.Model(&models.Role{}).
		Where("server_id = ? AND name = ?", serverID, models.EveryoneRoleName).
		Select("id").Scan(&everyoneID).Error
	if err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("load everyone role: %w", err))
	}

	member := &models.Member{ServerID: serverID, UserID: userID}
	err = s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("create member: %w", err)
		}
		if everyoneID == "" {
			return nil
		}
		link := models.MemberRole{MemberID: member.ID, RoleID: everyoneID}
		if err := tx.Create(&link).Error; err != nil {
			return fmt.Errorf("assign everyone role: %w", err)
		}
		return nil
	})
	if err != nil {
		return httpx.NewInternalFromError(err)
	}
	return nil
}

// requireOwner enforces that only the server owner may manage invites.
func (s *Invite) requireOwner(serverID, userID string) error {
	var server models.Server
	if err := s.deps.DB.Select("id, owner_id").Where("id = ?", serverID).Take(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.NewNotFound("server not found")
		}
		return httpx.NewInternalFromError(fmt.Errorf("load server: %w", err))
	}
	if server.OwnerID != userID {
		return httpx.NewForbidden("only the server owner can manage invites")
	}
	return nil
}

// load reads one invite with its server and inviter preloaded.
func (s *Invite) load(code string) (*models.Invite, error) {
	if strings.TrimSpace(code) == "" {
		return nil, httpx.NewField("code", "required")
	}
	if utf8.RuneCountInString(code) != inviteCodeLength {
		return nil, httpx.NewNotFound("invite not found")
	}
	var invite models.Invite
	err := s.deps.DB.Preload("Server").Preload("Inviter").Where("code = ?", code).Take(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("invite not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load invite: %w", err))
	}
	return &invite, nil
}

// URL builds the shareable join link of an invite.
func (s *Invite) URL(code string) string {
	return s.deps.Config.PublicURL + "/invite/" + code
}