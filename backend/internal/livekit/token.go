// Package livekit mints short lived video access tokens for voice and video
// channels and direct message rooms.
package livekit

import (
	"fmt"
	"time"

	"github.com/cordis/backend/internal/config"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
)

// TokenTTL is the lifetime of a minted token; the contract reports 3600.
const TokenTTL = time.Hour

// ExpiresIn is TokenTTL expressed in seconds for the API response.
const ExpiresIn int64 = int64(TokenTTL / time.Second)

// Grant describes the permissions requested for a room.
type Grant struct {
	RoomJoin      bool
	Room          string
	CanPublish    bool
	CanSubscribe  bool
	CanPublishData bool
}

// Token is the response payload of POST /api/livekit/token.
type Token struct {
	Token     string `json:"token"`
	URL       string `json:"url"`
	RoomName  string `json:"roomName"`
	ExpiresIn int64  `json:"expiresIn"`
}

// Service issues LiveKit access tokens using the configured API credentials.
type Service struct {
	apiKey    string
	apiSecret string
	url       string
}

// New builds the token service from configuration.
func New(cfg config.LiveKit) *Service {
	return &Service{apiKey: cfg.APIKey, apiSecret: cfg.APISecret, url: cfg.URL}
}

// URL returns the LiveKit websocket endpoint advertised to clients.
func (s *Service) URL() string { return s.url }

// Issue mints a signed access token for a room.
func (s *Service) Issue(identity, displayName string, grant Grant) (*Token, error) {
	accessToken := auth.NewAccessToken(s.apiKey, s.apiSecret)
	accessToken.SetIdentity(identity)
	accessToken.SetName(displayName)
	accessToken.SetTTL(int(TokenTTL.Seconds()))
	accessToken.AddGrant(&livekit.VideoGrant{
		RoomJoin:       grant.RoomJoin,
		Room:           grant.Room,
		CanPublish:     grant.CanPublish,
		CanSubscribe:   grant.CanSubscribe,
		CanPublishData: grant.CanPublishData,
	})
	token, err := accessToken.ToJWT()
	if err != nil {
		return nil, fmt.Errorf("mint livekit token: %w", err)
	}
	return &Token{
		Token:     token,
		URL:       s.url,
		RoomName:  grant.Room,
		ExpiresIn: ExpiresIn,
	}, nil
}