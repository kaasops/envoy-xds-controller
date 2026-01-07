package utils

import (
	"container/list"
	"sync"
	"time"
)

// LRUCacheItem represents a single item in the LRU cache
type LRUCacheItem struct {
	Key       string
	Value     interface{}
	Timestamp time.Time
}

// LRUCache provides a thread-safe LRU cache implementation with TTL support
type LRUCache struct {
	mu        sync.RWMutex
	capacity  int
	ll        *list.List               // doubly linked list for LRU ordering
	cache     map[string]*list.Element // map for O(1) lookups
	ttl       time.Duration            // optional TTL for cache items
	cacheType string                   // type of cache for metrics (e.g., "cluster", "http_filter")
}

// NewTypedLRUCache creates a new LRU cache with the specified capacity, TTL, and cache type
// The cache type is used for metrics identification
func NewTypedLRUCache(capacity int, ttl time.Duration, cacheType string) *LRUCache {
	return &LRUCache{
		capacity:  capacity,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		ttl:       ttl,
		cacheType: cacheType,
	}
}

// Get retrieves a value from the cache by key
// Returns the value and a boolean indicating if the key was found
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	_, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		// Record cache miss
		RecordCacheMiss(c.cacheType)
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	element, exists := c.cache[key]
	if !exists {
		// Record cache miss
		RecordCacheMiss(c.cacheType)
		return nil, false
	}

	item := element.Value.(*LRUCacheItem)

	// Check TTL if enabled
	if c.ttl > 0 && time.Since(item.Timestamp) > c.ttl {
		c.ll.Remove(element)
		delete(c.cache, key)
		// Record TTL eviction
		RecordCacheEviction(c.cacheType, "ttl")
		RecordCacheMiss(c.cacheType)
		// Update cache size metric
		UpdateCacheSize(c.cacheType, c.ll.Len())
		return nil, false
	}

	// Move to front (most recently used)
	c.ll.MoveToFront(element)
	item.Timestamp = time.Now() // Update timestamp on access

	// Record cache hit
	RecordCacheHit(c.cacheType)

	return item.Value, true
}

// Set adds or updates a value in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, update its value and move to front
	if element, exists := c.cache[key]; exists {
		c.ll.MoveToFront(element)
		item := element.Value.(*LRUCacheItem)
		item.Value = value
		item.Timestamp = time.Now()
		return
	}

	// Evict if at capacity
	if c.ll.Len() >= c.capacity {
		c.evictOldest()
		// Record LRU eviction
		RecordCacheEviction(c.cacheType, "lru")
	}

	// Add new item
	item := &LRUCacheItem{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}
	element := c.ll.PushFront(item)
	c.cache[key] = element

	// Update cache size metric
	UpdateCacheSize(c.cacheType, c.ll.Len())
}

// Remove explicitly removes a key from the cache
func (c *LRUCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.cache[key]; exists {
		c.ll.Remove(element)
		delete(c.cache, key)

		// Record manual eviction
		RecordCacheEviction(c.cacheType, "manual")

		// Update cache size metric
		UpdateCacheSize(c.cacheType, c.ll.Len())
	}
}

// Clear empties the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldSize := c.ll.Len()

	// Record bulk eviction if there were items
	if oldSize > 0 {
		RecordCacheEviction(c.cacheType, "bulk_clear")
	}

	c.ll = list.New()
	c.cache = make(map[string]*list.Element)

	// Update cache size metric (should be zero)
	UpdateCacheSize(c.cacheType, 0)
}

// Len returns the current number of items in the cache
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ll.Len()
}

// evictOldest removes the least recently used item from the cache
// Note: this method is not thread-safe and should be called with lock held
func (c *LRUCache) evictOldest() {
	if c.ll.Len() == 0 {
		return
	}

	oldest := c.ll.Back()
	if oldest != nil {
		item := oldest.Value.(*LRUCacheItem)
		delete(c.cache, item.Key)
		c.ll.Remove(oldest)
	}
}

// RemoveExpired removes all expired items from the cache
// Returns the number of items removed
func (c *LRUCache) RemoveExpired() int {
	if c.ttl <= 0 {
		return 0 // TTL not enabled, nothing to do
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	now := time.Now()

	for e := c.ll.Back(); e != nil; {
		item := e.Value.(*LRUCacheItem)
		if now.Sub(item.Timestamp) > c.ttl {
			next := e.Prev() // Get next before removing
			c.ll.Remove(e)
			delete(c.cache, item.Key)
			removed++

			// Record TTL-based eviction
			RecordCacheEviction(c.cacheType, "ttl")

			e = next
		} else {
			e = e.Prev()
		}
	}

	// Update cache size metric if items were removed
	if removed > 0 {
		UpdateCacheSize(c.cacheType, c.ll.Len())
	}

	return removed
}
