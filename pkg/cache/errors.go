package cache

import "errors"

var (
	// ErrCacheKeyNotFound is returned when a key is not found in the cache
	ErrCacheKeyNotFound = errors.New("cache key not found")

	// ErrCacheConnectionFailed is returned when the cache connection fails
	ErrCacheConnectionFailed = errors.New("cache connection failed")

	// ErrCacheOperationFailed is returned when a cache operation fails
	ErrCacheOperationFailed = errors.New("cache operation failed")
)
