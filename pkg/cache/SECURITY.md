# Redis Cache Security

This document outlines the security measures implemented in the Redis cache package and best practices for secure usage.

## Security Features

### 1. Input Validation

All cache keys are validated before use to prevent security vulnerabilities:

#### Key Validation (`validateKey`)

- **Empty key check**: Keys cannot be empty
- **Length limit**: Maximum 1024 characters to prevent memory exhaustion
- **UTF-8 validation**: Keys must be valid UTF-8 strings
- **Control character filtering**: Prevents newlines (`\n`), carriage returns (`\r`), tabs (`\t`), and null bytes (`\x00`)

```go
// Example: These keys will be rejected
cache.Set(ctx, "", "value", ttl)          // Error: empty key
cache.Set(ctx, "key\nwith\nnewlines", ...) // Error: invalid characters
cache.Set(ctx, strings.Repeat("a", 2000), ...) // Error: key too long
```

### 2. Pattern Sanitization

The `DeletePattern` method includes comprehensive security measures:

#### Pattern Sanitization (`sanitizePattern`)

Prevents prefix escape attacks by removing:
- Path traversal attempts: `../`, `/..`
- Leading slashes and dots: `/`, `.`
- Null bytes: `\x00`

#### Double-Check Validation

After scanning, all returned keys are verified to start with the agent's prefix:

```go
// Only keys starting with our prefix are included
for _, key := range scanKeys {
    if strings.HasPrefix(key, r.keyPrefix) {
        keys = append(keys, key)
    }
}
```

This prevents malicious patterns like `../../*` from accessing keys outside the agent's namespace.

### 3. TTL Validation

- **Negative TTL prevention**: TTL values cannot be negative
- **Expiry time validation**: `SetWithExpiry` ensures expiry time is in the future

```go
// These will be rejected
cache.Set(ctx, "key", "value", -5*time.Second)  // Error: negative TTL
cache.SetWithExpiry(ctx, "key", "value", time.Now().Add(-1*time.Hour)) // Error: past expiry
```

## Security Best Practices

### For Developers Using the Cache

#### 1. **Use Strong Redis Passwords**

```go
config := &cache.RedisConfig{
    Address:  "redis.example.com:6379",
    Password: os.Getenv("REDIS_PASSWORD"), // Use environment variables
    DB:       0,
}
```

#### 2. **Enable TLS for Production**

For production deployments, configure Redis with TLS:

```bash
# Redis with TLS
redis-server --tls-port 6379 \
    --tls-cert-file /path/to/cert.pem \
    --tls-key-file /path/to/key.pem \
    --tls-ca-cert-file /path/to/ca.pem
```

Note: The current SDK uses the standard go-redis client. For TLS support, you may need to access the underlying client:

```go
redisCache.GetClient().Options().TLSConfig = &tls.Config{...}
```

#### 3. **Validate User Input Before Caching**

Never cache unsanitized user input directly:

```go
// BAD: User input directly used as key
userInput := request.GetParameter("key")
cache.Set(ctx, userInput, value, ttl)

// GOOD: Sanitize and validate first
func sanitizeUserKey(input string) (string, error) {
    if len(input) > 100 {
        return "", errors.New("key too long")
    }
    // Remove special characters
    safe := strings.Map(func(r rune) rune {
        if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
            return r
        }
        return -1
    }, input)
    return safe, nil
}

safeKey, err := sanitizeUserKey(userInput)
if err != nil {
    return err
}
cache.Set(ctx, "user:"+safeKey, value, ttl)
```

#### 4. **Use Namespaced Keys**

Always use a consistent key naming scheme:

```go
// Good key patterns
cache.Set(ctx, "user:"+userID, userData, ttl)
cache.Set(ctx, "session:"+sessionID, sessionData, ttl)
cache.Set(ctx, "task:"+taskHash, result, ttl)

// Avoid generic keys that could collide
cache.Set(ctx, "data", value, ttl) // BAD: too generic
```

#### 5. **Set Appropriate TTLs**

Always set TTL for temporary data to prevent memory bloat:

```go
// Temporary data - use TTL
cache.Set(ctx, "temp:"+id, data, 5*time.Minute)

// Long-term data - use 0 for no expiration (use sparingly)
cache.Set(ctx, "config:version", version, 0)
```

#### 6. **Handle Errors Gracefully**

Don't expose internal errors to users:

```go
result, err := cache.Get(ctx, key)
if err != nil {
    if err == cache.ErrCacheKeyNotFound {
        // Expected error - handle normally
        return computeValue()
    }
    // Log internal error but return generic error to user
    log.Printf("Cache error: %v", err)
    return "An error occurred"
}
```

### For Production Deployments

#### 1. **Network Security**

```bash
# Bind Redis to specific interface
bind 127.0.0.1

# Or use firewall rules
iptables -A INPUT -p tcp --dport 6379 -s trusted-ip -j ACCEPT
iptables -A INPUT -p tcp --dport 6379 -j DROP
```

#### 2. **Redis ACL (Redis 6+)**

Create dedicated users with limited permissions:

```redis
# Create user for agent
ACL SETUSER agentuser on >strongpassword ~teneo:agent:* +@all -@dangerous

# Deny specific commands
ACL SETUSER agentuser -flushall -flushdb -config
```

#### 3. **Monitor Access Patterns**

```bash
# Monitor Redis commands
redis-cli MONITOR | grep -E "(KEYS|SCAN|FLUSHDB)"

# Check for suspicious patterns
redis-cli INFO stats | grep -E "(keyspace|commands)"
```

#### 4. **Regular Security Audits**

- Review Redis logs for unusual activity
- Check for keys outside expected patterns
- Monitor memory usage for DoS attacks
- Audit connection sources

## Vulnerability Mitigations

### 1. ✅ Key Collision Attack (Mitigated)

**Attack**: Malicious pattern to delete keys outside agent namespace

```go
// Attempted attack
cache.DeletePattern(ctx, "../../other_agent:*")
```

**Mitigation**:
- Pattern sanitization removes `../` and `/..`
- Double-check validation ensures only prefixed keys are deleted
- Keys are always validated before any operation

### 2. ✅ Control Character Injection (Mitigated)

**Attack**: Inject newlines or control characters to break logging or protocols

```go
// Attempted attack
cache.Set(ctx, "key\nINFO REPLICATION\n", "value", ttl)
```

**Mitigation**:
- validateKey() rejects keys with `\n`, `\r`, `\t`, `\x00`
- All keys are validated before use

### 3. ✅ Key Length DoS (Mitigated)

**Attack**: Extremely long keys to exhaust memory

```go
// Attempted attack
cache.Set(ctx, strings.Repeat("a", 1000000), "value", ttl)
```

**Mitigation**:
- Maximum key length of 1024 characters enforced
- Validation occurs before Redis operation

### 4. ✅ Negative TTL Exploitation (Mitigated)

**Attack**: Negative TTL to bypass expiration

```go
// Attempted attack
cache.Set(ctx, "key", "value", -1*time.Hour)
```

**Mitigation**:
- TTL values are validated to be non-negative
- SetWithExpiry validates future expiry times

## Security Testing

### Test Invalid Keys

```go
testCases := []string{
    "",                      // Empty
    "key\nwith\nnewlines",  // Control chars
    strings.Repeat("a", 2000), // Too long
    "key\x00with\x00nulls", // Null bytes
}

for _, key := range testCases {
    err := cache.Set(ctx, key, "value", time.Minute)
    if err == nil {
        t.Errorf("Expected error for invalid key: %s", key)
    }
}
```

### Test Pattern Escape

```go
// Should not delete keys outside agent prefix
cache.DeletePattern(ctx, "../../../*")
cache.DeletePattern(ctx, "/..")
cache.DeletePattern(ctx, "\x00*")

// Verify keys outside prefix are still present
exists, _ := otherCache.Exists(ctx, "should_still_exist")
if !exists {
    t.Error("Pattern escape vulnerability!")
}
```

## Reporting Security Issues

If you discover a security vulnerability in the cache implementation:

1. **Do not** open a public GitHub issue
2. Email security concerns to: [security contact]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

## Security Checklist

Before deploying to production:

- [ ] Redis password is set and strong
- [ ] Redis is not exposed to public internet
- [ ] Firewall rules restrict Redis access
- [ ] TLS is enabled for Redis connections
- [ ] Redis ACLs are configured (Redis 6+)
- [ ] Keys use consistent naming patterns
- [ ] User input is sanitized before caching
- [ ] Appropriate TTLs are set
- [ ] Error handling doesn't leak sensitive info
- [ ] Monitoring is in place for suspicious activity
- [ ] Regular security audits are scheduled

## Version History

- **v1.0.0** (January 2025)
  - Initial implementation with comprehensive security features
  - Input validation for all operations
  - Pattern sanitization for DeletePattern
  - Double-check validation for scanned keys
  - TTL validation

## References

- [Redis Security](https://redis.io/docs/management/security/)
- [OWASP Redis Security](https://cheatsheetseries.owasp.org/cheatsheets/Redis_Security_Cheat_Sheet.html)
- [go-redis Security Best Practices](https://redis.uptrace.dev/guide/security.html)
