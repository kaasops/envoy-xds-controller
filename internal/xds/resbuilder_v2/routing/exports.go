package routing

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

// NewRoutingBuilder creates a new instance of the RoutingBuilder interface
func NewRoutingBuilder(store *store.Store) *Builder {
	return NewBuilder(store)
}
