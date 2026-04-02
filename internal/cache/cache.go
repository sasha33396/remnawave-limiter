package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/remnawave/limiter/internal/api"
)

const (
	prefixUser     = "user:"
	prefixCooldown = "cooldown:"
	keyWhitelist   = "whitelist"
	keyRestoreQ    = "restore:queue"
)

// Cache wraps a Redis client for the centralized limiter.
type Cache struct {
	client *redis.Client
}

// New parses the Redis URL and creates a new Cache. Returns an error if the URL
// is invalid or the client cannot be created.
func New(redisURL string) (*Cache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Cache{client: redis.NewClient(opts)}, nil
}

// Ping checks the Redis connection.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the underlying Redis client.
func (c *Cache) Close() error {
	return c.client.Close()
}

// --- User cache ---

// SetUser stores a CachedUser as JSON with the given TTL.
func (c *Cache) SetUser(ctx context.Context, userID string, user *api.CachedUser, ttl time.Duration) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}
	return c.client.Set(ctx, prefixUser+userID, data, ttl).Err()
}

// GetUser retrieves a CachedUser by userID. Returns (nil, nil) if not found.
func (c *Cache) GetUser(ctx context.Context, userID string) (*api.CachedUser, error) {
	data, err := c.client.Get(ctx, prefixUser+userID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	var user api.CachedUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal user: %w", err)
	}
	return &user, nil
}

// --- Cooldowns ---

// SetCooldown sets a cooldown key for the given user with the specified TTL.
func (c *Cache) SetCooldown(ctx context.Context, userID string, ttl time.Duration) error {
	return c.client.Set(ctx, prefixCooldown+userID, "1", ttl).Err()
}

// IsCooldownActive checks whether a cooldown is active for the given user.
func (c *Cache) IsCooldownActive(ctx context.Context, userID string) (bool, error) {
	_, err := c.client.Get(ctx, prefixCooldown+userID).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get cooldown: %w", err)
	}
	return true, nil
}

// --- Whitelist ---

// AddToWhitelist adds a user ID to the whitelist set.
func (c *Cache) AddToWhitelist(ctx context.Context, userID string) error {
	return c.client.SAdd(ctx, keyWhitelist, userID).Err()
}

// RemoveFromWhitelist removes a user ID from the whitelist set.
func (c *Cache) RemoveFromWhitelist(ctx context.Context, userID string) error {
	return c.client.SRem(ctx, keyWhitelist, userID).Err()
}

// IsWhitelisted checks whether a user ID is in the whitelist set.
func (c *Cache) IsWhitelisted(ctx context.Context, userID string) (bool, error) {
	return c.client.SIsMember(ctx, keyWhitelist, userID).Result()
}

// InitWhitelist replaces the whitelist set with the provided user IDs.
// Uses a pipeline to atomically delete and re-populate.
func (c *Cache) InitWhitelist(ctx context.Context, userIDs []string) error {
	pipe := c.client.Pipeline()
	pipe.Del(ctx, keyWhitelist)
	if len(userIDs) > 0 {
		members := make([]interface{}, len(userIDs))
		for i, id := range userIDs {
			members[i] = id
		}
		pipe.SAdd(ctx, keyWhitelist, members...)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// --- Restore timers (auto mode) ---

// SetRestoreTimer adds a UUID to the restore queue sorted set with a score equal
// to the Unix timestamp when the timer expires.
func (c *Cache) SetRestoreTimer(ctx context.Context, uuid string, duration time.Duration) error {
	expiry := float64(time.Now().Add(duration).Unix())
	return c.client.ZAdd(ctx, keyRestoreQ, redis.Z{
		Score:  expiry,
		Member: uuid,
	}).Err()
}

// GetExpiredRestoreTimers returns UUIDs whose timers have expired (score <= now)
// and removes them from the sorted set atomically using a Lua script.
func (c *Cache) GetExpiredRestoreTimers(ctx context.Context) ([]string, error) {
	now := fmt.Sprintf("%d", time.Now().Unix())

	// Lua script: get expired members then remove them atomically
	script := redis.NewScript(`
		local expired = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
		if #expired > 0 then
			redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
		end
		return expired
	`)

	result, err := script.Run(ctx, c.client, []string{keyRestoreQ}, now).StringSlice()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get expired restore timers: %w", err)
	}
	return result, nil
}
