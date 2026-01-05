package resbuilder

import (
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/adapters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/filter_chains"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/main_builder"
)

// UpdateResourceBuilder initializes the internal builder with all required adapters
func UpdateResourceBuilder(rb *ResourceBuilder) {
	filterChainAdapter := adapters.NewFilterChainAdapter(filter_chains.NewFilterChainBuilder(rb.store), rb.store)
	routingAdapter := adapters.NewRoutingAdapter(rb.routesBuilder)

	builder := main_builder.NewMainBuilder(rb.store)
	builder.SetComponents(
		rb.filtersBuilder, // filters.Builder now implements HTTPFilterBuilder interface
		filterChainAdapter,
		routingAdapter,
		rb.filtersBuilder,  // filters.Builder also implements AccessLogBuilder interface
		rb.secretsBuilder,  // secrets.Builder now implements TLSBuilder interface
		rb.clustersBuilder, // clusters.Builder now implements ClusterExtractor interface
	)

	rb.mainBuilder = builder
}
