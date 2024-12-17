package handlers

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
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
	domainParamName             = "domain_name"
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

func (h *handler) getAvailableNodeIDs(ctx *gin.Context) []string {
	allNodeIDs := h.cache.GetNodeIDs()
	if v, exists := ctx.Get(middlewares.AvailableNodeIDs); exists {
		availableNodeIDs := v.(map[string]struct{})
		nodeIDs := make([]string, 0, len(availableNodeIDs))
		for _, nodeID := range allNodeIDs {
			if _, ok := availableNodeIDs[nodeID]; ok {
				nodeIDs = append(nodeIDs, nodeID)
			}
		}
		return nodeIDs
	}
	return allNodeIDs
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
	listeners, err := h.cache.GetListeners(nodeID)
	if err != nil {
		return nil, err
	}

	for _, listener := range listeners {
		if listener.Name != listenerName {
			continue
		}
		return listener, nil
	}

	return nil, fmt.Errorf("listener %v not found", listenerName)
}

func (h *handler) getListenersAll(nodeID string) ([]*listenerv3.Listener, error) {
	return h.cache.GetListeners(nodeID)
}

// getRouteConfigurationByName returns route configuration by name
func (h *handler) getRouteConfigurationByName(nodeID string, routeConfigurationName string) (*routev3.RouteConfiguration, error) {
	resources, err := h.cache.GetRouteConfigurations(nodeID)
	if err != nil {
		return nil, err
	}

	for _, rc := range resources {
		if rc.Name != routeConfigurationName {
			continue
		}
		return rc, nil
	}

	return nil, fmt.Errorf("RouteConfiguration %v not found", routeConfigurationName)
}

func (h *handler) getRouteConfigurationsAll(nodeID string) ([]*routev3.RouteConfiguration, error) {
	return h.cache.GetRouteConfigurations(nodeID)
}

func (h *handler) getClusterByName(nodeID, clusterName string) (*clusterv3.Cluster, error) {
	resources, err := h.cache.GetClusters(nodeID)
	if err != nil {
		return nil, err
	}

	for _, cluster := range resources {
		if cluster.Name != clusterName {
			continue
		}
		return cluster, nil
	}

	return nil, fmt.Errorf("cluster %v not found", clusterName)
}

func (h *handler) getClustersAll(nodeID string) ([]*clusterv3.Cluster, error) {
	return h.cache.GetClusters(nodeID)
}

func (h *handler) getSecretByName(nodeID, secretName string) (*tlsv3.Secret, error) {
	secrets, err := h.cache.GetSecrets(nodeID)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		if secret.Name != secretName {
			continue
		}
		clearSecretData(secret)
		return secret, nil
	}

	return nil, fmt.Errorf("cluster %v not found", secretName)
}

func (h *handler) getSecretsAll(nodeID string, clearSecret bool) ([]*tlsv3.Secret, error) {
	resources, err := h.cache.GetSecrets(nodeID)
	if err != nil {
		return nil, err
	}

	secrets := make([]*tlsv3.Secret, 0, len(resources))

	for _, secret := range resources {
		if clearSecret {
			clearSecretData(secret)
		}
		secrets = append(secrets, secret)
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

func (h *handler) getRDSNameForFilter(filter *listenerv3.Filter) string {
	hcmConfig := resourcev3.GetHTTPConnectionManager(filter)
	// Skip if filter not HttpConnectionManager
	if hcmConfig == nil {
		return ""
	}

	_, ok := hcmConfig.RouteSpecifier.(*hcmv3.HttpConnectionManager_Rds)
	if !ok {
		return ""
	}

	return hcmConfig.GetRds().GetRouteConfigName()
}
