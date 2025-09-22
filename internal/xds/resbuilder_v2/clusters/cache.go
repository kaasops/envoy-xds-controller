package clusters

import (
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

// cache provides thread-safe caching of cluster extraction results
// This is a backward-compatible wrapper around ClusterLRUCache
type cache struct {
	lruCache *ClusterLRUCache
}

// newCache creates a new cluster cache with default settings
func newCache() *cache {
	return &cache{
		lruCache: GetGlobalClusterCache(),
	}
}

// get retrieves cached clusters for the given key
func (c *cache) get(key string) ([]*cluster.Cluster, bool) {
	return c.lruCache.Get(key)
}

// set stores clusters in the cache for the given key
func (c *cache) set(key string, clusters []*cluster.Cluster) {
	c.lruCache.Set(key, clusters)
}
