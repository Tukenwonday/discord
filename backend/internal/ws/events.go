package ws

import (
	"encoding/json"
	"time"
)

// Client to server event names from section 6 of the contract.
const (
	ClientMessageCreate  = "message.create"
	ClientMessageUpdate  = "message.update"
	ClientMessageDelete  = "message.delete"
	ClientTypingStart    = "typing.start"
	ClientTypingStop     = "typing.stop"
	ClientPresenceUpdate = "presence.update"
	ClientFriendRequest  = "friend.request"
	ClientFriendAccepted = "friend.accepted"
	ClientHeartbeat      = "heartbeat"
)

// Server to client event names from section 6 of the contract.
const (
	EventReady        = "ready"
	EventMessageCreate  = "message.created"
	EventMessageUpdate  = "message.updated"
	EventMessageDelete  = "message.deleted"
	EventTypingStart    = "typing.started"
	EventTypingStop     = "typing.stopped"
	EventPresence       = "presence.updated"
	EventFriendRequest  = "friend.request"
	EventFriendAccepted = "friend.accepted"
	EventError          = "error"
)

// Envelope is the identical wire format used in both directions.
type Envelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data,omitempty"`
	Nonce string          `json:"nonce,omitempty"`
}

// ReadyPayload is sent immediately after a successful upgrade.
type ReadyPayload struct {
	UserID     string    `json:"userId"`
	SessionID  string    `json:"sessionId"`
	ServerTime time.Time `json:"serverTime"`
}

// ErrorPayload is emitted instead of dropping the connection on a bad frame.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Nonce   string `json:"nonce,omitempty"`
}

// TypingStartedPayload carries the full user for typing.started.
type TypingStartedPayload struct {
	ChannelID string `json:"channelId"`
	User      any    `json:"user"`
}

// TypingStoppedPayload carries only the user id for typing.stopped.
type TypingStoppedPayload struct {
	ChannelID string `json:"channelId"`
	UserID    string `json:"userId"`
}

// FriendPayload wraps a friend view for the friend events.
type FriendPayload struct {
	Friend any `json:"friend"`
}

// RedisEnvelope fans an event out to the other replicas. The origin instance id
// lets a replica skip the delivery it already performed locally.
type RedisEnvelope struct {
	Origin   string          `json:"origin"`
	Event    string          `json:"event"`
	Data     json.RawMessage `json:"data"`
	UserIDs  []string        `json:"userIds,omitempty"`
	Channel  string          `json:"channel,omitempty"`
	Exclude  []string        `json:"exclude,omitempty"`
}