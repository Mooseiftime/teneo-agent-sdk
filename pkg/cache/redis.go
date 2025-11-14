package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements the AgentCache interface using Redis
type RedisCache struct {
	client    *redis.Client
	keyPrefix string // Prefix for all keys to avoid collisions
}

// RedisConfig holds the configuration for Redis connection
type RedisConfig struct {
	// Address is the Redis server address (e.g., "localhost:6379")
	Address string

	// Password is the Redis password (empty if no password)
	Password string

	// DB is the Redis database number (0-15)
	DB int

	// KeyPrefix is prepended to all cache keys (e.g., "agent:myagent:")
	KeyPrefix string

	// MaxRetries is the maximum number of retries before giving up
	MaxRetries int

	// DialTimeout is the timeout for establishing new connections
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes
	WriteTimeout time.Duration

	// PoolSize is the maximum number of socket connections
	PoolSize int
}

// DefaultRedisConfig returns a default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Address:      "localhost:6379",
		Password:     "",
		DB:           0,
		KeyPrefix:    "teneo:agent:",
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	}
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *RedisConfig) (*RedisCache, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.DB,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:    client,
		keyPrefix: config.KeyPrefix,
	}, nil
}

// prefixKey adds the prefix to a key
func (r *RedisCache) prefixKey(key string) string {
	return r.keyPrefix + key
}

// validateKey validates a cache key for security and correctness
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if len(key) > 1024 {
		return fmt.Errorf("key too long (max 1024 characters)")
	}

	if !utf8.ValidString(key) {
		return fmt.Errorf("key must be valid UTF-8")
	}

	// Prevent control characters and newlines
	if strings.ContainsAny(key, "\n\r\t\x00") {
		return fmt.Errorf("key contains invalid characters")
	}

	return nil
}

// sanitizePattern sanitizes a pattern to prevent prefix escape attacks
func sanitizePattern(pattern string) string {
	// Remove any attempt to escape the prefix boundary
	pattern = strings.ReplaceAll(pattern, "../", "")
	pattern = strings.ReplaceAll(pattern, "/..", "")
	pattern = strings.TrimPrefix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, ".")

	// Remove any null bytes
	pattern = strings.ReplaceAll(pattern, "\x00", "")

	return pattern
}

// Set stores a value with an optional TTL
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Validate input
	if err := validateKey(key); err != nil {
		return err
	}

	if ttl < 0 {
		return fmt.Errorf("TTL cannot be negative")
	}

	prefixedKey := r.prefixKey(key)

	// Convert value to string or bytes
	var data interface{}
	switch v := value.(type) {
	case string:
		data = v
	case []byte:
		data = v
	default:
		// Marshal to JSON for complex types
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		data = jsonData
	}

	if err := r.client.Set(ctx, prefixedKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Get retrieves a value by key
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return "", err
	}

	prefixedKey := r.prefixKey(key)

	result, err := r.client.Get(ctx, prefixedKey).Result()
	if err == redis.Nil {
		return "", ErrCacheKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return result, nil
}

// GetBytes retrieves a value as bytes
func (r *RedisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return nil, err
	}

	prefixedKey := r.prefixKey(key)

	result, err := r.client.Get(ctx, prefixedKey).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return result, nil
}

// Delete removes a key from the cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	// Validate input
	if err := validateKey(key); err != nil {
		return err
	}

	prefixedKey := r.prefixKey(key)

	if err := r.client.Del(ctx, prefixedKey).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// DeletePattern removes all keys matching a pattern
// SECURITY: Pattern is validated and sanitized to prevent escaping the key prefix
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	// Validate pattern input
	if err := validateKey(pattern); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	// Sanitize pattern to prevent prefix escape
	sanitizedPattern := sanitizePattern(pattern)

	// Apply prefix AFTER sanitization to prevent escape
	prefixedPattern := r.prefixKey(sanitizedPattern)

	// Use SCAN to find matching keys - only scans within our prefix
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = r.client.Scan(ctx, cursor, prefixedPattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys with pattern %s: %w", pattern, err)
		}

		// Double-check: only include keys that actually start with our prefix
		for _, key := range scanKeys {
			if strings.HasPrefix(key, r.keyPrefix) {
				keys = append(keys, key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	// Delete all matching keys
	if len(keys) > 0 {
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys with pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return false, err
	}

	prefixedKey := r.prefixKey(key)

	result, err := r.client.Exists(ctx, prefixedKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}

	return result > 0, nil
}

// SetWithExpiry sets a key with an absolute expiration time
func (r *RedisCache) SetWithExpiry(ctx context.Context, key string, value interface{}, expiryTime time.Time) error {
	// Validate input
	if err := validateKey(key); err != nil {
		return err
	}

	ttl := time.Until(expiryTime)
	if ttl <= 0 {
		return fmt.Errorf("expiry time must be in the future")
	}

	return r.Set(ctx, key, value, ttl)
}

// Increment increments a counter key by 1
func (r *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return 0, err
	}

	prefixedKey := r.prefixKey(key)

	result, err := r.client.Incr(ctx, prefixedKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}

	return result, nil
}

// IncrementBy increments a counter key by a specific amount
func (r *RedisCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return 0, err
	}

	prefixedKey := r.prefixKey(key)

	result, err := r.client.IncrBy(ctx, prefixedKey, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s by %d: %w", key, value, err)
	}

	return result, nil
}

// SetIfNotExists sets a value only if the key doesn't exist
func (r *RedisCache) SetIfNotExists(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return false, err
	}

	if ttl < 0 {
		return false, fmt.Errorf("TTL cannot be negative")
	}

	prefixedKey := r.prefixKey(key)

	// Convert value to string or bytes
	var data interface{}
	switch v := value.(type) {
	case string:
		data = v
	case []byte:
		data = v
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
		data = jsonData
	}

	result, err := r.client.SetNX(ctx, prefixedKey, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set key %s if not exists: %w", key, err)
	}

	return result, nil
}

// GetTTL returns the remaining TTL for a key
func (r *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	// Validate input
	if err := validateKey(key); err != nil {
		return 0, err
	}

	prefixedKey := r.prefixKey(key)

	ttl, err := r.client.TTL(ctx, prefixedKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	if ttl < 0 {
		return 0, ErrCacheKeyNotFound
	}

	return ttl, nil
}

// Ping checks if the cache is available
func (r *RedisCache) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis ping failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Clear removes all keys with the agent's prefix
func (r *RedisCache) Clear(ctx context.Context) error {
	return r.DeletePattern(ctx, "*")
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisCache) GetClient() *redis.Client {
	return r.client
}
