package cache

import (
	"fmt"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"

	"google.golang.org/protobuf/proto"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

func GetResources(resources map[string][]types.Resource) (
	clusters []Cluster,
	endpoints []Endpoint,
	routes []Route,
	listeners []Listener,
	secrets []Secret,
	err error,
) {
	for t, resources := range resources {
		switch t {
		case resourcev3.ClusterType:
			clusters, err = GetClusters(resources)
			if err != nil {
				return
			}
		case resourcev3.EndpointType:
			endpoints, err = GetEndpoints(resources)
			if err != nil {
				return
			}
		case resourcev3.RouteType:
			routes, err = GetRoutes(resources)
			if err != nil {
				return
			}
		case resourcev3.ListenerType:
			listeners, err = GetListeners(resources)
			if err != nil {
				return
			}
		case resourcev3.SecretType:
			secrets, err = getSecrets(resources)
			if err != nil {
				return
			}
		case resourcev3.ScopedRouteType:
			if len(resources) > 0 {
				fmt.Println("TODO: ScopedRouteType")
			}
		case resourcev3.VirtualHostType:
			if len(resources) > 0 {
				fmt.Println("TODO: VirtualHostType")
			}
		case resourcev3.ExtensionConfigType:
			if len(resources) > 0 {
				fmt.Println("TODO: ExtensionConfigType")
			}
		case resourcev3.RuntimeType:
			if len(resources) > 0 {
				fmt.Println("TODO: RuntimeType")
			}
		case resourcev3.ThriftRouteType:
			if len(resources) > 0 {
				fmt.Println("TODO: ThriftRouteType")
			}
		}
	}

	return
}

// Convert xDS cache resources to Endpoints
func GetEndpoints(resources []types.Resource) ([]Endpoint, error) {
	var endpoints []Endpoint

	if len(resources) == 0 {
		return endpoints, nil
	}

	for _, r := range resources {
		endpoint, err := getEndpoint(r)
		if err != nil {
			return nil, err
		}

		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

// Convert xDS cache resource to Endpoint
func getEndpoint(resource types.Resource) (Endpoint, error) {
	var endpoint Endpoint

	v3Endpoint, ok := resource.(*endpointv3.Endpoint)
	if !ok {
		return endpoint, errors.Wrap(errors.New("getting resource not Endpoint"), fmt.Sprintf("Resource: %+v", resource))
	}

	endpoint.Address = v3Endpoint.GetAddress().GetSocketAddress().GetAddress()
	endpoint.Port = v3Endpoint.GetAddress().GetSocketAddress().GetPortValue()

	return endpoint, nil
}

// Convert xDS cache resources to Clusters
func GetClusters(resources []types.Resource) ([]Cluster, error) {
	var clusters []Cluster

	if len(resources) == 0 {
		return clusters, nil
	}

	for _, r := range resources {
		cluster, err := getCluster(r)
		if err != nil {
			return clusters, err
		}

		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// getCluster retrieves the Cluster object from the given resource.
// It returns the Cluster object and an error if the resource is not of type *clusterv3.Cluster.
// The resource parameter represents the input resource.
// The returned Cluster object contains the name, cluster type, load balancing policy, and endpoints.
func getCluster(resource types.Resource) (Cluster, error) {
	var cluster Cluster

	v3cluster, ok := resource.(*clusterv3.Cluster)
	if !ok {
		return cluster, errors.Wrap(errors.New("getting resource not Cluster"), fmt.Sprintf("Resource: %v", resource))
	}

	cluster.Name = v3cluster.GetName()
	cluster.ClusterType = fmt.Sprint(v3cluster.GetType())
	cluster.LbPolicy = fmt.Sprint(v3cluster.GetLbPolicy())
	cluster.Endpoints = getEndpointsFromLocalityLbEndpoints(v3cluster.GetLoadAssignment().GetEndpoints())

	return cluster, nil
}

// getEndpoints returns a slice of endpoints from a slice of LocalityLbEndpoints.
func getEndpointsFromLocalityLbEndpoints(llbes []*endpointv3.LocalityLbEndpoints) []Endpoint {
	endpoints := make([]Endpoint, 0)
	for _, llbe := range llbes {
		for _, lbe := range llbe.GetLbEndpoints() {
			var e Endpoint
			address := lbe.GetEndpoint().GetAddress().GetSocketAddress()
			if address != nil {
				e = Endpoint{
					Address: address.GetAddress(),
					Port:    address.GetPortValue(),
				}
			}
			endpoints = append(endpoints, e)
		}
	}
	return endpoints
}

// GetRoutes retrieves the routes from the given list of resources.
// It returns a slice of Route objects and an error if any occurred.
func GetRoutes(resources []types.Resource) ([]Route, error) {
	var routes []Route

	if len(resources) == 0 {
		return routes, nil
	}

	for _, r := range resources {
		route, err := getRoute(r)
		if err != nil {
			return routes, err
		}

		routes = append(routes, route)
	}

	return routes, nil
}

// getRoute is a function that takes a resource of type types.Resource as an input.
// It attempts to cast the resource to a RouteConfiguration (v3route).
// If the casting is not successful, it returns an error.
// If the casting is successful, it iterates over the VirtualHosts in the RouteConfiguration.
// For each VirtualHost, it appends the domain and routes to a new VirtualHost object (vh).
// It also appends any RequestHeadersToAdd to the VirtualHost object.
// Finally, it appends the VirtualHost object to the route and returns the route.
func getRoute(resource types.Resource) (Route, error) {
	var route Route

	v3route, ok := resource.(*routev3.RouteConfiguration)
	if !ok {
		return route, errors.Wrap(errors.New("getting resource not RouteConfiguration"), fmt.Sprintf("Resource: %+v", resource))
	}

	route.Name = v3route.GetName()

	for _, v3vh := range v3route.VirtualHosts {
		vh := VirtualHost{
			Domains: v3vh.Domains,
		}

		for _, v3r := range v3vh.Routes {
			vh.Routes = append(vh.Routes, v3r.GetName())
		}

		for _, v3rhta := range v3vh.RequestHeadersToAdd {
			vh.RequestsHeadersToAdds = append(vh.RequestsHeadersToAdds, RequestsHeadersToAdd{
				Action: fmt.Sprint(v3rhta.AppendAction),
				Header: Heades{
					Name:  v3rhta.Header.Key,
					Value: v3rhta.Header.Value,
				},
			})
		}

		route.VirtualHosts = append(route.VirtualHosts, vh)
	}

	return route, nil
}

// GetListeners returns a slice of Listener objects extracted from the given resources.
// If the resources slice is empty, it returns an empty slice of listeners.
func GetListeners(resources []types.Resource) ([]Listener, error) {
	var listeners []Listener

	if len(resources) == 0 {
		return listeners, nil
	}

	for _, r := range resources {
		listener, err := getListener(r)
		if err != nil {
			return listeners, err
		}

		listeners = append(listeners, listener)
	}

	return listeners, nil
}

func getListener(resource types.Resource) (Listener, error) {
	var listener Listener

	v3listener, ok := resource.(*listenerv3.Listener)
	if !ok {
		return listener, errors.Wrap(errors.New("getting resource not Listener"), fmt.Sprintf("Resource: %+v", resource))
	}

	listener.Name = v3listener.GetName()
	listener.Address = Address{
		Bind: v3listener.GetAddress().GetSocketAddress().GetAddress(),
		Port: v3listener.GetAddress().GetSocketAddress().GetPortValue(),
	}

	for _, v3fc := range v3listener.FilterChains {
		fc := FilterChain{
			Name: v3fc.GetName(),
		}

		if v3fc.FilterChainMatch != nil {
			fc.FilterChainMatch = &FilterChainMatch{
				Domains: v3fc.FilterChainMatch.ServerNames,
			}
		}

		if v3fc.TransportSocket != nil {
			fc.TransportSocket = &TransportSocket{
				Name: v3fc.TransportSocket.GetName(),
			}
		}

		for _, v3f := range v3fc.Filters {
			var filter Filter

			v3ftc, ok := v3f.ConfigType.(*listenerv3.Filter_TypedConfig)
			if !ok {
				return listener, errors.Wrap(errors.New("getting Filter not Filter_TypedConfig"), fmt.Sprintf("Filter: %+v", v3f))
			}

			switch v3ftc.TypedConfig.TypeUrl {
			case "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager":
				var err error
				filter, err = getFilterFromHttpConnectionManager(v3ftc.TypedConfig.GetValue())
				if err != nil {
					return listener, err
				}
				filter.FType = "HTTP"
			case "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy":
				var err error
				filter, err = getFilterFromTcpProxy(v3ftc.TypedConfig.GetValue())
				if err != nil {
					return listener, err
				}
				filter.FType = "TCP"
			default:
				return listener, errors.Wrap(errors.New("getting Filter Type not supported"), fmt.Sprintf("Filter: %+v", v3f))
			}

			fc.Filters = append(fc.Filters, filter)
		}

		listener.FilterChains = append(listener.FilterChains, fc)
	}

	return listener, nil
}

func getFilterFromHttpConnectionManager(str []byte) (Filter, error) {
	var filter Filter

	v3hcm := &hcmv3.HttpConnectionManager{}
	err := proto.Unmarshal(str, v3hcm)
	if err != nil {
		return filter, errors.Wrap(err, "Invalid listener filter config")
	}

	filter.StatPrefix = v3hcm.StatPrefix

	for _, hf := range v3hcm.GetHttpFilters() {
		filter.HttpFilters = append(filter.HttpFilters, hf.Name)
	}

	if v3hcm.GetRouteSpecifier() != nil {
		switch v3hcm.GetRouteSpecifier().(type) {
		case *hcmv3.HttpConnectionManager_Rds:
			v3rs := v3hcm.GetRouteSpecifier().(*hcmv3.HttpConnectionManager_Rds)
			route := v3rs.Rds.GetRouteConfigName()
			filter.RDS = &route
		case *hcmv3.HttpConnectionManager_RouteConfig:
			v3rs := v3hcm.GetRouteSpecifier().(*hcmv3.HttpConnectionManager_RouteConfig)
			v3rc := v3rs.RouteConfig
			route, err := getRoute(v3rc)
			if err != nil {
				return filter, err
			}
			filter.Route = &route
		}
	}

	return filter, nil
}

func getFilterFromTcpProxy(str []byte) (Filter, error) {
	var filter Filter

	tcpv3 := &tcpv3.TcpProxy{}
	err := proto.Unmarshal(str, tcpv3)
	if err != nil {
		return filter, errors.Wrap(err, "Invalid listener filter config")
	}

	filter.StatPrefix = tcpv3.StatPrefix
	cluster := tcpv3.GetCluster()
	filter.Cluster = &cluster

	return filter, nil
}

func getSecrets(resources []types.Resource) ([]Secret, error) {
	var secrets []Secret

	if len(resources) == 0 {
		return secrets, nil
	}
	for _, r := range resources {
		secret, err := getSecret(r)
		if err != nil {
			return secrets, err
		}

		secrets = append(secrets, secret)
	}

	return secrets, nil
}

func getSecret(resource types.Resource) (Secret, error) {
	var secret Secret

	v3secret, ok := resource.(*tlsv3.Secret)
	if !ok {
		return secret, errors.Wrap(errors.New("getting resource not Secret"), fmt.Sprintf("Resource: %+v", resource))
	}

	secret.Name = v3secret.GetName()

	return secret, nil
}
