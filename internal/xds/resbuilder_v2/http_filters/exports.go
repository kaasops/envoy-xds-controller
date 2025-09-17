package http_filters

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
)

// NewHTTPFilterBuilder creates a new HTTP filter builder
// that implements the resbuilder_v2.HTTPFilterBuilder interface
func NewHTTPFilterBuilder(store *store.Store) resbuilder_v2.HTTPFilterBuilder {
	return NewBuilder(store)
}