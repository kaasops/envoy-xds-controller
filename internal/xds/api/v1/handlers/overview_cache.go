package handlers

import (
	"sync"
	"time"
)

// OverviewCache provides TTL-based caching for overview responses.
// This prevents CPU-intensive operations (certificate parsing) on every request.
type OverviewCache struct {
	mu      sync.RWMutex
	entries map[string]*overviewCacheEntry
	ttl     time.Duration
	stopCh  chan struct{}
}

type overviewCacheEntry struct {
	response  *NodeOverviewResponse
	expiresAt time.Time
}

// NewOverviewCache creates a new cache with the specified TTL.
// It starts a background goroutine for periodic cleanup.
func NewOverviewCache(ttl time.Duration) *OverviewCache {
	c := &OverviewCache{
		entries: make(map[string]*overviewCacheEntry),
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// cleanupLoop periodically removes expired entries.
func (c *OverviewCache) cleanupLoop() {
	// Cleanup interval is 2x TTL to balance between memory usage and CPU overhead
	ticker := time.NewTicker(c.ttl * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// Stop stops the background cleanup goroutine.
func (c *OverviewCache) Stop() {
	close(c.stopCh)
}

// Get retrieves a cached overview response for the given nodeID.
// Returns nil if not found or expired.
func (c *OverviewCache) Get(nodeID string) *NodeOverviewResponse {
	c.mu.RLock()
	entry, ok := c.entries[nodeID]
	if !ok {
		c.mu.RUnlock()
		return nil
	}

	// Check expiration while still holding read lock
	if time.Now().After(entry.expiresAt) {
		c.mu.RUnlock()
		// Upgrade to write lock and double-check
		c.mu.Lock()
		defer c.mu.Unlock()
		// Re-check: another goroutine might have updated/deleted the entry
		if entry, ok := c.entries[nodeID]; ok && time.Now().After(entry.expiresAt) {
			delete(c.entries, nodeID)
		}
		return nil
	}

	response := entry.response
	c.mu.RUnlock()
	return response
}

// Set stores an overview response in the cache.
func (c *OverviewCache) Set(nodeID string, response *NodeOverviewResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[nodeID] = &overviewCacheEntry{
		response:  response,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes a specific nodeID from the cache.
func (c *OverviewCache) Invalidate(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, nodeID)
}

// InvalidateAll clears the entire cache.
func (c *OverviewCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*overviewCacheEntry)
}

// Cleanup removes all expired entries from the cache.
// Can be called periodically to prevent memory leaks.
func (c *OverviewCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for nodeID, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, nodeID)
		}
	}
}
