package adapters

import (
	"fmt"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/clusters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterExtractorAdapter adapts the clusters.Builder to implement the ClusterExtractor interface
type ClusterExtractorAdapter struct {
	builder *clusters.Builder
	store   *store.Store
}

// NewClusterExtractorAdapter creates a new adapter for the clusters.Builder
func NewClusterExtractorAdapter(builder *clusters.Builder, store *store.Store) interfaces.ClusterExtractor {
	return &ClusterExtractorAdapter{
		builder: builder,
		store:   store,
	}
}

// ExtractClustersFromFilterChains extracts clusters from filter chains
func (a *ClusterExtractorAdapter) ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain) ([]*cluster.Cluster, error) {
	// Get a slice from the object pool
	clustersPtr := utils.GetClusterSlice()
	defer utils.PutClusterSlice(clustersPtr)

	// Process each filter chain
	for _, fc := range filterChains {
		for _, filter := range fc.Filters {
			if tc := filter.GetTypedConfig(); tc != nil {
				// Check if this is a TCP proxy filter
				if tc.TypeUrl != "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy" {
					return nil, fmt.Errorf("unexpected filter type: %s", tc.TypeUrl)
				}

				// Unmarshal the TCP proxy configuration
				var tcpProxy tcpProxyv3.TcpProxy
				if err := tc.UnmarshalTo(&tcpProxy); err != nil {
					return nil, fmt.Errorf("failed to unmarshal TCP proxy config: %w", err)
				}

				// Extract the cluster name
				clusterName := tcpProxy.GetCluster()

				// Get the cluster from the store
				cl := a.store.GetSpecCluster(clusterName)
				if cl == nil {
					return nil, fmt.Errorf("cluster %s not found", clusterName)
				}

				// Unmarshal and validate the cluster
				xdsCluster, err := cl.UnmarshalV3AndValidate()
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", clusterName, err)
				}

				// Add the cluster to the result collection
				*clustersPtr = append(*clustersPtr, xdsCluster)
			}
		}
	}

	// Create a new slice to return to the caller - we can't return the pooled slice directly
	result := make([]*cluster.Cluster, len(*clustersPtr))
	copy(result, *clustersPtr)

	return result, nil
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
