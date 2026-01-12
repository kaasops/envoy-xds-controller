package resbuilder

import (
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/filter_chains"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/main_builder"
)

// UpdateResourceBuilder initializes the internal builder with all required components
// All builders now directly implement their respective interfaces:
// - filters.Builder implements HTTPFilterBuilder
// - filter_chains.Builder implements FilterChainBuilder
// - routes.Builder implements RoutingBuilder
// - secrets.Builder implements TLSBuilder
// - clusters.Builder implements ClusterExtractor
func UpdateResourceBuilder(rb *ResourceBuilder) {
	filterChainsBuilder := filter_chains.NewBuilder(rb.store)

	builder := main_builder.NewMainBuilder(rb.store)
	builder.SetComponents(
		rb.filtersBuilder,   // implements HTTPFilterBuilder
		filterChainsBuilder, // implements FilterChainBuilder
		rb.routesBuilder,    // implements RoutingBuilder
		rb.secretsBuilder,   // implements TLSBuilder
		rb.clustersBuilder,  // implements ClusterExtractor
	)

	rb.mainBuilder = builder
}
