package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/permissions"
	"gorm.io/gorm"
)

// ServerDetail is the composite payload of section 4 of the contract.
type ServerDetail struct {
	Server      *models.Server   `json:"server"`
	Channels    []models.Channel `json:"channels"`
	Members     []models.Member  `json:"members"`
	Roles       []models.Role    `json:"roles"`
	Owner       *models.User     `json:"owner"`
	UnreadCount int              `json:"unreadCount"`
	Invites     []models.Invite  `json:"invites"`
}

// CreateServerInput is the validated create server payload.
type CreateServerInput struct {
	Name        string
	Description string
}

// UpdateServerInput is the partial server update payload.
type UpdateServerInput struct {
	Name        *string
	Description *string
	IconURL     *string
	BannerURL   *string
}

// Server implements server lifecycle and membership queries.
type Server struct {
	deps Deps
}

// NewServer builds the server service.
func NewServer(deps Deps) *Server { return &Server{deps: deps} }

// List returns the paginated servers the user belongs to.
func (s *Server) List(userID string, opts PageOptions) (*Paginated[models.Server], error) {
	opts = opts.Normalize()
	q := s.deps.DB.Model(&models.Server{}).
		Joins("JOIN members ON members.server_id = servers.id AND members.user_id = ?", userID)
	rows, hasMore, next, err := paginate[models.Server](s.deps.DB, opts, "servers", q)
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	if rows == nil {
		rows = []models.Server{}
	}
	return &Paginated[models.Server]{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// Create provisions a server with its owner membership, an @everyone role and
// the default general text channel plus voice channel.
func (s *Server) Create(ownerID string, in CreateServerInput) (*ServerDetail, error) {
	name := strings.TrimSpace(in.Name)
	fields := map[string]string{}
	switch {
	case name == "":
		fields["name"] = "required"
	case utf8.RuneCountInString(name) > 64:
		fields["name"] = "must be between 1 and 64 characters"
	}
	description := strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(description) > 512 {
		fields["description"] = "must be at most 512 characters"
	}
	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}

	var owner models.User
	if err := s.deps.DB.Where("id = ?", ownerID).Take(&owner).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("owner not found")
		}
		return nil, fmt.Errorf("load owner: %w", err)
	}

	server := &models.Server{Name: name, Description: description, OwnerID: ownerID}
	everyone := &models.Role{Name: models.EveryoneRoleName, Color: "#5865f2", Permissions: permissions.Everyone}
	ownerRole := &models.Role{Name: "Owner", Color: "#f9a62b", Permissions: int64(permissions.Administrator), Hoist: true, Position: 100}
	general := &models.Channel{Name: "general", Type: models.ChannelTypeText, Position: 0}
	voice := &models.Channel{Name: "voice", Type: models.ChannelTypeVoice, Position: 1}

	err := s.deps.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(server).Error; err != nil {
			return fmt.Errorf("create server: %w", err)
		}
		everyone.ServerID = server.ID
		ownerRole.ServerID = server.ID
		if err := tx.Create(everyone).Error; err != nil {
			return fmt.Errorf("create everyone role: %w", err)
		}
		if err := tx.Create(ownerRole).Error; err != nil {
			return fmt.Errorf("create owner role: %w", err)
		}
		member := &models.Member{ServerID: server.ID, UserID: ownerID, RoleIDs: []string{everyone.ID, ownerRole.ID}}
		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("create member: %w", err)
		}
		links := []models.MemberRole{
			{MemberID: member.ID, RoleID: everyone.ID},
			{MemberID: member.ID, RoleID: ownerRole.ID},
		}
		if err := tx.Create(&links).Error; err != nil {
			return fmt.Errorf("assign member roles: %w", err)
		}
		general.ServerID = server.ID
		voice.ServerID = server.ID
		if err := tx.Create(general).Error; err != nil {
			return fmt.Errorf("create general channel: %w", err)
		}
		if err := tx.Create(voice).Error; err != nil {
			return fmt.Errorf("create voice channel: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.Get(server.ID, ownerID)
}

// Get returns the full server detail for a member; non members get 403.
func (s *Server) Get(serverID, userID string) (*ServerDetail, error) {
	if strings.TrimSpace(serverID) == "" {
		return nil, httpx.NewField("id", "required")
	}
	if _, err := requireMember(s.deps.DB, serverID, userID); err != nil {
		return nil, err
	}
	server, err := s.load(serverID)
	if err != nil {
		return nil, err
	}
	channels, err := s.channels(serverID)
	if err != nil {
		return nil, err
	}
	members, err := s.members(serverID)
	if err != nil {
		return nil, err
	}
	roles, err := s.roles(serverID)
	if err != nil {
		return nil, err
	}
	invites, err := s.invites(serverID)
	if err != nil {
		return nil, err
	}
	var owner models.User
	if err := s.deps.DB.Where("id = ?", server.OwnerID).Take(&owner).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load owner: %w", err))
	}
	return &ServerDetail{
		Server:      server,
		Channels:    channels,
		Members:     members,
		Roles:       roles,
		Owner:       &owner,
		UnreadCount: 0,
		Invites:     invites,
	}, nil
}

// Update applies a partial server update, requiring the owner or ManageServer.
func (s *Server) Update(serverID, userID string, in UpdateServerInput) (*models.Server, error) {
	if _, err := requireMember(s.deps.DB, serverID, userID); err != nil {
		return nil, err
	}
	if err := RequirePermission(s.deps.DB, serverID, userID, permissions.ManageServer); err != nil {
		return nil, err
	}
	fields := map[string]string{}
	updates := map[string]any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		switch {
		case name == "":
			fields["name"] = "required"
		case utf8.RuneCountInString(name) > 64:
			fields["name"] = "must be between 1 and 64 characters"
		default:
			updates["name"] = name
		}
	}
	if in.Description != nil {
		description := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(description) > 512 {
			fields["description"] = "must be at most 512 characters"
		} else {
			updates["description"] = description
		}
	}
	if in.IconURL != nil {
		updates["icon_url"] = nullableURL(*in.IconURL)
	}
	if in.BannerURL != nil {
		updates["banner_url"] = nullableURL(*in.BannerURL)
	}
	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}
	if len(updates) == 0 {
		return s.load(serverID)
	}
	if err := s.deps.DB.Model(&models.Server{}).Where("id = ?", serverID).Updates(updates).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("update server: %w", err))
	}
	return s.load(serverID)
}

// Delete removes a server; only the owner may do so.
func (s *Server) Delete(serverID, userID string) error {
	server, err := s.load(serverID)
	if err != nil {
		return err
	}
	if server.OwnerID != userID {
		return httpx.NewForbidden("only the server owner can delete it")
	}
	if err := s.deps.DB.Where("id = ?", serverID).Delete(&models.Server{}).Error; err != nil {
		return httpx.NewInternalFromError(fmt.Errorf("delete server: %w", err))
	}
	s.deps.Log.Info().Str("server_id", serverID).Msg("server deleted")
	return nil
}

func (s *Server) load(serverID string) (*models.Server, error) {
	var server models.Server
	if err := s.deps.DB.Where("id = ?", serverID).Take(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("server not found")
		}
		return nil, httpx.NewInternalFromError(fmt.Errorf("load server: %w", err))
	}
	return &server, nil
}

func (s *Server) channels(serverID string) ([]models.Channel, error) {
	var rows []models.Channel
	if err := s.deps.DB.Where("server_id = ?", serverID).
		Order("position ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load channels: %w", err))
	}
	if rows == nil {
		rows = []models.Channel{}
	}
	return rows, nil
}

func (s *Server) members(serverID string) ([]models.Member, error) {
	var rows []models.Member
	if err := s.deps.DB.Preload("User").Where("server_id = ?", serverID).
		Order("joined_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load members: %w", err))
	}
	for i := range rows {
		ids, err := memberRoleIDs(s.deps.DB, rows[i].ID)
		if err != nil {
			return nil, httpx.NewInternalFromError(err)
		}
		if ids == nil {
			ids = []string{}
		}
		rows[i].RoleIDs = ids
	}
	if rows == nil {
		rows = []models.Member{}
	}
	return rows, nil
}

func (s *Server) roles(serverID string) ([]models.Role, error) {
	var rows []models.Role
	if err := s.deps.DB.Where("server_id = ?", serverID).
		Order("position ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load roles: %w", err))
	}
	if rows == nil {
		rows = []models.Role{}
	}
	return rows, nil
}

func (s *Server) invites(serverID string) ([]models.Invite, error) {
	var rows []models.Invite
	if err := s.deps.DB.Where("server_id = ?", serverID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("load invites: %w", err))
	}
	if rows == nil {
		rows = []models.Invite{}
	}
	return rows, nil
}