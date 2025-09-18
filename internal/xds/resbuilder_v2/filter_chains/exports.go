package filter_chains

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

// NewFilterChainBuilder creates a new filter chain builder
func NewFilterChainBuilder(store *store.Store) *Builder {
	return NewBuilder(store)
}
