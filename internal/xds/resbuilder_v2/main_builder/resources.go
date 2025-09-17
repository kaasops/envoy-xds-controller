package main_builder

import (
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

// Resources represents all the Envoy resources built for a VirtualService
type Resources struct {
	// Listener is the namespaced name of the listener resource
	Listener helpers.NamespacedName

	// FilterChain is a slice of filter chains for the listener
	FilterChain []*listenerv3.FilterChain

	// RouteConfig is the route configuration for the HTTP connection manager
	RouteConfig *routev3.RouteConfiguration

	// Clusters is a slice of clusters used by the virtual service
	Clusters []*clusterv3.Cluster

	// Secrets is a slice of TLS secrets used by the virtual service
	Secrets []*tlsv3.Secret

	// UsedSecrets is a slice of namespaced names of the secrets used
	UsedSecrets []helpers.NamespacedName

	// Domains is a slice of domain names for the virtual service
	Domains []string
}

// HasTLSConfig returns true if the resources include TLS configuration
func (r *Resources) HasTLSConfig() bool {
	return len(r.Secrets) > 0
}

// HasRouteConfig returns true if the resources include route configuration
func (r *Resources) HasRouteConfig() bool {
	return r.RouteConfig != nil
}

// HasFilterChain returns true if the resources include filter chains
func (r *Resources) HasFilterChain() bool {
	return len(r.FilterChain) > 0
}

// HasClusters returns true if the resources include clusters
func (r *Resources) HasClusters() bool {
	return len(r.Clusters) > 0
}
