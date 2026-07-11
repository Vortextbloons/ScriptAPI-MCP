package npm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// diskCache stores successful npm responses across MCP restarts. Expiry is
// deliberately checked by the caller; stale values remain available as an
// offline fallback when the registry cannot be reached.
type diskCache struct{ dir string }
type diskEntry struct {
	Data    []byte    `json:"data"`
	SavedAt time.Time `json:"saved_at"`
}

func newDiskCache() *diskCache {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil
	}
	dir = filepath.Join(dir, "script-api-helper-mcp", "npm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	return &diskCache{dir: dir}
}

func (d *diskCache) path(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(d.dir, fmt.Sprintf("%x.json", sum))
}

func (d *diskCache) Get(key string) ([]byte, time.Time, bool) {
	if d == nil {
		return nil, time.Time{}, false
	}
	b, err := os.ReadFile(d.path(key))
	if err != nil {
		return nil, time.Time{}, false
	}
	var entry diskEntry
	if json.Unmarshal(b, &entry) != nil || entry.SavedAt.IsZero() {
		return nil, time.Time{}, false
	}
	return entry.Data, entry.SavedAt, true
}

func (d *diskCache) Set(key string, data []byte) {
	if d == nil {
		return
	}
	b, err := json.Marshal(diskEntry{Data: data, SavedAt: time.Now().UTC()})
	if err != nil {
		return
	}
	_ = os.WriteFile(d.path(key), b, 0o600)
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
