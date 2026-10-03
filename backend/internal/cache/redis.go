// Package cache wraps go-redis with the key layout from section 8 of the
// contract: refresh tokens, presence, online markers, typing dedupe, event
// fan-out and rate limiting.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis key prefixes.
const (
	KeyPrefixRefresh  = "cordis:refresh:"
	KeyPrefixPresence = "cordis:presence:"
	KeyPrefixOnline   = "cordis:online:"
	KeyPrefixTyping   = "cordis:typing:"
	KeyPrefixRate     = "cordis:rate:"
)

// Fixed TTLs mandated by the contract.
const (
	PresenceTTL  = 5 * time.Minute
	OnlineTTL    = 5 * time.Minute
	TypingTTL    = 10 * time.Second
	RateLimitTTL = 60 * time.Second
)

// Client is the shared Redis connection with helpers for every key family.
type Client struct {
	rdb *redis.Client
}

// New dials Redis from a redis:// URL and verifies the connection.
func New(ctx context.Context, redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// Raw exposes the underlying client for advanced commands such as pipelines.
func (c *Client) Raw() *redis.Client { return c.rdb }

// Close releases the connection pool.
func (c *Client) Close() error { return c.rdb.Close() }

// Ping verifies Redis is reachable, used by the readiness probe.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// StoreRefresh persists the jti to userID mapping with the refresh TTL.
func (c *Client) StoreRefresh(jti, userID string, ttl time.Duration) error {
	return c.rdb.Set(context.Background(), KeyPrefixRefresh+jti, userID, ttl).Err()
}

// ConsumeRefresh atomically reads and deletes the jti mapping, implementing
// single use refresh rotation.
func (c *Client) ConsumeRefresh(jti string) (string, bool, error) {
	ctx := context.Background()
	userID, err := c.rdb.GetDel(ctx, KeyPrefixRefresh+jti).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, fmt.Errorf("getdel refresh token: %w", err)
	}
	return userID, true, nil
}

// DeleteRefresh removes a refresh jti.
func (c *Client) DeleteRefresh(jti string) error {
	return c.rdb.Del(context.Background(), KeyPrefixRefresh+jti).Err()
}
// Presence is the presence hash value for a single user.
type Presence struct {
	Status       string `json:"status"`
	CustomStatus string `json:"customStatus"`
	At           string `json:"at"`
}

// SetPresence writes the presence hash plus the online marker.
func (c *Client) SetPresence(ctx context.Context, userID, status, customStatus string) error {
	key := KeyPrefixPresence + userID
	now := time.Now().UTC().Format(time.RFC3339)
	pipe := c.rdb.TxPipeline()
	pipe.HSet(ctx, key, "status", status, "customStatus", customStatus, "at", now)
	pipe.Expire(ctx, key, PresenceTTL)
	if status != "offline" && status != "invisible" {
		pipe.Set(ctx, KeyPrefixOnline+userID, now, OnlineTTL)
	} else {
		pipe.Del(ctx, KeyPrefixOnline+userID)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set presence: %w", err)
	}
	return nil
}

// GetPresence reads the presence hash, falling back to offline when absent.
func (c *Client) GetPresence(ctx context.Context, userID string) (Presence, error) {
	fields, err := c.rdb.HGetAll(ctx, KeyPrefixPresence+userID).Result()
	if err != nil {
		return Presence{}, fmt.Errorf("get presence: %w", err)
	}
	if len(fields) == 0 {
		return Presence{Status: "offline"}, nil
	}
	return Presence{Status: fields["status"], CustomStatus: fields["customStatus"], At: fields["at"]}, nil
}

// DeletePresence removes presence and the online marker for a user.
func (c *Client) DeletePresence(ctx context.Context, userID string) error {
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, KeyPrefixPresence+userID)
	pipe.Del(ctx, KeyPrefixOnline+userID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("delete presence: %w", err)
	}
	return nil
}

// RefreshOnline extends the online marker TTL without touching the status,
// which the heartbeat event uses to keep long lived sessions alive.
func (c *Client) RefreshOnline(ctx context.Context, userID string) error {
	return c.rdb.Expire(ctx, KeyPrefixOnline+userID, OnlineTTL).Err()
}

// IsOnline reports whether the online marker is still present.
func (c *Client) IsOnline(ctx context.Context, userID string) (bool, error) {
	n, err := c.rdb.Exists(ctx, KeyPrefixOnline+userID).Result()
	if err != nil {
		return false, fmt.Errorf("check online: %w", err)
	}
	return n > 0, nil
}

// SetTyping records a typing indicator scored by its expiry timestamp.
func (c *Client) SetTyping(ctx context.Context, channelID, userID string) error {
	key := KeyPrefixTyping + channelID
	score := time.Now().Add(TypingTTL).UnixMilli()
	pipe := c.rdb.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(score), Member: userID})
	pipe.Expire(ctx, key, TypingTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set typing: %w", err)
	}
	return nil
}

// ClearTyping removes a typing indicator.
func (c *Client) ClearTyping(ctx context.Context, channelID, userID string) error {
	if err := c.rdb.ZRem(ctx, KeyPrefixTyping+channelID, userID).Err(); err != nil {
		return fmt.Errorf("clear typing: %w", err)
	}
	return nil
}

// Allow increments the counter for scope/id and reports whether the request
// is inside the limit, along with the number of remaining requests.
func (c *Client) Allow(ctx context.Context, scope, id string, limit int) (bool, int64, error) {
	key := KeyPrefixRate + scope + ":" + id
	pipe := c.rdb.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, RateLimitTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, fmt.Errorf("rate limit: %w", err)
	}
	n := count.Val()
	remaining := int64(limit) - n
	if remaining < 0 {
		remaining = 0
	}
	return n <= int64(limit), remaining, nil
}

// Subscribe opens a subscription to the given channel.
func (c *Client) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channel)
}

// Publish sends a raw envelope to every replica listening on the channel.
func (c *Client) Publish(ctx context.Context, channel string, payload []byte) error {
	if err := c.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("publish %s: %w", channel, err)
	}
	return nil
}