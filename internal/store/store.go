package store

// New creates a new store instance
func New() Store {
	return NewOptimizedStore()
}
