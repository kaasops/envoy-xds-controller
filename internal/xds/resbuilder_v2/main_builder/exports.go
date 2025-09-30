package main_builder

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
)

// Ensure Builder implements interfaces.MainBuilder
var _ interfaces.MainBuilder = (*Builder)(nil)

// NewMainBuilder creates a new Main Resource Builder that implements the MainBuilder interface
// This is the main entry point for using the Main Resource Building component
func NewMainBuilder(store store.Store) interfaces.MainBuilder {
	return &Builder{
		store: store,
		// These components would normally be injected, but we're creating them here for simplicity
		// In a more advanced implementation, they could be provided via dependency injection
		httpFilterBuilder:  nil,                 // Will be set in the ResourceBuilder
		filterChainBuilder: nil,                 // Will be set in the ResourceBuilder
		routingBuilder:     nil,                 // Will be set in the ResourceBuilder
		accessLogBuilder:   nil,                 // Will be set in the ResourceBuilder
		tlsBuilder:         nil,                 // Will be set in the ResourceBuilder
		clusterExtractor:   nil,                 // Will be set in the ResourceBuilder
		cache:              newResourcesCache(), // Initialize cache
	}
}

// SetComponents sets all the component builders for the Main Builder
// This allows the ResourceBuilder to inject its components into the Main Builder
func (b *Builder) SetComponents(
	httpFilterBuilder interfaces.HTTPFilterBuilder,
	filterChainBuilder interfaces.FilterChainBuilder,
	routingBuilder interfaces.RoutingBuilder,
	accessLogBuilder interfaces.AccessLogBuilder,
	tlsBuilder interfaces.TLSBuilder,
	clusterExtractor interfaces.ClusterExtractor,
) {
	b.httpFilterBuilder = httpFilterBuilder
	b.filterChainBuilder = filterChainBuilder
	b.routingBuilder = routingBuilder
	b.accessLogBuilder = accessLogBuilder
	b.tlsBuilder = tlsBuilder
	b.clusterExtractor = clusterExtractor
}
