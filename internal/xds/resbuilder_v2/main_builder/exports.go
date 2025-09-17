package main_builder

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
)

// Ensure Builder implements resbuilder_v2.MainBuilder
var _ resbuilder_v2.MainBuilder = (*Builder)(nil)

// NewMainBuilder creates a new Main Resource Builder that implements the MainBuilder interface
// This is the main entry point for using the Main Resource Building component
func NewMainBuilder(store *store.Store) resbuilder_v2.MainBuilder {
	return &Builder{
		store: store,
		// These components would normally be injected, but we're creating them here for simplicity
		// In a more advanced implementation, they could be provided via dependency injection
		httpFilterBuilder: nil, // Will be set in the ResourceBuilder
		filterChainBuilder: nil, // Will be set in the ResourceBuilder
		routingBuilder:   nil, // Will be set in the ResourceBuilder
		accessLogBuilder: nil, // Will be set in the ResourceBuilder
		tlsBuilder:       nil, // Will be set in the ResourceBuilder
		clusterExtractor: nil, // Will be set in the ResourceBuilder
	}
}

// Ensure Builder implements resbuilder_v2.MainBuilder
var _ resbuilder_v2.MainBuilder = (*Builder)(nil)

// SetComponents sets all the component builders for the Main Builder
// This allows the ResourceBuilder to inject its components into the Main Builder
func (b *Builder) SetComponents(
	httpFilterBuilder resbuilder_v2.HTTPFilterBuilder,
	filterChainBuilder resbuilder_v2.FilterChainBuilder,
	routingBuilder resbuilder_v2.RoutingBuilder,
	accessLogBuilder resbuilder_v2.AccessLogBuilder,
	tlsBuilder resbuilder_v2.TLSBuilder,
	clusterExtractor resbuilder_v2.ClusterExtractor,
) {
	b.httpFilterBuilder = httpFilterBuilder
	b.filterChainBuilder = filterChainBuilder
	b.routingBuilder = routingBuilder
	b.accessLogBuilder = accessLogBuilder
	b.tlsBuilder = tlsBuilder
	b.clusterExtractor = clusterExtractor
}