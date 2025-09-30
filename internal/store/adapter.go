package store

import "os"

// StoreAdapter wraps LegacyStore to implement the Store interface
type StoreAdapter struct {
	*LegacyStore
}

// NewStoreAdapter creates a new Store instance based on the STORE_USE_OPTIMIZED environment variable
func NewStoreAdapter() Store {
	if os.Getenv("STORE_USE_OPTIMIZED") == "true" {
		return NewOptimizedStore()
	}
	return &StoreAdapter{
		LegacyStore: New(),
	}
}

// NewStoreAdapterFromLegacy wraps an existing LegacyStore
func NewStoreAdapterFromLegacy(legacy *LegacyStore) Store {
	return &StoreAdapter{
		LegacyStore: legacy,
	}
}

// Copy returns a new Store interface containing a copy of the data
func (s *StoreAdapter) Copy() Store {
	return &StoreAdapter{
		LegacyStore: s.LegacyStore.Copy(),
	}
}

// Ensure StoreAdapter implements Store interface
var _ Store = (*StoreAdapter)(nil)
