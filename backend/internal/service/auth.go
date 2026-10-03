package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cordis/backend/internal/auth"
	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"gorm.io/gorm"
)

// Validation patterns from section 5 of the contract.
var (
	usernameRe = regexp.MustCompile(`^[a-z0-9_.]+$`)
	emailRe    = regexp.MustCompile(`^[^@\s]+@[^@\s.]+\.[^@\s]+$`)
)

// AuthResult is the response body of register and login.
type AuthResult struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int64        `json:"expiresIn"`
}

// TokenPair is the response body of a token refresh.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// RegisterInput is the validated register payload.
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

// Auth implements registration, login, refresh rotation and logout.
type Auth struct {
	deps Deps
}

// NewAuth builds the auth service.
func NewAuth(deps Deps) *Auth { return &Auth{deps: deps} }

// Register validates the payload, creates the account and issues a token pair.
func (s *Auth) Register(in RegisterInput) (*AuthResult, error) {
	fields := map[string]string{}

	username := strings.ToLower(strings.TrimSpace(in.Username))
	switch {
	case username == "":
		fields["username"] = "required"
	case utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 32:
		fields["username"] = "must be between 3 and 32 characters"
	case !usernameRe.MatchString(username):
		fields["username"] = "may only contain lowercase letters, numbers, dots and underscores"
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	switch {
	case email == "":
		fields["email"] = "required"
	case len(email) > 254:
		fields["email"] = "must be at most 254 characters"
	case !emailRe.MatchString(email):
		fields["email"] = "must be a valid email address"
	}

	password := in.Password
	switch {
	case password == "":
		fields["password"] = "required"
	case len(password) < 8 || len(password) > 128:
		fields["password"] = "must be between 8 and 128 characters"
	case !hasLetterAndDigit(password):
		fields["password"] = "must contain at least one letter and one digit"
	}

	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = username
	} else if utf8.RuneCountInString(displayName) > 64 {
		fields["displayName"] = "must be between 1 and 64 characters"
	}

	if len(fields) > 0 {
		return nil, httpx.NewValidation("validation failed").WithFields(fields)
	}

	var existing models.User
	err := s.deps.DB.Where("username = ? OR email = ?", username, email).Take(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if err == nil {
		if existing.Username == username {
			fields["username"] = "already taken"
		}
		if existing.Email == email {
			fields["email"] = "already taken"
		}
		return nil, httpx.NewConflict("username or email already registered").WithFields(fields)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Status:       models.StatusOffline,
	}
	if err := s.deps.DB.Create(user).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issue(user)
}

// hasLetterAndDigit enforces the password policy of the contract.
func hasLetterAndDigit(s string) bool {
	var letter, digit bool
	for _, r := range s {
		if unicode.IsLetter(r) {
			letter = true
		}
		if unicode.IsDigit(r) {
			digit = true
		}
	}
	return letter && digit
}

// LoginInput carries the credentials of a login request.
type LoginInput struct {
	Login    string
	Password string
}

// Login authenticates with a username or an email address.
func (s *Auth) Login(in LoginInput) (*AuthResult, error) {
	login := strings.ToLower(strings.TrimSpace(in.Login))
	if login == "" {
		return nil, httpx.NewField("login", "required")
	}
	if in.Password == "" {
		return nil, httpx.NewField("password", "required")
	}

	var user models.User
	err := s.deps.DB.Where("username = ? OR email = ?", login, login).Take(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewUnauthorized("invalid credentials")
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	if !auth.CheckPassword(user.PasswordHash, in.Password) {
		return nil, httpx.NewUnauthorized("invalid credentials")
	}
	return s.issue(&user)
}

// Refresh rotates a refresh token: the presented jti is consumed and a fresh
// pair is issued, so a replayed token fails.
func (s *Auth) Refresh(refreshToken string) (*TokenPair, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, httpx.NewField("refreshToken", "required")
	}
	claims, err := s.deps.Tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil, httpx.NewUnauthorized("invalid or expired refresh token")
	}
	userID, err := s.deps.Tokens.RotateRefresh(s.deps.Cache, claims.ID)
	if err != nil {
		if errors.Is(err, auth.ErrRevoked) {
			return nil, httpx.NewUnauthorized("refresh token has already been used")
		}
		return nil, httpx.NewInternalFromError(err)
	}

	var user models.User
	if err := s.deps.DB.Where("id = ?", userID).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NewUnauthorized("account no longer exists")
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	return s.issuePair(&user)
}

// Logout revokes a refresh token; unknown tokens still answer 204.
func (s *Auth) Logout(refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return httpx.NewField("refreshToken", "required")
	}
	claims, err := s.deps.Tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil
	}
	if err := s.deps.Tokens.RevokeRefresh(s.deps.Cache, claims.ID); err != nil {
		return httpx.NewInternalFromError(err)
	}
	return nil
}

// issue builds the user plus token pair response.
func (s *Auth) issue(user *models.User) (*AuthResult, error) {
	pair, err := s.issuePair(user)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

// issuePair mints and stores a new access and refresh token pair.
func (s *Auth) issuePair(user *models.User) (*TokenPair, error) {
	access, expiresIn, err := s.deps.Tokens.IssueAccess(user.ID, user.Username)
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	refresh, jti, err := s.deps.Tokens.IssueRefresh(user.ID)
	if err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	if err := s.deps.Tokens.StoreRefresh(s.deps.Cache, jti, user.ID); err != nil {
		return nil, httpx.NewInternalFromError(err)
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: expiresIn}, nil
}