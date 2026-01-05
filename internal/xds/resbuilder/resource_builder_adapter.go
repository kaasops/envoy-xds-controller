package resbuilder

import (
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/adapters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/filter_chains"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/main_builder"
)

// UpdateResourceBuilder initializes the internal builder with all required adapters
func UpdateResourceBuilder(rb *ResourceBuilder) {
	httpFilterAdapter := adapters.NewHTTPFilterAdapter(rb.filtersBuilder, rb.store)
	filterChainAdapter := adapters.NewFilterChainAdapter(filter_chains.NewFilterChainBuilder(rb.store), rb.store)
	routingAdapter := adapters.NewRoutingAdapter(rb.routesBuilder)
	accessLogAdapter := rb.filtersBuilder

	builder := main_builder.NewMainBuilder(rb.store)
	builder.SetComponents(
		httpFilterAdapter,
		filterChainAdapter,
		routingAdapter,
		accessLogAdapter,
		rb.secretsBuilder,  // secrets.Builder now implements TLSBuilder interface
		rb.clustersBuilder, // clusters.Builder now implements ClusterExtractor interface
	)

	rb.mainBuilder = builder
}
