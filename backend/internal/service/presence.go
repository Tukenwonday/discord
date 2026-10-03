package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cordis/backend/internal/cache"
	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
)

// Presence is the presence.updated event payload.
type Presence struct {
	UserID       string    `json:"userId"`
	Status       string    `json:"status"`
	CustomStatus string    `json:"customStatus"`
	At           time.Time `json:"at"`
}

// PresenceStore maintains presence in Redis and fans presence.updated out to
// every user who shares a server with the subject.
type PresenceStore struct {
	deps Deps
}

// NewPresenceStore builds the presence service.
func NewPresenceStore(deps Deps) *PresenceStore { return &PresenceStore{deps: deps} }

// Connect marks a user online and broadcasts the change.
func (s *PresenceStore) Connect(ctx context.Context, userID, status, customStatus string) error {
	if status == "" {
		status = models.StatusOnline
	}
	if err := s.deps.Cache.SetPresence(ctx, userID, status, customStatus); err != nil {
		return fmt.Errorf("set presence: %w", err)
	}
	s.broadcast(ctx, userID, status, customStatus)
	return nil
}

// Disconnect marks a user offline unless other sockets of the same user remain
// connected, which the hub knows through its own socket count.
func (s *PresenceStore) Disconnect(ctx context.Context, userID string) error {
	if err := s.deps.Cache.DeletePresence(ctx, userID); err != nil {
		return fmt.Errorf("clear presence: %w", err)
	}
	s.broadcast(ctx, userID, models.StatusOffline, "")
	return nil
}

// Update applies an explicit presence.update from a client.
func (s *PresenceStore) Update(userID, status, customStatus string) (*Presence, error) {
	clean := strings.ToLower(strings.TrimSpace(status))
	switch {
	case clean == "":
		clean = models.StatusOnline
	case clean == models.StatusOffline:
		return nil, httpx.NewValidation("validation failed").
			WithFields(map[string]string{"status": "must be one of online, idle, dnd, invisible"})
	case !models.ValidStatus(clean):
		return nil, httpx.NewValidation("validation failed").
			WithFields(map[string]string{"status": "must be one of online, idle, dnd, invisible"})
	}
	if utf8.RuneCountInString(customStatus) > 128 {
		return nil, httpx.NewValidation("validation failed").
			WithFields(map[string]string{"customStatus": "must be at most 128 characters"})
	}

	ctx := ctxBackground()
	if err := s.deps.Cache.SetPresence(ctx, userID, clean, customStatus); err != nil {
		return nil, httpx.NewInternalFromError(fmt.Errorf("set presence: %w", err))
	}
	presence := &Presence{
		UserID:       userID,
		Status:       clean,
		CustomStatus: customStatus,
		At:           time.Now().UTC(),
	}
	s.deps.Pub.PublishToUsers(s.audience(ctx, userID), EventPresenceUpdate, presence)
	return presence, nil
}

// Heartbeat refreshes the online marker so a long lived session keeps its
// presence TTL, as required by section 8 of the contract.
func (s *PresenceStore) Heartbeat(userID string) error {
	if err := s.deps.Cache.RefreshOnline(ctxBackground(), userID); err != nil {
		return fmt.Errorf("refresh presence: %w", err)
	}
	return nil
}

// Get reads the presence of a user, defaulting to offline.
func (s *PresenceStore) Get(ctx context.Context, userID string) Presence {
	stored, err := s.deps.Cache.GetPresence(ctx, userID)
	if err != nil || stored.Status == "" {
		return Presence{UserID: userID, Status: models.StatusOffline, At: time.Now().UTC()}
	}
	at := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339, stored.At); err == nil {
		at = parsed.UTC()
	}
	return Presence{
		UserID:       userID,
		Status:       stored.Status,
		CustomStatus: stored.CustomStatus,
		At:           at,
	}
}

// Online reports whether the user currently holds a live online marker.
func (s *PresenceStore) Online(ctx context.Context, userID string) bool {
	online, err := s.deps.Cache.IsOnline(ctx, userID)
	if err != nil {
		return false
	}
	return online
}

// audience lists every user who shares a server with the subject, deduplicated.
func (s *PresenceStore) audience(ctx context.Context, userID string) []string {
	servers := []string{}
	err := s.deps.DB.Model(&models.Member{}).Where("user_id = ?", userID).
		Pluck("server_id", &servers).Error
	if err != nil || len(servers) == 0 {
		return []string{}
	}
	ids := []string{}
	if err := s.deps.DB.Model(&models.Member{}).Where("server_id IN ?", servers).
		Distinct().Pluck("user_id", &ids).Error; err != nil {
		return []string{}
	}
	return ids
}

// broadcast publishes presence.updated to the subject's shared audience.
func (s *PresenceStore) broadcast(ctx context.Context, userID, status, customStatus string) {
	presence := Presence{
		UserID:       userID,
		Status:       status,
		CustomStatus: customStatus,
		At:           time.Now().UTC(),
	}
	s.deps.Pub.PublishToUsers(s.audience(ctx, userID), EventPresenceUpdate, presence)
}

// MarkOffline is the presence cleanup used when the whole process shuts down.
func (s *PresenceStore) MarkOffline(ctx context.Context, userID string) {
	if err := s.deps.Cache.DeletePresence(ctx, userID); err != nil {
		s.deps.Log.Warn().Err(err).Str("user_id", userID).Msg("clear presence failed")
	}
}

// PresenceKeyPrefix exposes the Redis prefix for operational tooling.
func PresenceKeyPrefix() string { return cache.KeyPrefixPresence }