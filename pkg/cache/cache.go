package cache

import (
	"context"
	"time"
)

// AgentCache defines the interface for agent caching operations
// This interface allows different implementations (Redis, in-memory, etc.)
type AgentCache interface {
	// Set stores a value with an optional TTL (time-to-live)
	// If ttl is 0, the key will not expire
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a value by key
	// Returns nil if the key doesn't exist
	Get(ctx context.Context, key string) (string, error)

	// GetBytes retrieves a value as bytes
	GetBytes(ctx context.Context, key string) ([]byte, error)

	// Delete removes a key from the cache
	Delete(ctx context.Context, key string) error

	// DeletePattern removes all keys matching a pattern (e.g., "session:*")
	DeletePattern(ctx context.Context, pattern string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// SetWithExpiry sets a key with an absolute expiration time
	SetWithExpiry(ctx context.Context, key string, value interface{}, expiryTime time.Time) error

	// Increment increments a counter key by 1
	Increment(ctx context.Context, key string) (int64, error)

	// IncrementBy increments a counter key by a specific amount
	IncrementBy(ctx context.Context, key string, value int64) (int64, error)

	// SetIfNotExists sets a value only if the key doesn't exist (returns true if set)
	SetIfNotExists(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)

	// GetTTL returns the remaining TTL for a key
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// Ping checks if the cache is available
	Ping(ctx context.Context) error

	// Close closes the cache connection
	Close() error

	// Clear removes all keys with the agent's prefix (useful for testing)
	Clear(ctx context.Context) error
}

// NoOpCache is a cache implementation that does nothing (for when Redis is disabled)
type NoOpCache struct{}

func (c *NoOpCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (c *NoOpCache) Get(ctx context.Context, key string) (string, error) {
	return "", ErrCacheKeyNotFound
}

func (c *NoOpCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return nil, ErrCacheKeyNotFound
}

func (c *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (c *NoOpCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (c *NoOpCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (c *NoOpCache) SetWithExpiry(ctx context.Context, key string, value interface{}, expiryTime time.Time) error {
	return nil
}

func (c *NoOpCache) Increment(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (c *NoOpCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (c *NoOpCache) SetIfNotExists(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return false, nil
}

func (c *NoOpCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, ErrCacheKeyNotFound
}

func (c *NoOpCache) Ping(ctx context.Context) error {
	return nil
}

func (c *NoOpCache) Close() error {
	return nil
}

func (c *NoOpCache) Clear(ctx context.Context) error {
	return nil
}
