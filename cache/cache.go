// Package cache defines a generic cache interface and an in-memory
// implementation suitable for tests and short-lived processes.
//
// For production, use the Redis-backed implementation in
// github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cache/redis.
//
// Values are serialized as JSON. Get takes a destination pointer that the
// stored bytes are unmarshalled into. A miss is signaled by ErrNotFound.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// ErrNotFound is returned by Get when the key is absent or expired.
var ErrNotFound = errors.New("cache: key not found")

// Cache is the operation contract every backend implements.
type Cache interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Flush(ctx context.Context) error
}

// Memory is an in-memory Cache implementation. TTL is enforced lazily on
// Get / Exists. Concurrent-safe.
//
// Intended for tests and development. Long-running processes should use
// the Redis backend or invoke Sweep periodically to reclaim expired keys.
type Memory struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
}

type memoryEntry struct {
	data   []byte
	expiry time.Time // zero = never expires
}

// NewMemory returns an empty Memory cache.
func NewMemory() *Memory {
	return &Memory{entries: make(map[string]memoryEntry)}
}

// Get unmarshals the cached bytes for key into dest. Returns ErrNotFound
// if the key is absent or expired.
func (m *Memory) Get(ctx context.Context, key string, dest any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if !ok || expired(entry) {
		return ErrNotFound
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(entry.data, dest); err != nil {
		return err
	}
	return nil
}

// Set serializes value as JSON and stores it under key. ttl <= 0 means
// the entry never expires.
func (m *Memory) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	entry := memoryEntry{data: data}
	if ttl > 0 {
		entry.expiry = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.entries[key] = entry
	m.mu.Unlock()
	return nil
}

// Delete removes key. A miss is not an error.
func (m *Memory) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()
	return nil
}

// Exists reports whether key is present and unexpired.
func (m *Memory) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()
	if !ok || expired(entry) {
		return false, nil
	}
	return true, nil
}

// Flush removes all entries.
func (m *Memory) Flush(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	m.entries = make(map[string]memoryEntry)
	m.mu.Unlock()
	return nil
}

// Sweep removes all expired entries. Useful for long-running processes that
// use Memory and want to reclaim space without waiting for the next Get.
func (m *Memory) Sweep() int {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	var removed int
	for k, e := range m.entries {
		if !e.expiry.IsZero() && e.expiry.Before(now) {
			delete(m.entries, k)
			removed++
		}
	}
	return removed
}

func expired(e memoryEntry) bool {
	return !e.expiry.IsZero() && e.expiry.Before(time.Now())
}

// Compile-time check that Memory satisfies Cache.
var _ Cache = (*Memory)(nil)
