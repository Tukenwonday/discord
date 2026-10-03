// Package auth issues and verifies the JWT access and refresh tokens and
// hashes passwords with bcrypt.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/cordis/backend/internal/cache"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType values carried in the typ claim.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Errors returned by token verification, translated to unauthorized by the
// transport layer.
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrWrongType    = errors.New("unexpected token type")
	ErrRevoked      = errors.New("token revoked")
)

// AccessClaims is the payload of a short lived access token.
type AccessClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TokenTyp string `json:"typ"`
	jwt.RegisteredClaims
}

// RefreshClaims is the payload of a rotating refresh token.
type RefreshClaims struct {
	UserID   string `json:"user_id"`
	TokenTyp string `json:"typ"`
	jwt.RegisteredClaims
}

// Manager signs tokens and performs refresh rotation against Redis.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager builds a token manager.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// AccessTTL exposes the configured access token lifetime.
func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }

// RefreshTTL exposes the configured refresh token lifetime.
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// IssueAccess mints a signed access token and returns it with its lifetime in
// seconds.
func (m *Manager) IssueAccess(userID, username string) (string, int64, error) {
	now := time.Now().UTC()
	claims := AccessClaims{
		UserID:   userID,
		Username: username,
		TokenTyp: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}
	return tok, int64(m.accessTTL.Seconds()), nil
}

// IssueRefresh mints a refresh token and returns it together with its jti so
// the caller can persist the rotation record.
func (m *Manager) IssueRefresh(userID string) (string, string, error) {
	now := time.Now().UTC()
	jti := uuid.NewString()
	claims := RefreshClaims{
		UserID:   userID,
		TokenTyp: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}
	return tok, jti, nil
}

// ParseAccess validates the signature, lifetime and type of an access token.
func (m *Manager) ParseAccess(token string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}
	if !parsed.Valid || claims.TokenTyp != TokenTypeAccess {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseRefresh validates the signature, lifetime and type of a refresh token.
func (m *Manager) ParseRefresh(token string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}
	if !parsed.Valid || claims.TokenTyp != TokenTypeRefresh {
		return nil, ErrWrongType
	}
	if claims.ID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// StoreRefresh records a refresh jti so it can be rotated exactly once.
func (m *Manager) StoreRefresh(rdb *cache.Client, jti, userID string) error {
	if err := rdb.StoreRefresh(jti, userID, m.refreshTTL); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

// RotateRefresh consumes the jti of an old token and returns the owning user
// id. Reusing a token therefore fails with ErrRevoked.
func (m *Manager) RotateRefresh(rdb *cache.Client, jti string) (string, error) {
	userID, ok, err := rdb.ConsumeRefresh(jti)
	if err != nil {
		return "", fmt.Errorf("consume refresh token: %w", err)
	}
	if !ok {
		return "", ErrRevoked
	}
	return userID, nil
}

// RevokeRefresh removes a refresh jti, used by logout.
func (m *Manager) RevokeRefresh(rdb *cache.Client, jti string) error {
	if err := rdb.DeleteRefresh(jti); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}