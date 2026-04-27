package npm

import (
	"sync"
	"time"
)

// CacheEntry holds cached data with an expiration time
type CacheEntry struct {
	Data      []byte
	ExpiresAt time.Time
}

// Cache is a simple in-memory cache with TTL
type Cache struct {
	mu    sync.RWMutex
	store map[string]CacheEntry
}

// NewCache creates a new Cache instance
func NewCache() *Cache {
	return &Cache{
		store: make(map[string]CacheEntry),
	}
}

// Get retrieves cached data if it exists and has not expired
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	entry, ok := c.store[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.store, key)
		c.mu.Unlock()
		return nil, false
	}
	return entry.Data, true
}

// Set stores data in the cache with a given TTL
func (c *Cache) Set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	c.store[key] = CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	c.store = make(map[string]CacheEntry)
	c.mu.Unlock()
}

// Delete removes a specific key from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
}
