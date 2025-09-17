package adapters

import (
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/clusters"
)

// ClusterExtractorAdapter adapts the clusters.Builder to implement the ClusterExtractor interface
type ClusterExtractorAdapter struct {
	builder *clusters.Builder
	store   *store.Store
}

// NewClusterExtractorAdapter creates a new adapter for the clusters.Builder
func NewClusterExtractorAdapter(builder *clusters.Builder, store *store.Store) resbuilder_v2.ClusterExtractor {
	return &ClusterExtractorAdapter{
		builder: builder,
		store:   store,
	}
}

// ExtractClustersFromFilterChains extracts clusters from filter chains
func (a *ClusterExtractorAdapter) ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain) ([]*cluster.Cluster, error) {
	// This method isn't directly available in clusters.Builder
	// Implementing based on extractClustersFromFilterChains in builder.go
	
	var clusters []*cluster.Cluster
	
	// For each filter chain
	for _, fc := range filterChains {
		for _, filter := range fc.Filters {
			// Look for TCP proxy filters with cluster references
			if tc := filter.GetTypedConfig(); tc != nil {
				// In a real implementation, you would extract clusters from TCP proxy config
				// This would involve:
				// 1. Checking if the filter is a TCP proxy
				// 2. Extracting the cluster name
				// 3. Getting the cluster from the store
				// 4. Adding it to the results
			}
		}
	}
	
	// In a real implementation, this would need to be fully implemented
	// For now, returning an empty slice to demonstrate the structure
	return clusters, nil
}

// ExtractClustersFromVirtualHost extracts clusters from a virtual host
func (a *ClusterExtractorAdapter) ExtractClustersFromVirtualHost(virtualHost *routev3.VirtualHost) ([]*cluster.Cluster, error) {
	// Delegate to the wrapped builder's FromVirtualHostRoutes method
	return a.builder.FromVirtualHostRoutes(virtualHost)
}

// ExtractClustersFromHTTPFilters extracts clusters from HTTP filters
func (a *ClusterExtractorAdapter) ExtractClustersFromHTTPFilters(httpFilters []*hcmv3.HttpFilter) ([]*cluster.Cluster, error) {
	// Delegate to the wrapped builder's FromOAuth2HTTPFilters method
	// This method specifically handles OAuth2 filters that may reference clusters
	return a.builder.FromOAuth2HTTPFilters(httpFilters)
}