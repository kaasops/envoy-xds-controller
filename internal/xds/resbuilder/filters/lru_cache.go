package filters

import (
	"time"

	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/utils"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultCacheCapacity is the default maximum number of cached HTTP filter configurations
	DefaultCacheCapacity = 1000

	// DefaultCacheTTL is the default time-to-live for cached HTTP filter configurations
	// 10 minutes is a reasonable default as HTTP filter configurations change less frequently
	DefaultCacheTTL = 10 * time.Minute
)

// HTTPFilterLRUCache provides a thread-safe LRU cache for HTTP filter configurations
type HTTPFilterLRUCache struct {
	lru *utils.LRUCache
}

// NewHTTPFilterLRUCache creates a new LRU cache for HTTP filter configurations
// with optional custom capacity and TTL
func NewHTTPFilterLRUCache(capacity int, ttl time.Duration) *HTTPFilterLRUCache {
	if capacity <= 0 {
		capacity = DefaultCacheCapacity
	}

	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	return &HTTPFilterLRUCache{
		lru: utils.NewTypedLRUCache(capacity, ttl, "http_filter_lru"),
	}
}

// Get retrieves cached HTTP filters for the given key
func (c *HTTPFilterLRUCache) Get(key string) ([]*hcmv3.HttpFilter, bool) {
	value, exists := c.lru.Get(key)
	if !exists {
		return nil, false
	}

	// Type assertion
	filters, ok := value.([]*hcmv3.HttpFilter)
	if !ok {
		return nil, false
	}

	// Get a slice from the pool
	resultPtr := utils.GetHTTPFilterSlice()

	// Create deep copies to avoid mutation issues
	for _, filter := range filters {
		*resultPtr = append(*resultPtr, proto.Clone(filter).(*hcmv3.HttpFilter))
	}

	// Create a new slice to return to the caller - we can't return the pooled slice directly
	finalResult := make([]*hcmv3.HttpFilter, len(*resultPtr))
	copy(finalResult, *resultPtr)

	// Return the slice to the pool
	utils.PutHTTPFilterSlice(resultPtr)

	return finalResult, true
}

// Set stores HTTP filters in the cache for the given key
func (c *HTTPFilterLRUCache) Set(key string, filters []*hcmv3.HttpFilter) {
	// Get a slice from the pool
	cachedPtr := utils.GetHTTPFilterSlice()

	// Store deep copies to avoid mutation issues
	for _, filter := range filters {
		*cachedPtr = append(*cachedPtr, proto.Clone(filter).(*hcmv3.HttpFilter))
	}

	// Create a permanent slice for the cache - we can't store the pooled slice
	permanentCached := make([]*hcmv3.HttpFilter, len(*cachedPtr))
	copy(permanentCached, *cachedPtr)

	// Store in the LRU cache
	c.lru.Set(key, permanentCached)

	// Return the slice to the pool
	utils.PutHTTPFilterSlice(cachedPtr)
}

// Remove explicitly removes a key from the cache
func (c *HTTPFilterLRUCache) Remove(key string) {
	c.lru.Remove(key)
}

// Clear empties the cache
func (c *HTTPFilterLRUCache) Clear() {
	c.lru.Clear()
}

// Len returns the current number of items in the cache
func (c *HTTPFilterLRUCache) Len() int {
	return c.lru.Len()
}

// RemoveExpired removes all expired items from the cache
// Returns the number of items removed
func (c *HTTPFilterLRUCache) RemoveExpired() int {
	return c.lru.RemoveExpired()
}

// globalHTTPFilterCache is the singleton instance of the HTTP filter cache
var globalHTTPFilterCache = NewHTTPFilterLRUCache(DefaultCacheCapacity, DefaultCacheTTL)

// GetGlobalHTTPFilterCache returns the singleton instance of the HTTP filter cache
func GetGlobalHTTPFilterCache() *HTTPFilterLRUCache {
	return globalHTTPFilterCache
}
