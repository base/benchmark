package services

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Data      []byte
	ExpiresAt time.Time
}

// MemoryCache provides simple in-memory caching functionality
type MemoryCache struct {
	data map[string]CacheItem
	ttl  time.Duration
	l    log.Logger
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache(ttl time.Duration, l log.Logger) *MemoryCache {
	cache := &MemoryCache{
		data: make(map[string]CacheItem),
		ttl:  ttl,
		l:    l,
	}

	// Start cleanup goroutine if TTL is set
	if ttl > 0 {
		go cache.cleanup()
	}

	return cache
}

// Get retrieves data from cache
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	item, exists := c.data[key]
	if !exists || (c.ttl > 0 && time.Now().After(item.ExpiresAt)) {
		delete(c.data, key)
		return nil, false
	}
	return item.Data, true
}

// Set stores data in cache
func (c *MemoryCache) Set(key string, data []byte) {
	expiresAt := time.Now().Add(c.ttl)
	if c.ttl <= 0 {
		// No expiration for TTL <= 0
		expiresAt = time.Time{}
	}

	c.data[key] = CacheItem{
		Data:      data,
		ExpiresAt: expiresAt,
	}
}

// cleanup removes expired items periodically
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.ExpiresAt) {
				delete(c.data, key)
				c.l.Debug("Cache item expired and removed", "key", key)
			}
		}
	}
}
