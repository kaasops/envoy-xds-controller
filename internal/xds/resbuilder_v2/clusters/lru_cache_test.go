package clusters

import (
	"testing"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

// createTestCluster creates a simple test cluster with the given name
func createTestCluster(name string) *cluster.Cluster {
	return &cluster.Cluster{
		Name: name,
	}
}

func TestClusterLRUCache_BasicOperations(t *testing.T) {
	cache := NewClusterLRUCache(5, 0) // No TTL

	// Create test clusters
	cluster1 := createTestCluster("test-cluster-1")
	cluster2 := createTestCluster("test-cluster-2")

	// Test Set and Get
	cache.Set("key1", []*cluster.Cluster{cluster1, cluster2})
	clusters, found := cache.Get("key1")
	if !found {
		t.Errorf("Failed to get clusters for key1")
	}
	if len(clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(clusters))
	}
	if clusters[0].Name != "test-cluster-1" || clusters[1].Name != "test-cluster-2" {
		t.Errorf("Cluster names don't match expected values")
	}

	// Test getting non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Errorf("Unexpectedly found clusters for nonexistent key")
	}

	// Test Remove
	cache.Remove("key1")
	_, found = cache.Get("key1")
	if found {
		t.Errorf("Found clusters for removed key")
	}

	// Test Clear
	cache.Set("key1", []*cluster.Cluster{cluster1})
	cache.Set("key2", []*cluster.Cluster{cluster2})
	cache.Clear()
	_, found = cache.Get("key1")
	if found {
		t.Errorf("Found clusters for key1 after clear")
	}
	_, found = cache.Get("key2")
	if found {
		t.Errorf("Found clusters for key2 after clear")
	}
}

func TestClusterLRUCache_DeepCopy(t *testing.T) {
	cache := NewClusterLRUCache(5, 0) // No TTL

	// Create a test cluster
	origCluster := createTestCluster("test-cluster")
	origClusters := []*cluster.Cluster{origCluster}

	// Store in cache
	cache.Set("key1", origClusters)

	// Get from cache
	cachedClusters, found := cache.Get("key1")
	if !found {
		t.Fatalf("Failed to get clusters for key1")
	}

	// Modify the original cluster
	origCluster.Name = "modified-name"

	// Verify the cached version wasn't modified (deep copy)
	if cachedClusters[0].Name != "test-cluster" {
		t.Errorf("Expected cached cluster name to remain 'test-cluster', got '%s'", cachedClusters[0].Name)
	}

	// Modify the cached cluster
	cachedClusters[0].Name = "another-name"

	// Get from cache again
	cachedClusters2, _ := cache.Get("key1")

	// Verify this new copy wasn't affected by modifying the first copy
	if cachedClusters2[0].Name != "test-cluster" {
		t.Errorf("Expected second cached copy to have name 'test-cluster', got '%s'", cachedClusters2[0].Name)
	}
}

func TestClusterLRUCache_TTL(t *testing.T) {
	cache := NewClusterLRUCache(5, 100*time.Millisecond) // Short TTL for testing

	// Create a test cluster
	testCluster := createTestCluster("test-cluster")
	clusters := []*cluster.Cluster{testCluster}

	// Add to cache
	cache.Set("key1", clusters)

	// Immediately get the key - should exist
	_, found := cache.Get("key1")
	if !found {
		t.Errorf("key1 should exist immediately after adding")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Try to get the key again - should not exist
	_, found = cache.Get("key1")
	if found {
		t.Errorf("key1 should have expired")
	}
}

func TestClusterLRUCache_RemoveExpired(t *testing.T) {
	cache := NewClusterLRUCache(5, 100*time.Millisecond) // Short TTL for testing

	// Create test clusters
	cluster1 := createTestCluster("test-cluster-1")
	cluster2 := createTestCluster("test-cluster-2")
	cluster3 := createTestCluster("test-cluster-3")

	// Add keys with clusters
	cache.Set("key1", []*cluster.Cluster{cluster1})
	cache.Set("key2", []*cluster.Cluster{cluster2})
	cache.Set("key3", []*cluster.Cluster{cluster3})

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Add more keys
	cluster4 := createTestCluster("test-cluster-4")
	cluster5 := createTestCluster("test-cluster-5")
	cache.Set("key4", []*cluster.Cluster{cluster4})
	cache.Set("key5", []*cluster.Cluster{cluster5})

	// Remove expired keys
	removed := cache.RemoveExpired()
	if removed != 3 {
		t.Errorf("Expected 3 expired keys removed, got %d", removed)
	}

	// key1, key2, key3 should be gone
	_, found := cache.Get("key1")
	if found {
		t.Errorf("key1 should have been removed")
	}
	_, found = cache.Get("key2")
	if found {
		t.Errorf("key2 should have been removed")
	}
	_, found = cache.Get("key3")
	if found {
		t.Errorf("key3 should have been removed")
	}

	// key4, key5 should still be there
	_, found = cache.Get("key4")
	if !found {
		t.Errorf("key4 should still be in the cache")
	}
	_, found = cache.Get("key5")
	if !found {
		t.Errorf("key5 should still be in the cache")
	}
}

func TestGetGlobalClusterCache(t *testing.T) {
	// Get the global cache instance
	cache1 := GetGlobalClusterCache()
	if cache1 == nil {
		t.Errorf("GetGlobalClusterCache() returned nil")
	}

	// Get it again and verify it's the same instance
	cache2 := GetGlobalClusterCache()
	if cache2 == nil {
		t.Errorf("Second call to GetGlobalClusterCache() returned nil")
	}

	// Verify it's the same instance (singleton pattern)
	if cache1 != cache2 {
		t.Errorf("GetGlobalClusterCache() returned different instances")
	}

	// Simple operation to verify it works
	testCluster := createTestCluster("test-global-cluster")
	cache1.Set("global-test-key", []*cluster.Cluster{testCluster})
	
	clusters, found := cache2.Get("global-test-key")
	if !found {
		t.Errorf("Failed to get clusters from global cache")
	}
	if len(clusters) != 1 || clusters[0].Name != "test-global-cluster" {
		t.Errorf("Retrieved incorrect cluster data from global cache")
	}
}