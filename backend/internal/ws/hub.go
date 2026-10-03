// Package ws implements the WebSocket endpoint of section 6: the hub that owns
// the local socket registry and fans events out across replicas through Redis,
// the per socket read and write pumps, presence bookkeeping and the dispatcher
// that turns client frames into service calls.
package ws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/cordis/backend/internal/cache"
	"github.com/cordis/backend/internal/config"
	"github.com/cordis/backend/internal/service"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Hub owns every socket connected to this process. It implements
// service.Publisher so HTTP handlers and WebSocket handlers emit byte identical
// events, and it forwards each event to the other replicas over REDIS_CHANNEL.
type Hub struct {
	cfg   *config.Config
	cache *cache.Client
	deps  service.Deps
	log   zerolog.Logger
	reg   *registry
	// instance identifies this process inside every Redis envelope so a replica
	// can skip the delivery it already performed locally.
	instance string

	mu      sync.RWMutex
	subs    map[*client]struct{}
	stopped bool
}

// NewHub builds a hub for one process instance.
func NewHub(cfg *config.Config, rdb *cache.Client, deps service.Deps, log zerolog.Logger) *Hub {
	return &Hub{
		cfg:      cfg,
		cache:    rdb,
		deps:     deps,
		log:      log,
		reg:      newRegistry(),
		instance: uuid.NewString(),
		subs:     make(map[*client]struct{}),
	}
}

// InstanceID exposes the process identity embedded in every Redis envelope.
func (h *Hub) InstanceID() string { return h.instance }

// Deps exposes the shared service dependencies to the dispatcher.
func (h *Hub) Deps() service.Deps { return h.deps }

// Register adds a socket to the local registry and starts its pumps.
func (h *Hub) Register(c *client) {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		c.close()
		return
	}
	h.subs[c] = struct{}{}
	h.mu.Unlock()

	h.reg.add(c)
	go c.writePump()
	go c.readPump()
}

// Unregister removes a socket and marks the user offline once their last socket
// is gone, so a user with several devices keeps their presence.
func (h *Hub) Unregister(c *client) {
	h.mu.Lock()
	delete(h.subs, c)
	h.mu.Unlock()

	h.reg.remove(c)
	c.close()

	if h.reg.socketCount(c.userID) == 0 {
		h.MarkOffline(c.userID)
	}
}

// Stop closes every local socket, used during graceful shutdown.
func (h *Hub) Stop() {
	h.mu.Lock()
	h.stopped = true
	sockets := make([]*client, 0, len(h.subs))
	for c := range h.subs {
		sockets = append(sockets, c)
	}
	h.mu.Unlock()

	for _, c := range sockets {
		h.Unregister(c)
	}
}

// Online reports whether this process holds at least one socket of a user.
func (h *Hub) Online(userID string) bool { return h.reg.socketCount(userID) > 0 }

// Join subscribes a socket to a channel or direct message conversation so
// channel scoped events can reach it.
func (h *Hub) Join(c *client, destinationID string) {
	if destinationID == "" {
		return
	}
	if err := service.RequireDestination(h.deps.DB, destinationID, c.userID); err != nil {
		return
	}
	h.reg.join(c, destinationID)
}

// Sockets exposes the registry to the dispatcher.
func (h *Hub) Sockets() *registry { return h.reg }

// PublishToUser implements service.Publisher.
func (h *Hub) PublishToUser(userID, event string, data any) {
	if userID == "" {
		return
	}
	frame := marshalEnvelope(event, data)
	h.deliverToUsers([]string{userID}, frame, nil)
	h.fanout(RedisEnvelope{UserIDs: []string{userID}, Event: event, Data: encode(data)})
}

// PublishToUsers implements service.Publisher.
func (h *Hub) PublishToUsers(userIDs []string, event string, data any) {
	unique := dedupe(userIDs)
	if len(unique) == 0 {
		return
	}
	frame := marshalEnvelope(event, data)
	h.deliverToUsers(unique, frame, nil)
	h.fanout(RedisEnvelope{UserIDs: unique, Event: event, Data: encode(data)})
}

// PublishToChannel implements service.Publisher.
func (h *Hub) PublishToChannel(channelID, event string, data any) {
	h.publishChannel(channelID, event, data, nil)
}

// PublishToChannelExcluding broadcasts to a destination while skipping the
// listed user ids. Typing events use it so the originating socket does not
// receive its own indicator, as required by section 6.
func (h *Hub) PublishToChannelExcluding(channelID, event string, data any, exclude []string) {
	h.publishChannel(channelID, event, data, exclude)
}

func (h *Hub) publishChannel(channelID, event string, data any, exclude []string) {
	if channelID == "" {
		return
	}
	env := RedisEnvelope{Channel: channelID, Event: event, Data: encode(data), Exclude: exclude}
	h.fanout(env)

	// The local delivery mirrors what a remote replica would compute, so a
	// single instance behaves identically whether or not Redis is reachable.
	audience, err := h.audience(channelID)
	if err != nil {
		h.log.Warn().Err(err).Str("destination_id", channelID).Msg("resolve audience failed")
		return
	}
	h.deliverToUsers(audience, marshalEnvelope(event, data), exclude)
}

// deliverToUsers writes one frame to the local sockets of the listed users.
func (h *Hub) deliverToUsers(userIDs []string, frame []byte, exclude []string) {
	skip := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		if id != "" {
			skip[id] = struct{}{}
		}
	}
	for _, id := range userIDs {
		if _, found := skip[id]; found {
			continue
		}
		for _, c := range h.reg.userClients(id) {
			c.enqueue(frame)
		}
	}
}

// audience resolves the users that receive an event for a destination: the
// members of the owning server, or the participants of a direct message.
func (h *Hub) audience(destinationID string) ([]string, error) {
	return service.ChannelAudience(h.deps.DB, destinationID)
}

// fanout publishes an envelope to the other replicas. The origin instance id is
// embedded so the publishing process does not deliver its own event twice.
func (h *Hub) fanout(env RedisEnvelope) {
	env.Origin = h.instance
	payload, err := json.Marshal(env)
	if err != nil {
		h.log.Error().Err(err).Str("event", env.Event).Msg("marshal redis envelope failed")
		return
	}
	if err := h.cache.Publish(context.Background(), h.cfg.RedisChannel, payload); err != nil {
		h.log.Warn().Err(err).Str("event", env.Event).Msg("publish redis envelope failed")
	}
}

// dedupe removes empty and repeated ids while preserving order.
func dedupe(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// marshalEnvelope builds the wire frame for one event.
func marshalEnvelope(event string, data any) []byte {
	return marshalRawEnvelope(event, encode(data))
}

// marshalRawEnvelope rebuilds a frame from an already encoded payload.
func marshalRawEnvelope(event string, data json.RawMessage) []byte {
	out, err := json.Marshal(Envelope{Event: event, Data: data})
	if err != nil {
		fallback, fallbackErr := json.Marshal(Envelope{Event: EventError, Data: encode(ErrorPayload{
			Code:    "internal_error",
			Message: "event payload could not be encoded",
		})})
		if fallbackErr != nil {
			return []byte(`{"event":"` + EventError + `"}`)
		}
		return fallback
	}
	return out
}

// encode serialises a payload, returning nil when the value cannot be
// marshalled so the frame still carries the event name.
func encode(data any) json.RawMessage {
	if data == nil {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return raw
}
// Subscribe starts consuming envelopes published by the other replicas. The
// returned function stops the subscription and waits for the goroutine.
func (h *Hub) Subscribe(ctx context.Context) func() {
	pubsub := h.cache.Subscribe(ctx, h.cfg.RedisChannel)
	if _, err := pubsub.Receive(ctx); err != nil {
		h.log.Warn().Err(err).Str("channel", h.cfg.RedisChannel).Msg("subscribe to redis channel failed")
		_ = pubsub.Close()
		return func() {}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = pubsub.Close() }()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				h.handleRedisMessage([]byte(msg.Payload))
			}
		}
	}()

	return func() {
		_ = pubsub.Close()
		<-done
	}
}

// handleRedisMessage delivers a remote envelope to the local sockets, skipping
// the one this process published itself.
func (h *Hub) handleRedisMessage(payload []byte) {
	var env RedisEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		h.log.Warn().Err(err).Msg("decode redis envelope failed")
		return
	}
	if env.Origin == h.instance {
		return
	}
	frame := marshalRawEnvelope(env.Event, env.Data)

	if env.Channel != "" {
		audience, err := h.audience(env.Channel)
		if err != nil {
			h.log.Warn().Err(err).Str("destination_id", env.Channel).Msg("resolve audience failed")
			return
		}
		h.deliverToUsers(audience, frame, env.Exclude)
		return
	}
	if len(env.UserIDs) == 0 {
		return
	}
	h.deliverToUsers(dedupe(env.UserIDs), frame, env.Exclude)
}

// MarkOffline clears the Redis presence of a user. The hub calls it when the
// last socket of that user closes.
func (h *Hub) MarkOffline(userID string) {
	if userID == "" {
		return
	}
	service.NewPresenceStore(h.deps).MarkOffline(context.Background(), userID)
}