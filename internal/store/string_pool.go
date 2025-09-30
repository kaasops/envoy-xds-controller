package store

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	defaultNamespace = "default"
)

// StringPool implements string interning to reduce memory allocations
// It maintains a pool of unique strings and returns references to them
type StringPool struct {
	mu   sync.RWMutex
	pool map[string]string

	// Statistics (atomic for lock-free access)
	hits   uint64
	misses uint64
}

// NewStringPool creates a new string interning pool with optimized initial capacity
func NewStringPool() *StringPool {
	return &StringPool{
		pool: make(map[string]string, 1000), // Pre-allocate for common strings
	}
}

// Intern returns an interned version of the string
// If the string already exists in the pool, returns the existing reference
// Otherwise, adds it to the pool and returns it
func (sp *StringPool) Intern(s string) string {
	if s == "" {
		return ""
	}

	// Fast path: check if string already exists
	sp.mu.RLock()
	if interned, ok := sp.pool[s]; ok {
		sp.mu.RUnlock()
		atomic.AddUint64(&sp.hits, 1)
		return interned
	}
	sp.mu.RUnlock()

	// Slow path: add string to pool
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Double-check in case another goroutine added it
	if interned, ok := sp.pool[s]; ok {
		atomic.AddUint64(&sp.hits, 1)
		return interned
	}

	// Add to pool
	sp.pool[s] = s
	atomic.AddUint64(&sp.misses, 1)
	return s
}

// InternUID optimized for UID strings (most common case)
func (sp *StringPool) InternUID(uid string) string {
	if uid == "" {
		return uid
	}
	return sp.Intern(uid)
}

// InternNamespace optimized for namespace strings with common case optimization
func (sp *StringPool) InternNamespace(ns string) string {
	// Fast path for most common namespace
	if ns == defaultNamespace || ns == "" {
		return ns
	}
	return sp.Intern(ns)
}

// Size returns the number of unique strings in the pool
func (sp *StringPool) Size() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return len(sp.pool)
}

// Clear removes all strings from the pool
func (sp *StringPool) Clear() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.pool = make(map[string]string)
}

// StringPoolStats returns statistics about the string pool
type StringPoolStats struct {
	UniqueStrings int
	MemoryBytes   int64
	Hits          uint64
	Misses        uint64
	HitRatio      float64
}

// GetStats returns statistics about the string pool
func (sp *StringPool) GetStats() StringPoolStats {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var memoryBytes int64
	for s := range sp.pool {
		// Each string uses len(s) bytes plus overhead
		// String header is 16 bytes on 64-bit systems
		memoryBytes += int64(len(s)) + int64(unsafe.Sizeof(s))
	}

	hits := atomic.LoadUint64(&sp.hits)
	misses := atomic.LoadUint64(&sp.misses)
	total := hits + misses
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	return StringPoolStats{
		UniqueStrings: len(sp.pool),
		MemoryBytes:   memoryBytes,
		Hits:          hits,
		Misses:        misses,
		HitRatio:      hitRatio,
	}
}

// Cleanup removes strings when pool becomes too large
func (sp *StringPool) Cleanup(maxSize int) int {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if len(sp.pool) <= maxSize {
		return 0
	}

	// Simple cleanup: clear half when too large
	newPool := make(map[string]string, maxSize)
	count := 0
	for k, v := range sp.pool {
		if count >= maxSize/2 {
			break
		}
		newPool[k] = v
		count++
	}

	removed := len(sp.pool) - len(newPool)
	sp.pool = newPool
	return removed
}
