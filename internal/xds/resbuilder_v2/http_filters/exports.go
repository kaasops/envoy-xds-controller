package http_filters

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
)

// NewHTTPFilterBuilder creates a new HTTP filter builder
// that implements the interfaces.HTTPFilterBuilder interface
func NewHTTPFilterBuilder(store *store.Store) interfaces.HTTPFilterBuilder {
	return NewBuilder(store)
}
