package utils

import (
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

// Object pools for frequently allocated slices to reduce GC pressure
var (
	StringSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 8) // Initial capacity of 8
		},
	}
	
	ClusterSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]*cluster.Cluster, 0, 4) // Initial capacity of 4
		},
	}
)

// GetStringSlice returns a string slice from the pool
func GetStringSlice() []string {
	return StringSlicePool.Get().([]string)
}

// PutStringSlice returns a string slice to the pool after clearing it
func PutStringSlice(s []string) {
	s = s[:0] // Clear the slice but keep capacity
	StringSlicePool.Put(s)
}

// GetClusterSlice returns a cluster slice from the pool
func GetClusterSlice() []*cluster.Cluster {
	return ClusterSlicePool.Get().([]*cluster.Cluster)
}

// PutClusterSlice returns a cluster slice to the pool after clearing it
func PutClusterSlice(s []*cluster.Cluster) {
	s = s[:0] // Clear the slice but keep capacity
	ClusterSlicePool.Put(s)
}