// Package redis is a Redis-backed implementation of cache.Cache.
//
// Values are JSON-encoded so that the same data round-trips identically
// across the in-memory and Redis backends.
//
// Example:
//
//	rc := redis.New(redis.Config{Addr: "localhost:6379"})
//	defer rc.Close()
//
//	if err := rc.Set(ctx, "user:42", user, 5*time.Minute); err != nil { ... }
//	var u User
//	if err := rc.Get(ctx, "user:42", &u); err != nil { ... }
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	frameworkCache "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cache"
	goredis "github.com/redis/go-redis/v9"
)

// Config controls the Redis client connection.
type Config struct {
	Addr      string // host:port (e.g. localhost:6379)
	Username  string
	Password  string
	DB        int
	KeyPrefix string        // optional prefix prepended to every key
	Timeout   time.Duration // command timeout, default 5s
}

// Cache is a Redis-backed cache.Cache implementation.
type Cache struct {
	client *goredis.Client
	prefix string
	owns   bool // true when this Cache owns the client lifecycle
}

// New constructs a Cache with a fresh redis client built from cfg.
// Close releases the connection pool.
func New(cfg Config) *Cache {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	client := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	})
	return &Cache{client: client, prefix: cfg.KeyPrefix, owns: true}
}

// NewFromClient adapts an existing redis client. The caller retains
// ownership of the client; Close on the returned Cache is a no-op.
func NewFromClient(client *goredis.Client, keyPrefix string) *Cache {
	return &Cache{client: client, prefix: keyPrefix, owns: false}
}

// Close releases the connection pool when this Cache owns it.
func (c *Cache) Close() error {
	if c == nil || c.client == nil || !c.owns {
		return nil
	}
	return c.client.Close()
}

// Get unmarshals the value for key into dest. Returns cache.ErrNotFound on miss.
func (c *Cache) Get(ctx context.Context, key string, dest any) error {
	raw, err := c.client.Get(ctx, c.k(key)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return frameworkCache.ErrNotFound
		}
		return fmt.Errorf("cache/redis: get %q: %w", key, err)
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("cache/redis: unmarshal %q: %w", key, err)
	}
	return nil
}

// Set serializes value as JSON and stores it. ttl <= 0 means no expiration.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache/redis: marshal %q: %w", key, err)
	}
	if ttl < 0 {
		ttl = 0
	}
	if err := c.client.Set(ctx, c.k(key), data, ttl).Err(); err != nil {
		return fmt.Errorf("cache/redis: set %q: %w", key, err)
	}
	return nil
}

// Delete removes key. A miss is not an error.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, c.k(key)).Err(); err != nil {
		return fmt.Errorf("cache/redis: del %q: %w", key, err)
	}
	return nil
}

// Exists reports whether key is present.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.k(key)).Result()
	if err != nil {
		return false, fmt.Errorf("cache/redis: exists %q: %w", key, err)
	}
	return n > 0, nil
}

// Flush removes all keys in the current DB. Use with care; this is FLUSHDB,
// not a prefix-scoped delete.
func (c *Cache) Flush(ctx context.Context) error {
	if err := c.client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("cache/redis: flushdb: %w", err)
	}
	return nil
}

func (c *Cache) k(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + key
}

// Compile-time check that Cache satisfies the framework Cache interface.
var _ frameworkCache.Cache = (*Cache)(nil)
