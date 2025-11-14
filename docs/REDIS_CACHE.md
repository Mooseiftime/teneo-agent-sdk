# Agent Cache (Redis)

The Teneo Agent SDK includes a built-in Redis caching framework that allows agents to persist data across restarts. This is particularly useful for maintaining state, caching expensive computations, and enabling distributed agent deployments.

## Features

- **Easy to use**: Just add Redis connection URL to environment variables
- **Automatic key prefixing**: All keys are automatically prefixed with agent name to avoid collisions
- **Graceful degradation**: If Redis is unavailable, the agent continues to work without caching
- **Type-safe interface**: Clean Go interface for all cache operations
- **TTL support**: Set expiration times for cached data
- **Pattern-based deletion**: Delete multiple keys matching a pattern
- **Distributed rate limiting**: Share rate limit state across multiple agent instances

## Quick Start

### 1. Install Redis

**Using Docker:**
```bash
docker run -d -p 6379:6379 redis:latest
```

**Using Homebrew (macOS):**
```bash
brew install redis
brew services start redis
```

### 2. Configure Your Agent

Add these environment variables to your `.env` file:

```bash
# Enable Redis cache
REDIS_ENABLED=true

# Redis connection (default: localhost:6379)
REDIS_ADDRESS=localhost:6379

# Optional: Redis password
REDIS_PASSWORD=

# Optional: Redis database number (0-15, default: 0)
REDIS_DB=0

# Optional: Custom key prefix (default: teneo:agent:<agent_name>:)
REDIS_KEY_PREFIX=myagent:
```

Alternatively, you can use `REDIS_URL` instead of `REDIS_ADDRESS` for convenience.

### 3. Access the Cache in Your Agent

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/TeneoProtocolAI/teneo-agent-sdk/pkg/agent"
)

type MyAgent struct {
    cache cache.AgentCache // Store reference to cache
}

func (a *MyAgent) Initialize(ctx context.Context, config interface{}) error {
    // Get the enhanced agent instance and store cache reference
    if enhancedAgent, ok := config.(*agent.EnhancedAgent); ok {
        a.cache = enhancedAgent.GetCache()
    }
    return nil
}

func (a *MyAgent) ProcessTask(ctx context.Context, task string) (string, error) {
    // Check if result is in cache
    cacheKey := "task:" + task
    cached, err := a.cache.Get(ctx, cacheKey)
    if err == nil {
        log.Printf("Cache hit for task: %s", task)
        return cached, nil
    }

    // Process task
    result := "Processed: " + task

    // Store result in cache for 1 hour
    if err := a.cache.Set(ctx, cacheKey, result, 1*time.Hour); err != nil {
        log.Printf("Warning: failed to cache result: %v", err)
    }

    return result, nil
}
```

## Cache Interface

The `AgentCache` interface provides the following methods:

### Basic Operations

```go
// Set stores a value with optional TTL
Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

// Get retrieves a string value
Get(ctx context.Context, key string) (string, error)

// GetBytes retrieves a byte array value
GetBytes(ctx context.Context, key string) ([]byte, error)

// Delete removes a key
Delete(ctx context.Context, key string) error

// Exists checks if a key exists
Exists(ctx context.Context, key string) (bool, error)
```

### Advanced Operations

```go
// DeletePattern removes all keys matching a pattern
DeletePattern(ctx context.Context, pattern string) error

// SetWithExpiry sets a key with absolute expiration time
SetWithExpiry(ctx context.Context, key string, value interface{}, expiryTime time.Time) error

// SetIfNotExists sets only if key doesn't exist (returns true if set)
SetIfNotExists(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)

// GetTTL returns remaining TTL for a key
GetTTL(ctx context.Context, key string) (time.Duration, error)

// Increment increments a counter by 1
Increment(ctx context.Context, key string) (int64, error)

// IncrementBy increments a counter by specific amount
IncrementBy(ctx context.Context, key string, value int64) (int64, error)

// Ping checks if cache is available
Ping(ctx context.Context) error

// Clear removes all keys with agent's prefix
Clear(ctx context.Context) error

// Close closes the cache connection
Close() error
```

## Usage Examples

### Example 1: Caching API Responses

```go
func (a *MyAgent) fetchData(ctx context.Context, userID string) (string, error) {
    cacheKey := "user:" + userID

    // Try to get from cache
    cached, err := a.cache.Get(ctx, cacheKey)
    if err == nil {
        return cached, nil
    }

    // Fetch from API
    data := fetchFromAPI(userID)

    // Cache for 5 minutes
    a.cache.Set(ctx, cacheKey, data, 5*time.Minute)

    return data, nil
}
```

### Example 2: Session Management

```go
func (a *MyAgent) createSession(ctx context.Context, sessionID string, data interface{}) error {
    // Store session data with 24 hour expiration
    return a.cache.Set(ctx, "session:"+sessionID, data, 24*time.Hour)
}

func (a *MyAgent) getSession(ctx context.Context, sessionID string) (string, error) {
    return a.cache.Get(ctx, "session:"+sessionID)
}

func (a *MyAgent) deleteSession(ctx context.Context, sessionID string) error {
    return a.cache.Delete(ctx, "session:"+sessionID)
}
```

### Example 3: Rate Limiting

```go
func (a *MyAgent) checkRateLimit(ctx context.Context, userID string) (bool, error) {
    key := "ratelimit:" + userID

    // Increment counter
    count, err := a.cache.Increment(ctx, key)
    if err != nil {
        return false, err
    }

    // Set expiration on first increment
    if count == 1 {
        a.cache.Set(ctx, key, count, 1*time.Minute)
    }

    // Allow up to 10 requests per minute
    return count <= 10, nil
}
```

### Example 4: Distributed Lock

```go
func (a *MyAgent) acquireLock(ctx context.Context, resourceID string, ttl time.Duration) (bool, error) {
    lockKey := "lock:" + resourceID

    // Try to acquire lock (set only if not exists)
    acquired, err := a.cache.SetIfNotExists(ctx, lockKey, "locked", ttl)
    if err != nil {
        return false, err
    }

    return acquired, nil
}

func (a *MyAgent) releaseLock(ctx context.Context, resourceID string) error {
    return a.cache.Delete(ctx, "lock:"+resourceID)
}
```

### Example 5: Complex Data with JSON

```go
type UserProfile struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Score int    `json:"score"`
}

func (a *MyAgent) saveProfile(ctx context.Context, userID string, profile UserProfile) error {
    // The cache automatically marshals structs to JSON
    return a.cache.Set(ctx, "profile:"+userID, profile, 24*time.Hour)
}

func (a *MyAgent) loadProfile(ctx context.Context, userID string) (*UserProfile, error) {
    data, err := a.cache.GetBytes(ctx, "profile:"+userID)
    if err != nil {
        return nil, err
    }

    var profile UserProfile
    if err := json.Unmarshal(data, &profile); err != nil {
        return nil, err
    }

    return &profile, nil
}
```

### Example 6: Pattern-Based Cleanup

```go
func (a *MyAgent) clearUserData(ctx context.Context, userID string) error {
    // Delete all keys matching the pattern
    return a.cache.DeletePattern(ctx, "user:"+userID+":*")
}

func (a *MyAgent) clearAllSessions(ctx context.Context) error {
    // Delete all session keys
    return a.cache.DeletePattern(ctx, "session:*")
}
```

## Key Naming Conventions

All keys are automatically prefixed with your agent's name. For example, if your agent is named "My Agent" and you set a key "task:123", the actual Redis key will be:

```
teneo:agent:my_agent:task:123
```

This prevents key collisions when multiple agents share the same Redis instance.

### Recommended Key Patterns

- **Tasks**: `task:<task_id>` or `task:<hash>`
- **Sessions**: `session:<session_id>`
- **Users**: `user:<user_id>:<data_type>`
- **Rate limits**: `ratelimit:<identifier>:<timewindow>`
- **Locks**: `lock:<resource_id>`
- **Temporary data**: `temp:<identifier>`

## Configuration Options

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ENABLED` | Enable Redis caching | `false` |
| `REDIS_ADDRESS` | Redis server address | `localhost:6379` |
| `REDIS_URL` | Alternative to REDIS_ADDRESS | - |
| `REDIS_PASSWORD` | Redis password | `` (empty) |
| `REDIS_DB` | Redis database number (0-15) | `0` |
| `REDIS_KEY_PREFIX` | Custom key prefix | `teneo:agent:<name>:` |

### Programmatic Configuration

You can also configure Redis programmatically:

```go
config := agent.DefaultConfig()
config.RedisEnabled = true
config.RedisAddress = "redis.example.com:6379"
config.RedisPassword = "secret"
config.RedisDB = 1
config.RedisKeyPrefix = "myagent:"
```

## Error Handling

The cache is designed to be fault-tolerant. If Redis is unavailable:

1. The agent will log a warning but continue to run
2. A no-op cache is used (all operations succeed but do nothing)
3. Your agent logic should handle cache misses gracefully

```go
result, err := a.cache.Get(ctx, key)
if err == cache.ErrCacheKeyNotFound {
    // Key not found - compute the value
    result = computeValue()
} else if err != nil {
    // Other error - log but continue
    log.Printf("Cache error: %v", err)
    result = computeValue()
}
```

## Performance Considerations

- **Connection pooling**: The cache uses connection pooling by default (10 connections)
- **Timeouts**: Operations have sensible timeouts (3s read, 3s write)
- **Retries**: Failed operations are retried up to 3 times
- **TTL**: Always set TTL for temporary data to prevent memory bloat

## Testing

For testing, you can disable Redis or use a test Redis instance:

```go
// Disable cache for tests
config.RedisEnabled = false

// Or use a separate Redis DB
config.RedisDB = 15 // Use DB 15 for testing
```

You can also clear all cache data:

```go
// Clear all keys with agent's prefix
agent.GetCache().Clear(ctx)
```

## Advanced: Direct Redis Access

If you need advanced Redis features not covered by the interface, you can access the underlying Redis client:

```go
import "github.com/TeneoProtocolAI/teneo-agent-sdk/pkg/cache"

if redisCache, ok := agent.GetCache().(*cache.RedisCache); ok {
    client := redisCache.GetClient()
    // Use client for advanced operations
    client.HSet(ctx, "myhash", "field", "value")
}
```

## Troubleshooting

### Connection Errors

If you see connection errors:

1. Verify Redis is running: `redis-cli ping` (should return "PONG")
2. Check the Redis address is correct
3. Verify firewall rules allow connections to Redis
4. Check Redis logs for errors

### Permission Errors

If you see permission errors:

1. Verify the Redis password is correct
2. Check Redis ACL settings (if using Redis 6+)

### Memory Issues

If Redis is using too much memory:

1. Set TTL on all keys
2. Configure Redis maxmemory policy
3. Use `redis-cli MEMORY STATS` to analyze usage
4. Consider using key patterns to delete old data

## Production Recommendations

1. **Use persistent Redis**: Configure Redis with AOF or RDB persistence
2. **Set maxmemory**: Configure Redis maxmemory and eviction policy
3. **Monitor**: Use Redis INFO command to monitor performance
4. **Backup**: Regular backups of Redis data
5. **Secure**: Use strong passwords and TLS for production
6. **Scale**: Consider Redis Cluster for horizontal scaling

## See Also

- [Redis Documentation](https://redis.io/documentation)
- [go-redis Documentation](https://redis.uptrace.dev/)
