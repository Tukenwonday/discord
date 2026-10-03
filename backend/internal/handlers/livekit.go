package handlers

import (
	"net/http"
	"strings"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/livekit"
	"github.com/cordis/backend/internal/middleware"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Room types accepted by the token endpoint.
const (
	roomTypeChannel = "channel"
	roomTypeDM      = "dm"
)

// LiveKit mints short lived video access tokens.
type LiveKit struct {
	tokens  *livekit.Service
	channels *service.Channel
	dms     *service.DM
	users   *service.User
}

// NewLiveKit builds the token handler.
func NewLiveKit(deps service.Deps) *LiveKit {
	return &LiveKit{
		tokens:   livekit.New(deps.Config.LiveKit),
		channels: service.NewChannel(deps),
		dms:      service.NewDM(deps),
		users:    service.NewUser(deps),
	}
}

// tokenRequest is the body of POST /api/livekit/token.
type tokenRequest struct {
	RoomName     string `json:"roomName"`
	RoomType     string `json:"roomType"`
	CanPublish   *bool  `json:"canPublish"`
	CanSubscribe *bool  `json:"canSubscribe"`
}

// Token mints a LiveKit access token after verifying that the caller may reach
// the room. The room name must be channel:<uuid> or dm:<uuid>.
func (h *LiveKit) Token(c *gin.Context) {
	var req tokenRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.RespondError(c, err)
		return
	}

	userID := middleware.UserID(c)
	kind, id, ok := splitRoom(req.RoomName)
	if !ok {
		httpx.RespondValidation(c, map[string]string{
			"roomName": "must be channel:<uuid> or dm:<uuid>",
		})
		return
	}

	canPublish, canSubscribe, err := h.resolveAccess(c, kind, id, userID, req)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	user, err := h.users.Get(userID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	token, err := h.tokens.Issue(userID, user.DisplayName, livekit.Grant{
		RoomJoin:      true,
		Room:          req.RoomName,
		CanPublish:    canPublish,
		CanSubscribe:  canSubscribe,
		CanPublishData: true,
	})
	if err != nil {
		httpx.RespondError(c, httpx.NewInternal("could not mint a video token"))
		return
	}
	httpx.RespondJSON(c, http.StatusOK, token)
}

// resolveAccess verifies membership and derives the publish and subscribe
// grants. Voice and video channels may publish, text channels may not.
func (h *LiveKit) resolveAccess(c *gin.Context, kind, id, userID string, req tokenRequest) (bool, bool, error) {
	switch kind {
	case roomTypeChannel:
		if req.RoomType != "" && req.RoomType != roomTypeChannel {
			return false, false, httpx.NewField("roomType", "must be channel or dm")
		}
		channel, err := h.channels.GetForMember(id, userID)
		if err != nil {
			return false, false, err
		}
		// A text channel is a listen only room, while voice and video rooms
		// grant publishing to every member.
		publishable := channel.Type == models.ChannelTypeVoice || channel.Type == models.ChannelTypeVideo
		canPublish := publishable
		if req.CanPublish != nil {
			canPublish = publishable && *req.CanPublish
		}
		canSubscribe := true
		if req.CanSubscribe != nil {
			canSubscribe = *req.CanSubscribe
		}
		return canPublish, canSubscribe, nil
	case roomTypeDM:
		if req.RoomType != "" && req.RoomType != roomTypeDM {
			return false, false, httpx.NewField("roomType", "must be channel or dm")
		}
		if _, err := h.dms.Get(id, userID); err != nil {
			return false, false, err
		}
		canPublish := true
		if req.CanPublish != nil {
			canPublish = *req.CanPublish
		}
		canSubscribe := true
		if req.CanSubscribe != nil {
			canSubscribe = *req.CanSubscribe
		}
		return canPublish, canSubscribe, nil
	default:
		return false, false, httpx.NewValidation("unsupported room type")
	}
}

// splitRoom parses the room name into its kind and identifier.
func splitRoom(roomName string) (string, string, bool) {
	trimmed := strings.TrimSpace(roomName)
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	kind := parts[0]
	id := strings.TrimSpace(parts[1])
	if (kind != roomTypeChannel && kind != roomTypeDM) || id == "" {
		return "", "", false
	}
	return kind, id, true
}