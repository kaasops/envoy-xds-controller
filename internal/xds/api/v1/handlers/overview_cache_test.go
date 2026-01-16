package handlers

import (
	"sync"
	"testing"
	"time"
)

func TestOverviewCache_BasicOperations(t *testing.T) {
	cache := NewOverviewCache(100 * time.Millisecond)
	defer cache.Stop()

	// Test Set and Get
	response := &NodeOverviewResponse{
		NodeID: "test-node",
		Summary: OverviewSummary{
			TotalDomains:   5,
			TotalEndpoints: 10,
		},
	}

	cache.Set("test-node", response)

	got := cache.Get("test-node")
	if got == nil {
		t.Fatal("Expected to get cached response, got nil")
	}
	if got.NodeID != "test-node" {
		t.Errorf("Expected NodeID 'test-node', got '%s'", got.NodeID)
	}
	if got.Summary.TotalDomains != 5 {
		t.Errorf("Expected TotalDomains 5, got %d", got.Summary.TotalDomains)
	}

	// Test getting non-existent key
	got = cache.Get("nonexistent")
	if got != nil {
		t.Errorf("Expected nil for nonexistent key, got %v", got)
	}
}

func TestOverviewCache_Invalidate(t *testing.T) {
	cache := NewOverviewCache(time.Hour) // Long TTL
	defer cache.Stop()

	response := &NodeOverviewResponse{NodeID: "test-node"}
	cache.Set("test-node", response)

	// Verify it's cached
	if cache.Get("test-node") == nil {
		t.Fatal("Response should be cached")
	}

	// Invalidate
	cache.Invalidate("test-node")

	// Verify it's gone
	if cache.Get("test-node") != nil {
		t.Error("Response should be invalidated")
	}
}

func TestOverviewCache_InvalidateAll(t *testing.T) {
	cache := NewOverviewCache(time.Hour) // Long TTL
	defer cache.Stop()

	cache.Set("node1", &NodeOverviewResponse{NodeID: "node1"})
	cache.Set("node2", &NodeOverviewResponse{NodeID: "node2"})
	cache.Set("node3", &NodeOverviewResponse{NodeID: "node3"})

	// Verify all cached
	if cache.Get("node1") == nil || cache.Get("node2") == nil || cache.Get("node3") == nil {
		t.Fatal("All responses should be cached")
	}

	// Invalidate all
	cache.InvalidateAll()

	// Verify all gone
	if cache.Get("node1") != nil || cache.Get("node2") != nil || cache.Get("node3") != nil {
		t.Error("All responses should be invalidated")
	}
}

func TestOverviewCache_TTLExpiration(t *testing.T) {
	cache := NewOverviewCache(50 * time.Millisecond)
	defer cache.Stop()

	response := &NodeOverviewResponse{NodeID: "test-node"}
	cache.Set("test-node", response)

	// Immediately get - should exist
	if cache.Get("test-node") == nil {
		t.Error("Response should exist immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if cache.Get("test-node") != nil {
		t.Error("Response should have expired")
	}
}

func TestOverviewCache_Cleanup(t *testing.T) {
	cache := NewOverviewCache(50 * time.Millisecond)
	defer cache.Stop()

	// Add entries
	cache.Set("node1", &NodeOverviewResponse{NodeID: "node1"})
	cache.Set("node2", &NodeOverviewResponse{NodeID: "node2"})

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Add fresh entry
	cache.Set("node3", &NodeOverviewResponse{NodeID: "node3"})

	// Run cleanup
	cache.Cleanup()

	// node1, node2 should be cleaned up, node3 should remain
	cache.mu.RLock()
	entriesCount := len(cache.entries)
	cache.mu.RUnlock()

	if entriesCount != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", entriesCount)
	}

	if cache.Get("node3") == nil {
		t.Error("Fresh entry should still exist")
	}
}

func TestOverviewCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache := NewOverviewCache(time.Hour)
	defer cache.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := "node"
			for j := 0; j < iterations; j++ {
				cache.Set(nodeID, &NodeOverviewResponse{
					NodeID: nodeID,
					Summary: OverviewSummary{
						TotalDomains: id*iterations + j,
					},
				})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = cache.Get("node")
			}
		}()
	}

	// Concurrent invalidations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations/10; j++ {
				cache.Invalidate("node")
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	// If we reach here without deadlock or panic, the test passes
}

func TestOverviewCache_GetExpiredThenDelete(t *testing.T) {
	cache := NewOverviewCache(10 * time.Millisecond)
	defer cache.Stop()

	cache.Set("test-node", &NodeOverviewResponse{NodeID: "test-node"})

	// Wait for expiration
	time.Sleep(50 * time.Millisecond)

	// Get should return nil and trigger deletion
	result := cache.Get("test-node")
	if result != nil {
		t.Error("Expired entry should return nil")
	}

	// Verify entry is actually deleted
	cache.mu.RLock()
	_, exists := cache.entries["test-node"]
	cache.mu.RUnlock()

	if exists {
		t.Error("Expired entry should be deleted from map")
	}
}
