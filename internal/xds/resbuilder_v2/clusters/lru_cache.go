package clusters

import (
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultCacheCapacity is the default maximum number of cached cluster configurations
	DefaultCacheCapacity = 500

	// DefaultCacheTTL is the default time-to-live for cached cluster configurations
	// 5 minutes is a reasonable default that balances freshness with performance
	DefaultCacheTTL = 5 * time.Minute
)

// ClusterLRUCache provides a thread-safe LRU cache for cluster configurations
type ClusterLRUCache struct {
	lru *utils.LRUCache
}

// NewClusterLRUCache creates a new LRU cache for cluster configurations
// with optional custom capacity and TTL
func NewClusterLRUCache(capacity int, ttl time.Duration) *ClusterLRUCache {
	if capacity <= 0 {
		capacity = DefaultCacheCapacity
	}

	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	return &ClusterLRUCache{
		lru: utils.NewTypedLRUCache(capacity, ttl, "cluster_lru"),
	}
}

// Get retrieves cached clusters for the given key
func (c *ClusterLRUCache) Get(key string) ([]*cluster.Cluster, bool) {
	value, exists := c.lru.Get(key)
	if !exists {
		return nil, false
	}

	// Type assertion
	clusters, ok := value.([]*cluster.Cluster)
	if !ok {
		return nil, false
	}

	// Return deep copies to avoid mutation issues
	result := make([]*cluster.Cluster, len(clusters))
	for i, cl := range clusters {
		result[i] = proto.Clone(cl).(*cluster.Cluster)
	}

	return result, true
}

// Set stores clusters in the cache for the given key
func (c *ClusterLRUCache) Set(key string, clusters []*cluster.Cluster) {
	// Store deep copies to avoid mutation issues
	cached := make([]*cluster.Cluster, len(clusters))
	for i, cl := range clusters {
		cached[i] = proto.Clone(cl).(*cluster.Cluster)
	}

	c.lru.Set(key, cached)
}

// Remove explicitly removes a key from the cache
func (c *ClusterLRUCache) Remove(key string) {
	c.lru.Remove(key)
}

// Clear empties the cache
func (c *ClusterLRUCache) Clear() {
	c.lru.Clear()
}

// Len returns the current number of items in the cache
func (c *ClusterLRUCache) Len() int {
	return c.lru.Len()
}

// RemoveExpired removes all expired items from the cache
// Returns the number of items removed
func (c *ClusterLRUCache) RemoveExpired() int {
	return c.lru.RemoveExpired()
}

// globalClusterCache is the singleton instance of the cluster cache
var globalClusterCache = NewClusterLRUCache(DefaultCacheCapacity, DefaultCacheTTL)

// GetGlobalClusterCache returns the singleton instance of the cluster cache
func GetGlobalClusterCache() *ClusterLRUCache {
	return globalClusterCache
}
