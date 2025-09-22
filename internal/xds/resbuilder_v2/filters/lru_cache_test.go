package filters

import (
	"testing"
	"time"

	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

// createTestHTTPFilter creates a simple test HTTP filter with the given name
func createTestHTTPFilter(name string) *hcmv3.HttpFilter {
	return &hcmv3.HttpFilter{
		Name: name,
	}
}

func TestHTTPFilterLRUCache_BasicOperations(t *testing.T) {
	cache := NewHTTPFilterLRUCache(5, 0) // No TTL

	// Create test filters
	filter1 := createTestHTTPFilter("test-filter-1")
	filter2 := createTestHTTPFilter("test-filter-2")

	// Test Set and Get
	cache.Set("key1", []*hcmv3.HttpFilter{filter1, filter2})
	filters, found := cache.Get("key1")
	if !found {
		t.Errorf("Failed to get filters for key1")
	}
	if len(filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(filters))
	}
	if filters[0].Name != "test-filter-1" || filters[1].Name != "test-filter-2" {
		t.Errorf("Filter names don't match expected values")
	}

	// Test getting non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Errorf("Unexpectedly found filters for nonexistent key")
	}

	// Test Remove
	cache.Remove("key1")
	_, found = cache.Get("key1")
	if found {
		t.Errorf("Found filters for removed key")
	}

	// Test Clear
	cache.Set("key1", []*hcmv3.HttpFilter{filter1})
	cache.Set("key2", []*hcmv3.HttpFilter{filter2})
	cache.Clear()
	_, found = cache.Get("key1")
	if found {
		t.Errorf("Found filters for key1 after clear")
	}
	_, found = cache.Get("key2")
	if found {
		t.Errorf("Found filters for key2 after clear")
	}
}

func TestHTTPFilterLRUCache_DeepCopy(t *testing.T) {
	cache := NewHTTPFilterLRUCache(5, 0) // No TTL

	// Create a test filter
	origFilter := createTestHTTPFilter("test-filter")
	origFilters := []*hcmv3.HttpFilter{origFilter}

	// Store in cache
	cache.Set("key1", origFilters)

	// Get from cache
	cachedFilters, found := cache.Get("key1")
	if !found {
		t.Fatalf("Failed to get filters for key1")
	}

	// Modify the original filter
	origFilter.Name = "modified-name"

	// Verify the cached version wasn't modified (deep copy)
	if cachedFilters[0].Name != "test-filter" {
		t.Errorf("Expected cached filter name to remain 'test-filter', got '%s'", cachedFilters[0].Name)
	}

	// Modify the cached filter
	cachedFilters[0].Name = "another-name"

	// Get from cache again
	cachedFilters2, _ := cache.Get("key1")

	// Verify this new copy wasn't affected by modifying the first copy
	if cachedFilters2[0].Name != "test-filter" {
		t.Errorf("Expected second cached copy to have name 'test-filter', got '%s'", cachedFilters2[0].Name)
	}
}

func TestHTTPFilterLRUCache_TTL(t *testing.T) {
	cache := NewHTTPFilterLRUCache(5, 100*time.Millisecond) // Short TTL for testing

	// Create a test filter
	testFilter := createTestHTTPFilter("test-filter")
	filters := []*hcmv3.HttpFilter{testFilter}

	// Add to cache
	cache.Set("key1", filters)

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

func TestHTTPFilterLRUCache_RemoveExpired(t *testing.T) {
	cache := NewHTTPFilterLRUCache(5, 100*time.Millisecond) // Short TTL for testing

	// Create test filters
	filter1 := createTestHTTPFilter("test-filter-1")
	filter2 := createTestHTTPFilter("test-filter-2")
	filter3 := createTestHTTPFilter("test-filter-3")

	// Add keys with filters
	cache.Set("key1", []*hcmv3.HttpFilter{filter1})
	cache.Set("key2", []*hcmv3.HttpFilter{filter2})
	cache.Set("key3", []*hcmv3.HttpFilter{filter3})

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Add more keys
	filter4 := createTestHTTPFilter("test-filter-4")
	filter5 := createTestHTTPFilter("test-filter-5")
	cache.Set("key4", []*hcmv3.HttpFilter{filter4})
	cache.Set("key5", []*hcmv3.HttpFilter{filter5})

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

// This test verifies that the object pool integration works without errors
// We can't directly check pool usage stats, but we can verify the operations complete successfully
func TestHTTPFilterLRUCache_ObjectPool(t *testing.T) {
	cache := NewHTTPFilterLRUCache(20, 0) // No TTL, larger capacity to fit all test items

	// Create test filters
	filter1 := createTestHTTPFilter("test-filter-1")
	filter2 := createTestHTTPFilter("test-filter-2")

	// Set and get multiple times to exercise the object pool
	for i := 0; i < 10; i++ {
		key := "key" + string(rune('0'+i))
		cache.Set(key, []*hcmv3.HttpFilter{filter1, filter2})
	}

	// Verify we can retrieve the items
	for i := 0; i < 10; i++ {
		key := "key" + string(rune('0'+i))
		filters, found := cache.Get(key)
		if !found {
			t.Errorf("Failed to get filters for key %s", key)
		}
		if len(filters) != 2 {
			t.Errorf("Expected 2 filters for key %s, got %d", key, len(filters))
		}
	}

	// The test passes if all operations complete without errors
}

func TestGetGlobalHTTPFilterCache(t *testing.T) {
	// Get the global cache instance
	cache1 := GetGlobalHTTPFilterCache()
	if cache1 == nil {
		t.Errorf("GetGlobalHTTPFilterCache() returned nil")
	}

	// Get it again and verify it's the same instance
	cache2 := GetGlobalHTTPFilterCache()
	if cache2 == nil {
		t.Errorf("Second call to GetGlobalHTTPFilterCache() returned nil")
	}

	// Verify it's the same instance (singleton pattern)
	if cache1 != cache2 {
		t.Errorf("GetGlobalHTTPFilterCache() returned different instances")
	}

	// Simple operation to verify it works
	testFilter := createTestHTTPFilter("test-global-filter")
	cache1.Set("global-test-key", []*hcmv3.HttpFilter{testFilter})

	filters, found := cache2.Get("global-test-key")
	if !found {
		t.Errorf("Failed to get filters from global cache")
	}
	if len(filters) != 1 || filters[0].Name != "test-global-filter" {
		t.Errorf("Retrieved incorrect filter data from global cache")
	}
}
