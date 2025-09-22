package tls

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

// NewTLSBuilder creates a new instance of the TLSBuilder interface
func NewTLSBuilder(store *store.Store) *Builder {
	return NewBuilder(store)
}
