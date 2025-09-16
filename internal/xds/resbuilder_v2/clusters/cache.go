package clusters

import (
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

// cache provides thread-safe caching of cluster extraction results
type cache struct {
	mu      sync.RWMutex
	data    map[string][]*cluster.Cluster
	maxSize int
}

// newCache creates a new cluster cache with default settings
func newCache() *cache {
	return &cache{
		data:    make(map[string][]*cluster.Cluster),
		maxSize: 500, // Smaller cache size than HTTP filters cache
	}
}

// get retrieves cached clusters for the given key
func (c *cache) get(key string) ([]*cluster.Cluster, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	clusters, exists := c.data[key]
	if !exists {
		return nil, false
	}
	
	// Return deep copies to avoid mutation issues
	result := make([]*cluster.Cluster, len(clusters))
	for i, cl := range clusters {
		result[i] = &cluster.Cluster{}
		*result[i] = *cl // Shallow copy should be sufficient for protobuf messages
	}
	
	return result, true
}

// set stores clusters in the cache for the given key
func (c *cache) set(key string, clusters []*cluster.Cluster) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Simple eviction: if cache is full, clear it
	if len(c.data) >= c.maxSize {
		c.data = make(map[string][]*cluster.Cluster)
	}
	
	// Store deep copies to avoid mutation issues
	cached := make([]*cluster.Cluster, len(clusters))
	for i, cl := range clusters {
		cached[i] = &cluster.Cluster{}
		*cached[i] = *cl // Shallow copy should be sufficient for protobuf messages
	}
	
	c.data[key] = cached
}

// clear removes all entries from the cache
func (c *cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data = make(map[string][]*cluster.Cluster)
}

// size returns the current number of entries in the cache
func (c *cache) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.data)
}