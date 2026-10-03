package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"gorm.io/gorm"
)

// User implements profile reads, profile updates and user search.
type User struct {
	deps Deps
}

// NewUser builds the user service.
func NewUser(deps Deps) *User { return &User{deps: deps} }

// Get loads a user by id.
func (s *User) Get(id string) (*models.User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, httpx.NewField("id", "required")
	}
	var user models.User
	if err := s.deps.DB.Where("id = ?", id).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewNotFound("user not found")
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	return &user, nil
}

// UpdateProfileInput carries the partial profile update of PATCH /users/me.
type UpdateProfileInput struct {
	DisplayName  *string
	Bio          *string
	AvatarURL    *string
	BannerURL    *string
	Status       *string
	CustomStatus *string
}

// UpdateProfile validates and applies a partial profile update.
func (s *User) UpdateProfile(id string, in UpdateProfileInput) (*models.User, error) {
	user, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	updates := map[string]any{}

	if in.DisplayName != nil {
		name := strings.TrimSpace(*in.DisplayName)
		if name == "" {
			fields["displayName"] = "required"
		} else if utf8.RuneCountInString(name) > 64 {
			fields["displayName"] = "must be between 1 and 64 characters"
		} else {
			updates["display_name"] = name
		}
	}
	if in.Bio != nil {
		bio := strings.TrimSpace(*in.Bio)
		if utf8.RuneCountInString(bio) > 512 {
			fields["bio"] = "must be at most 512 characters"
		} else {
			updates["bio"] = bio
		}
	}
	if in.AvatarURL != nil {
		updates["avatar_url"] = nullableURL(*in.AvatarURL)
	}
	if in.BannerURL != nil {
		updates["banner_url"] = nullableURL(*in.BannerURL)
	}
	if in.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*in.Status))
		if !models.ValidStatus(status) {
			fields["status"] = "must be one of online, idle, dnd, offline, invisible"
		} else {
			updates["status"] = status
		}
	}
	if in.CustomStatus != nil {
		custom := strings.TrimSpace(*in.CustomStatus)
		if utf8.RuneCountInString(custom) > 128 {
			fields["customStatus"] = "must be at most 128 characters"
		} else {
			updates["custom_status"] = custom
		}
	}

	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}
	if len(updates) == 0 {
		return user, nil
	}

	if err := s.deps.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return s.Get(id)
}

// nullableURL maps an empty string to a NULL column.
func nullableURL(v string) any {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// Search returns a paginated list of users matching a username, display name
// or email query.
func (s *User) Search(query string, opts PageOptions) (*Paginated[models.User], error) {
	opts = opts.Normalize()
	q := s.deps.DB.Model(&models.User{})
	trimmed := strings.TrimSpace(query)
	if trimmed != "" {
		like := "%" + strings.ToLower(trimmed) + "%"
		q = q.Where("LOWER(username) LIKE ? OR LOWER(display_name) LIKE ? OR email LIKE ?", like, like, like)
	}
	rows, hasMore, next, err := paginate[models.User](s.deps.DB, opts, "users", q)
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	if rows == nil {
		rows = []models.User{}
	}
	return &Paginated[models.User]{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// IDsIn is a small helper that validates a non empty identifier list.
func validateIDs(ids []string) error {
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return httpx.NewField("id", "required")
		}
	}
	return nil
}

// errNotFound builds the canonical not found error for an entity name.
func errNotFound(entity string) error { return httpx.NewNotFound(entity + " not found") }