package resbuilder_v2

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

// HTTPFilterBuilder is responsible for building HTTP filters
type HTTPFilterBuilder interface {
	BuildHTTPFilters(vs *v1alpha1.VirtualService) ([]*hcmv3.HttpFilter, error)
	BuildRBACFilter(vs *v1alpha1.VirtualService) (*rbacFilter.RBAC, error)
}

// FilterChainBuilder is responsible for building filter chains
type FilterChainBuilder interface {
	BuildFilterChains(params *FilterChainsParams) ([]*listenerv3.FilterChain, error)
	BuildFilterChainParams(vs *v1alpha1.VirtualService, nn helpers.NamespacedName, 
						httpFilters []*hcmv3.HttpFilter, listenerIsTLS bool, 
						virtualHost *routev3.VirtualHost) (*FilterChainsParams, error)
	CheckFilterChainsConflicts(vs *v1alpha1.VirtualService) error
}

// RoutingBuilder is responsible for building routing configuration
type RoutingBuilder interface {
	BuildRouteConfiguration(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, 
						  nn helpers.NamespacedName) (*routev3.VirtualHost, *routev3.RouteConfiguration, error)
	BuildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName) (*routev3.VirtualHost, error)
}

// AccessLogBuilder is responsible for building access log configuration
type AccessLogBuilder interface {
	BuildAccessLogConfigs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error)
}

// TLSBuilder is responsible for building TLS configuration
type TLSBuilder interface {
	GetTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error)
	GetSecretNameToDomains(vs *v1alpha1.VirtualService, domains []string) (map[helpers.NamespacedName][]string, error)
}

// ClusterExtractor is responsible for extracting clusters
type ClusterExtractor interface {
	ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain) ([]*cluster.Cluster, error)
	ExtractClustersFromVirtualHost(virtualHost *routev3.VirtualHost) ([]*cluster.Cluster, error)
	ExtractClustersFromHTTPFilters(httpFilters []*hcmv3.HttpFilter) ([]*cluster.Cluster, error)
}

// MainBuilder is the interface for the main resource building component
type MainBuilder interface {
	// BuildResources builds all resources for a VirtualService
	BuildResources(vs *v1alpha1.VirtualService) (interface{}, error)
	
	// SetComponents sets all the component builders for the Main Builder
	SetComponents(
		httpFilterBuilder HTTPFilterBuilder,
		filterChainBuilder FilterChainBuilder,
		routingBuilder RoutingBuilder,
		accessLogBuilder AccessLogBuilder,
		tlsBuilder TLSBuilder,
		clusterExtractor ClusterExtractor,
	)
}