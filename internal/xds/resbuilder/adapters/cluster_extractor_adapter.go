package adapters

import (
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/clusters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/interfaces"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterExtractorAdapter adapts the clusters.Builder to implement the ClusterExtractor interface
type ClusterExtractorAdapter struct {
	builder *clusters.Builder
	store   store.Store
}

// NewClusterExtractorAdapter creates a new adapter for the clusters.Builder
func NewClusterExtractorAdapter(builder *clusters.Builder, store store.Store) interfaces.ClusterExtractor {
	return &ClusterExtractorAdapter{
		builder: builder,
		store:   store,
	}
}

// ExtractClustersFromFilterChains extracts clusters from filter chains.
// Delegates to clusters.ExtractClustersFromFilterChains which supports
// multiple filter types including TCP Proxy, HTTP Connection Manager,
// and generic filters (like Kafka Broker) via JSON-based extraction.
func (a *ClusterExtractorAdapter) ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain) ([]*cluster.Cluster, error) {
	return clusters.ExtractClustersFromFilterChains(filterChains, a.store)
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

// ExtractClustersFromTracingRaw extracts clusters from inline tracing configuration
func (a *ClusterExtractorAdapter) ExtractClustersFromTracingRaw(tr *runtime.RawExtension) ([]*cluster.Cluster, error) {
	// Delegate to the wrapped builder's FromTracingRaw method
	return a.builder.FromTracingRaw(tr)
}

// ExtractClustersFromTracingRef extracts clusters from tracing reference
func (a *ClusterExtractorAdapter) ExtractClustersFromTracingRef(vs *v1alpha1.VirtualService) ([]*cluster.Cluster, error) {
	// Delegate to the wrapped builder's FromTracingRef method
	return a.builder.FromTracingRef(vs)
}
