package ws

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
	"github.com/cordis/backend/internal/service"
)

// Payload shapes of the client to server events of section 6.
type (
	// messageCreateData is the body of message.create.
	messageCreateData struct {
		ChannelID   string                   `json:"channelId"`
		Content     string                   `json:"content"`
		ReplyToID   *string                  `json:"replyToId,omitempty"`
		Attachments []models.AttachmentInput `json:"attachments,omitempty"`
	}
	// messageUpdateData is the body of message.update.
	messageUpdateData struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	// messageDeleteData is the body of message.delete.
	messageDeleteData struct {
		ID string `json:"id"`
	}
	// typingData is the body of typing.start and typing.stop.
	typingData struct {
		ChannelID string `json:"channelId"`
	}
	// presenceData is the body of presence.update.
	presenceData struct {
		Status       string `json:"status"`
		CustomStatus string `json:"customStatus,omitempty"`
	}
	// friendRequestData is the body of friend.request.
	friendRequestData struct {
		UserID string `json:"userId"`
	}
	// friendAcceptedData is the body of friend.accepted.
	friendAcceptedData struct {
		FriendID string `json:"friendId"`
	}
)

// dispatch routes one inbound frame to its handler. Returning an error makes
// the read pump answer with an error event carrying the inbound nonce instead of
// dropping the connection.
func (h *Hub) dispatch(c *client, env Envelope) error {
	switch env.Event {
	case ClientMessageCreate:
		return h.handleMessageCreate(c, env.Data)
	case ClientMessageUpdate:
		return h.handleMessageUpdate(c, env.Data)
	case ClientMessageDelete:
		return h.handleMessageDelete(c, env.Data)
	case ClientTypingStart:
		return h.handleTyping(c, env.Data, true)
	case ClientTypingStop:
		return h.handleTyping(c, env.Data, false)
	case ClientPresenceUpdate:
		return h.handlePresenceUpdate(c, env.Data)
	case ClientFriendRequest:
		return h.handleFriendRequest(c, env.Data)
	case ClientFriendAccepted:
		return h.handleFriendAccepted(c, env.Data)
	case ClientHeartbeat:
		return h.handleHeartbeat(c)
	default:
		return httpx.NewValidation("unknown event: " + env.Event)
	}
}

// decodeData unmarshals the data member of an envelope, rejecting malformed
// frames with a validation error.
func decodeData(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return httpx.NewValidation("event data is required")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return httpx.NewValidation("event data is malformed")
	}
	return nil
}

// handleMessageCreate stores a message and lets the service broadcast it, so a
// message sent over the socket and one sent over HTTP produce the same event.
func (h *Hub) handleMessageCreate(c *client, raw json.RawMessage) error {
	var data messageCreateData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.ChannelID) == "" {
		return httpx.NewField("channelId", "required")
	}
	if err := service.RequireDestination(h.deps.DB, data.ChannelID, c.userID); err != nil {
		return err
	}

	in := service.CreateMessageInput{
		Content:     data.Content,
		ReplyToID:   data.ReplyToID,
		Attachments: data.Attachments,
	}

	messages := service.NewMessage(h.deps)
	// A destination with no owning server is a direct message conversation.
	if service.DestinationServerID(h.deps.DB, data.ChannelID) == "" {
		if _, err := messages.CreateDM(data.ChannelID, c.userID, in); err != nil {
			return err
		}
		h.Join(c, data.ChannelID)
		return nil
	}
	if _, err := messages.Create(data.ChannelID, c.userID, in); err != nil {
		return err
	}
	h.Join(c, data.ChannelID)
	return nil
}

// handleMessageUpdate edits a message owned by the sender.
func (h *Hub) handleMessageUpdate(c *client, raw json.RawMessage) error {
	var data messageUpdateData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.ID) == "" {
		return httpx.NewField("id", "required")
	}
	_, err := service.NewMessage(h.deps).Update(data.ID, c.userID, data.Content)
	return err
}

// handleMessageDelete removes a message the sender authored or can moderate.
func (h *Hub) handleMessageDelete(c *client, raw json.RawMessage) error {
	var data messageDeleteData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.ID) == "" {
		return httpx.NewField("id", "required")
	}
	return service.NewMessage(h.deps).Delete(data.ID, c.userID)
}

// handleTyping broadcasts a typing indicator to the destination audience and
// excludes the originating socket, as required by section 6.
func (h *Hub) handleTyping(c *client, raw json.RawMessage, start bool) error {
	var data typingData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.ChannelID) == "" {
		return httpx.NewField("channelId", "required")
	}
	if err := service.RequireDestination(h.deps.DB, data.ChannelID, c.userID); err != nil {
		return err
	}

	ctx := context.Background()
	if start {
		if err := h.cache.SetTyping(ctx, data.ChannelID, c.userID); err != nil {
			return httpx.NewInternalFromError(err)
		}
	} else if err := h.cache.ClearTyping(ctx, data.ChannelID, c.userID); err != nil {
		return httpx.NewInternalFromError(err)
	}

	if start {
		user, err := service.NewUser(h.deps).Get(c.userID)
		if err != nil {
			return err
		}
		h.PublishToChannelExcluding(data.ChannelID, EventTypingStart,
			TypingStartedPayload{ChannelID: data.ChannelID, User: user}, []string{c.userID})
		return nil
	}

	h.PublishToChannelExcluding(data.ChannelID, EventTypingStop,
		TypingStoppedPayload{ChannelID: data.ChannelID, UserID: c.userID}, []string{c.userID})
	return nil
}

// handlePresenceUpdate applies an explicit status change. The presence service
// already broadcasts it to the shared audience, so nothing is re-sent here.
func (h *Hub) handlePresenceUpdate(c *client, raw json.RawMessage) error {
	var data presenceData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	_, err := service.NewPresenceStore(h.deps).Update(c.userID, data.Status, data.CustomStatus)
	return err
}

// handleFriendRequest creates a pending friendship.
func (h *Hub) handleFriendRequest(c *client, raw json.RawMessage) error {
	var data friendRequestData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.UserID) == "" {
		return httpx.NewField("userId", "required")
	}
	_, _, err := service.NewFriend(h.deps).Request(c.userID, data.UserID)
	return err
}

// handleFriendAccepted accepts an incoming request.
func (h *Hub) handleFriendAccepted(c *client, raw json.RawMessage) error {
	var data friendAcceptedData
	if err := decodeData(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.FriendID) == "" {
		return httpx.NewField("friendId", "required")
	}
	_, err := service.NewFriend(h.deps).Accept(c.userID, data.FriendID)
	return err
}

// handleHeartbeat refreshes the presence TTL of the sender so a long lived
// session never silently goes offline.
func (h *Hub) handleHeartbeat(c *client) error {
	return service.NewPresenceStore(h.deps).Heartbeat(c.userID)
}

// compileTimeAssertion keeps the dispatcher honest about the publisher contract.
var _ service.Publisher = (*Hub)(nil)
