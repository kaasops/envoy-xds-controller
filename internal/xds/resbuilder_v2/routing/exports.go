package routing

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

// NewRoutingBuilder creates a new instance of the RoutingBuilder interface
func NewRoutingBuilder(store *store.LegacyStore) *Builder {
	return NewBuilder(store)
}
