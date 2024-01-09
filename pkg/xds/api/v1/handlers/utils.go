package handlers

import (
	"fmt"
	"net/url"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type getParam struct {
	name     string
	required bool
	onlyOne  bool
}

var (
	nodeIDParamName             = "node_id"
	listenerParamName           = "listener_name"
	filterChainParamName        = "filter_chain_name"
	filterParamName             = "filter_name"
	httpFilterParamName         = "http_filter_name"
	routeConfigurationParamName = "route_configuration_name"
	clustersParamName           = "cluster_name"
	secretParamName             = "secret_name"
)

// ****
// Methods for work with query parameters
// ****

func (h *handler) getParams(queryValues url.Values, params []getParam) (map[string][]string, error) {
	paramsValues := map[string][]string{}

	for _, param := range params {
		if param.required {
			paramValue, err := h.getRequiredOnlyOneParam(queryValues, param.name)
			if err != nil {
				return nil, err
			}
			paramsValues[param.name] = []string{paramValue}
			continue
		}

		paramValue, err := h.getNotRequiredOnlyOneParam(queryValues, param.name)
		if err != nil {
			return nil, err
		}
		paramsValues[param.name] = []string{paramValue}
	}

	return paramsValues, nil

}

func (h *handler) getNotRequiredOnlyOneParam(queryValues url.Values, param string) (string, error) {
	// Get parameter
	paramValue, ok := queryValues[param]
	if !ok {
		return "", nil
	}

	// Check set only one param
	if len(paramValue) != 1 {
		return "", fmt.Errorf("only 1 %s is allowed", param)
	}

	return paramValue[0], nil
}

func (h *handler) getRequiredOnlyOneParam(queryValues url.Values, param string) (string, error) {
	// Get parameter
	paramValue, ok := queryValues[param]
	if !ok {
		return "", fmt.Errorf("param %s is required", param)
	}

	// Check set only one param
	if len(paramValue) != 1 {
		return "", fmt.Errorf("only 1 %s is allowed", param)
	}

	return paramValue[0], nil
}

// ****
// Methods for work xDS Cache
// ****

func (h *handler) getListenerByName(nodeID string, listenerName string) (*listenerv3.Listener, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	for _, listener := range resources[resourcev3.ListenerType] {
		v3listener, ok := listener.(*listenerv3.Listener)
		if !ok {
			return nil, fmt.Errorf("listener is not v3")
		}

		if v3listener.Name != listenerName {
			continue
		}

		return v3listener, nil
	}

	return nil, fmt.Errorf("listener %v not found", listenerName)
}

// getRouteConfigurationByName returns route configuration by name
func (h *handler) getRouteConfigurationByName(nodeID string, routeConfigurationName string) (*routev3.RouteConfiguration, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	for _, rc := range resources[resourcev3.RouteType] {
		v3rc, ok := rc.(*routev3.RouteConfiguration)
		if !ok {
			return nil, fmt.Errorf("route is not v3")
		}

		if v3rc.Name != routeConfigurationName {
			continue
		}

		return v3rc, nil
	}

	return nil, fmt.Errorf("RouteConfiguration %v not found", routeConfigurationName)
}

func (h *handler) getRouteConfigurationsAll(nodeID string) ([]*routev3.RouteConfiguration, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	rcs := []*routev3.RouteConfiguration{}

	for _, rc := range resources[resourcev3.RouteType] {
		v3rc, ok := rc.(*routev3.RouteConfiguration)
		if !ok {
			return nil, fmt.Errorf("route is not v3")
		}

		rcs = append(rcs, v3rc)
	}

	return rcs, nil
}

func (h *handler) getClusterByName(nodeID, clusterName string) (*clusterv3.Cluster, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	for _, cluster := range resources[resourcev3.ClusterType] {
		v3cluster, ok := cluster.(*clusterv3.Cluster)
		if !ok {
			return nil, fmt.Errorf("cluster is not v3")
		}

		if v3cluster.Name != clusterName {
			continue
		}

		return v3cluster, nil
	}

	return nil, fmt.Errorf("cluster %v not found", clusterName)
}

func (h *handler) getClustersAll(nodeID string) ([]*clusterv3.Cluster, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	clusters := []*clusterv3.Cluster{}

	for _, cluster := range resources[resourcev3.ClusterType] {
		v3cluster, ok := cluster.(*clusterv3.Cluster)
		if !ok {
			return nil, fmt.Errorf("cluster is not v3")
		}

		clusters = append(clusters, v3cluster)
	}

	return clusters, nil
}

func (h *handler) getSecretByName(nodeID, secretName string) (*tlsv3.Secret, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	for _, secret := range resources[resourcev3.SecretType] {
		v3secret, ok := secret.(*tlsv3.Secret)
		if !ok {
			return nil, fmt.Errorf("cluster is not v3")
		}

		if v3secret.Name != secretName {
			continue
		}

		clearSecretData(v3secret)

		return v3secret, nil
	}

	return nil, fmt.Errorf("cluster %v not found", secretName)
}

func (h *handler) getSecretsAll(nodeID string) ([]*tlsv3.Secret, error) {
	resources, _, err := h.cache.GetResources(nodeID)
	if err != nil {
		return nil, err
	}

	secrets := []*tlsv3.Secret{}

	for _, secret := range resources[resourcev3.SecretType] {
		v3secret, ok := secret.(*tlsv3.Secret)
		if !ok {
			return nil, fmt.Errorf("secret is not v3")
		}
		clearSecretData(v3secret)

		secrets = append(secrets, v3secret)
	}

	return secrets, nil
}

func clearSecretData(secret *tlsv3.Secret) {
	switch secret.Type.(type) {
	case *tlsv3.Secret_TlsCertificate:
		secret.Type = &tlsv3.Secret_TlsCertificate{}
	case *tlsv3.Secret_SessionTicketKeys:
		secret.Type = &tlsv3.Secret_SessionTicketKeys{}
	case *tlsv3.Secret_ValidationContext:
		secret.Type = &tlsv3.Secret_ValidationContext{}
	case *tlsv3.Secret_GenericSecret:
		secret.Type = &tlsv3.Secret_GenericSecret{}
	}
}

func (h *handler) getFilterChainByName(listener *listenerv3.Listener, filterChainName string) (*listenerv3.FilterChain, error) {
	for _, filterChain := range listener.FilterChains {
		if filterChain.Name != filterChainName {
			continue
		}

		return filterChain, nil
	}

	return nil, fmt.Errorf("filter chain %v not found", filterChainName)
}

func (h *handler) getFilterByName(filterChain *listenerv3.FilterChain, filterName string) (*listenerv3.Filter, error) {
	for _, filter := range filterChain.Filters {
		if filter.Name != filterName {
			continue
		}

		return filter, nil
	}

	return nil, fmt.Errorf("filter %v not found", filterName)
}
