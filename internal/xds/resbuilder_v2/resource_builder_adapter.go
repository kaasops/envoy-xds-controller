package resbuilder_v2

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/adapters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/clusters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filter_chains"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/main_builder"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/routes"
)

// UpdateResourceBuilder updates the ResourceBuilder to use the MainBuilder interface
// This can be called after creating a ResourceBuilder to add the MainBuilder functionality
func UpdateResourceBuilder(rb *ResourceBuilder) {
	// Create adapter components
	httpFilterAdapter := adapters.NewHTTPFilterAdapter(rb.filtersBuilder, rb.store)
	filterChainAdapter := adapters.NewFilterChainAdapter(filter_chains.NewFilterChainBuilder(rb.store), rb.store)
	routingAdapter := adapters.NewRoutingAdapter(rb.routesBuilder)
	accessLogAdapter := rb.filtersBuilder // filtersBuilder already implements AccessLogBuilder
	tlsAdapter := adapters.NewTLSAdapter(rb.store)
	clusterExtractorAdapter := adapters.NewClusterExtractorAdapter(rb.clustersBuilder, rb.store)

	// Create MainBuilder
	mainBuilder := main_builder.NewMainBuilder(rb.store)

	// Set components on MainBuilder
	mainBuilder.SetComponents(
		httpFilterAdapter,
		filterChainAdapter,
		routingAdapter,
		accessLogAdapter,
		tlsAdapter,
		clusterExtractorAdapter,
	)

	// Set MainBuilder on ResourceBuilder
	rb.mainBuilder = mainBuilder
}

// NewModernResourceBuilder creates a new ResourceBuilder with the MainBuilder already configured
func NewModernResourceBuilder(store *store.Store) *ResourceBuilder {
	// Create standard ResourceBuilder
	rb := &ResourceBuilder{
		store:           store,
		clustersBuilder: clusters.NewBuilder(store),
		filtersBuilder:  filters.NewBuilder(store),
		routesBuilder:   routes.NewBuilder(store),
		secretsBuilder:  nil, // Not used with MainBuilder
	}

	// Update with MainBuilder
	UpdateResourceBuilder(rb)

	return rb
}

// BuildResourcesWithMainBuilder builds resources using the MainBuilder
// This can be used as a drop-in replacement for BuildResources
func (rb *ResourceBuilder) BuildResourcesWithMainBuilder(vs *v1alpha1.VirtualService) (*Resources, error) {
	// Make sure MainBuilder is initialized
	if rb.mainBuilder == nil {
		UpdateResourceBuilder(rb)
	}

	// Call MainBuilder.BuildResources
	result, err := rb.mainBuilder.BuildResources(vs)
	if err != nil {
		return nil, err
	}

	// Convert result from interface{} to *Resources
	// In a real implementation, this would require a proper type assertion
	// and possibly a conversion between different Resources structs
	mainResources, ok := result.(*main_builder.Resources)
	if !ok {
		// This shouldn't happen if the MainBuilder is implemented correctly
		return nil, err
	}

	// Convert from main_builder.Resources to resbuilder_v2.Resources
	resources := &Resources{
		Listener:    mainResources.Listener,
		FilterChain: mainResources.FilterChain,
		RouteConfig: mainResources.RouteConfig,
		Clusters:    mainResources.Clusters,
		Secrets:     mainResources.Secrets,
		UsedSecrets: mainResources.UsedSecrets,
		Domains:     mainResources.Domains,
	}

	return resources, nil
}