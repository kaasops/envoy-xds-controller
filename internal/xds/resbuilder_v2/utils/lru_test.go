package utils

import (
	"testing"
	"time"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(5, 0) // No TTL

	// Test Set and Get
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	if !found {
		t.Errorf("Failed to get value for key1")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	// Test overwriting existing key
	cache.Set("key1", "value1_updated")
	value, found = cache.Get("key1")
	if !found {
		t.Errorf("Failed to get updated value for key1")
	}
	if value != "value1_updated" {
		t.Errorf("Expected 'value1_updated', got '%v'", value)
	}

	// Test getting non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Errorf("Unexpectedly found value for nonexistent key")
	}

	// Test Len
	if cache.Len() != 1 {
		t.Errorf("Expected cache length 1, got %d", cache.Len())
	}

	// Test Remove
	cache.Remove("key1")
	_, found = cache.Get("key1")
	if found {
		t.Errorf("Found value for removed key")
	}
	if cache.Len() != 0 {
		t.Errorf("Expected cache length 0 after remove, got %d", cache.Len())
	}

	// Test Clear
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("Expected cache length 0 after clear, got %d", cache.Len())
	}
}

func TestLRUCache_LRUEviction(t *testing.T) {
	cache := NewLRUCache(3, 0) // Capacity of 3, no TTL

	// Fill the cache
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add one more element to trigger eviction
	cache.Set("key4", "value4")

	// key2 should have been evicted (least recently used)
	_, found := cache.Get("key2")
	if found {
		t.Errorf("key2 should have been evicted")
	}

	// key1, key3, and key4 should still be in the cache
	_, found = cache.Get("key1")
	if !found {
		t.Errorf("key1 should still be in the cache")
	}
	_, found = cache.Get("key3")
	if !found {
		t.Errorf("key3 should still be in the cache")
	}
	_, found = cache.Get("key4")
	if !found {
		t.Errorf("key4 should be in the cache")
	}
}

func TestLRUCache_TTL(t *testing.T) {
	cache := NewLRUCache(5, 100*time.Millisecond) // Short TTL for testing

	// Add a key
	cache.Set("key1", "value1")

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

func TestLRUCache_RemoveExpired(t *testing.T) {
	cache := NewLRUCache(5, 100*time.Millisecond) // Short TTL for testing

	// Add keys
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Add more keys
	cache.Set("key4", "value4")
	cache.Set("key5", "value5")

	// Remove expired keys
	removed := cache.RemoveExpired()
	if removed != 3 {
		t.Errorf("Expected 3 expired keys removed, got %d", removed)
	}

	// Check cache size
	if cache.Len() != 2 {
		t.Errorf("Expected 2 items in cache after removing expired, got %d", cache.Len())
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

func TestLRUCache_ThreadSafety(t *testing.T) {
	cache := NewLRUCache(100, 0) // No TTL
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := "key" + string(rune('a'+i%26))
			cache.Set(key, i)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := "key" + string(rune('a'+i%26))
			cache.Get(key)
		}
		done <- true
	}()

	// Remover goroutine
	go func() {
		for i := 0; i < 50; i++ {
			key := "key" + string(rune('a'+i%26))
			cache.Remove(key)
		}
		done <- true
	}()

	// Wait for all goroutines to finish
	<-done
	<-done
	<-done

	// The test passes if there are no race conditions or panics
}