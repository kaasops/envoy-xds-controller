package handlers

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/gin-gonic/gin"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"

	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp_proxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"

	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"

	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type getFilterTypeResponse struct {
	FType string `json:"filter_type"`
}

// getFilterType retrieves the filter type for a specific Filter.
// @Summary Get filter type retrieves the filter type for a specific Filter
// @Tags filter
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string true "Listener name" format(string) example("listener-1") required(true) allowEmptyValue(false)
// @Param filter_chain_name query string false "Filter chain name" format(string) example("filter-chain-1") required(false) allowEmptyValue(true)
// @Param filter_name query string false "Filter name" format(string) example("filter-1") required(false) allowEmptyValue(true)
// @Success 200 {object} getFilterTypeResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/filterType [get]
func (h *handler) getFilterType(ctx *gin.Context) {
	filters, err := h.getRequestFilters(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(filters) != 1 {
		ctx.JSON(500, gin.H{"error": "filter_name is required if filter_chain have more than one filter"})
		return
	}

	response := getFilterTypeResponse{
		FType: filters[0].Name,
	}
	ctx.JSON(200, response)
}

type GetHCMFilterResponse struct {
	Filters []*hcmv3.HttpConnectionManager `json:"filters"`
}

// getFilters retrieves the filters for a specific Filter Chain, Listener and Node ID.
// @Summary Get Filters retrieves the filters for a specific Filter Chain, Listener and Node ID. (only http_connection_manager)
// @Tags filter
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string true "Listener name" format(string) example("listener-1") required(true) allowEmptyValue(false)
// @Param filter_chain_name query string false "Filter chain name. If not set works only if the listener has only one Filter Chain" format(string) example("filter-chain-1") required(false) allowEmptyValue(true)
// @Param filter_name query string false "Filter name" format(string) example("filter-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetHCMFilterResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/filters [get]
func (h *handler) getFilter(ctx *gin.Context) {
	filters, err := h.getRequestFilters(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := GetHCMFilterResponse{}

	for _, filter := range filters {
		hcmConfig := resourcev3.GetHTTPConnectionManager(filter)
		if hcmConfig == nil {
			ctx.JSON(500, gin.H{"error": "filter is not http_connection_manager"})
			return
		}

		response.Filters = append(response.Filters, hcmConfig)
	}

	ctx.JSON(200, response)
}

type GetTCPProxyFilterResponse struct {
	Filters []*tcp_proxyv3.TcpProxy `json:"filters"`
}

// getTCPProxyFilters retrieves the TCP Proxy filters for a specific Filter Chain, Listener and Node ID.
// @Summary Get TCP filters for a specific Filter Chain, Listener and Node ID.
// @Tags filter
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string true "Listener name" format(string) example("listener-1") required(true) allowEmptyValue(false)
// @Param filter_chain_name query string false "Filter chain name" format(string) example("filter-chain-1") required(false) allowEmptyValue(true)
// @Param filter_name query string false "Filter name" format(string) example("filter-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetTCPProxyFilterResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/filtersTCPProxy [get]
func (h *handler) getTCPProxyFilters(ctx *gin.Context) {
	filters, err := h.getRequestFilters(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := GetTCPProxyFilterResponse{}

	for _, filter := range filters {
		if typedConfig := filter.GetTypedConfig(); typedConfig != nil {
			tcpProxyConfig := &tcp_proxyv3.TcpProxy{}
			if err := typedConfig.UnmarshalTo(tcpProxyConfig); err != nil {
				ctx.JSON(500, gin.H{"error": "filter is not tcp_proxy"})
				return
			}

			response.Filters = append(response.Filters, tcpProxyConfig)
		}
	}

	ctx.JSON(200, response)
}

type GetHttpsFilterResponse struct {
	HttpFilters []*hcmv3.HttpFilter `json:"httpFilters"`
}

// getHTTPFilters retrieves the HTTP filters for a specific Filter.
// @Summary Get HTTP filters for a specific Filter.
// @Tags filter
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string true "Listener name" format(string) example("listener-1") required(true) allowEmptyValue(false)
// @Param filter_chain_name query string false "Filter chain name" format(string) example("filter-chain-1") required(false) allowEmptyValue(true)
// @Param filter_name query string false "Filter name" format(string) example("filter-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetHttpsFilterResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/httpFilters [get]
func (h *handler) getHTTPFilters(ctx *gin.Context) {
	filters, err := h.getRequestFilters(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := GetHttpsFilterResponse{}

	for _, filter := range filters {
		hcmConfig := resourcev3.GetHTTPConnectionManager(filter)
		response.HttpFilters = append(response.HttpFilters, hcmConfig.HttpFilters...)
	}

	ctx.JSON(200, response)
}

type GetHttpFilterRouterResponse struct {
	Router *routerv3.Router `json:"router"`
}

// getHTTPFilterRouter retrieves the HTTP filter router for a specific Filter.
// @Summary Get HTTP filter router for a specific Filter.
// @Tags filter
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string true "Listener name" format(string) example("listener-1") required(true) allowEmptyValue(false)
// @Param filter_chain_name query string false "Filter chain name" format(string) example("filter-chain-1") required(false) allowEmptyValue(true)
// @Param filter_name query string false "Filter name" format(string) example("filter-1") required(false) allowEmptyValue(true)
// @Param http_filter_name query string false "HTTP filter name" format(string) example("http-filter-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetHttpFilterRouterResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/httpFilterRouter [get]
func (h *handler) getHTTPFilterRouter(ctx *gin.Context) {
	filters, err := h.getRequestFilters(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(filters) != 1 {
		ctx.JSON(500, gin.H{"error": "filter_name is required if filter_chain have more than one filter"})
		return
	}

	hfName, err := h.getNotRequiredOnlyOneParam(ctx.Request.URL.Query(), httpFilterParamName)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	hf, err := h.getRequestHTTPFilter(filters[0], hfName)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if typedConfig := hf.GetTypedConfig(); typedConfig != nil {
		routeConfig := &routerv3.Router{}
		if err := typedConfig.UnmarshalTo(routeConfig); err != nil {
			ctx.JSON(500, gin.H{"error": "filter is not router"})
			return
		}

		response := GetHttpFilterRouterResponse{
			Router: routeConfig,
		}

		ctx.JSON(200, response)
		return
	}
}

type GetHttpFilterRBACResponse struct {
	Router *rbacv3.RBAC `json:"rbac"`
}

// getHTTPFilterRBAC retrieves the HTTP filter RBAC for a specific Filter.
// @Summary Get HTTP filter RBAC for a specific Filter.
// @Tags filter
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string true "Listener name" format(string) example("listener-1") required(true) allowEmptyValue(false)
// @Param filter_chain_name query string false "Filter chain name" format(string) example("filter-chain-1") required(false) allowEmptyValue(true)
// @Param filter_name query string false "Filter name" format(string) example("filter-1") required(false) allowEmptyValue(true)
// @Param http_filter_name query string false "HTTP filter name" format(string) example("http-filter-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetHttpFilterRBACResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/httpFilterRBAC [get]
func (h *handler) getHTTPFilterRBAC(ctx *gin.Context) {
	filters, err := h.getRequestFilters(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(filters) != 1 {
		ctx.JSON(500, gin.H{"error": "filter_name is required if filter_chain have more than one filter"})
		return
	}

	hfName, err := h.getNotRequiredOnlyOneParam(ctx.Request.URL.Query(), httpFilterParamName)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	hf, err := h.getRequestHTTPFilter(filters[0], hfName)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if typedConfig := hf.GetTypedConfig(); typedConfig != nil {
		routeConfig := &rbacv3.RBAC{}
		if err := typedConfig.UnmarshalTo(routeConfig); err != nil {
			ctx.JSON(500, gin.H{"error": "filter is not RBAC"})
			return
		}

		response := GetHttpFilterRBACResponse{
			Router: routeConfig,
		}

		ctx.JSON(200, response)
		return
	}
}

func (h *handler) getRequestFilters(ctx *gin.Context) ([]*listenerv3.Filter, error) {

	queryValues := ctx.Request.URL.Query()
	var filters []*listenerv3.Filter

	params, err := h.getParamsForFilterRequests(queryValues)
	if err != nil {
		return nil, err
	}

	// Check node_id exist in cache
	nodeIDs := h.getAvailableNodeIDs(ctx)
	if !slices.Contains(nodeIDs, params[nodeIDParamName][0]) {
		return nil, fmt.Errorf("node_id not found in cache. node_id: %v", params[nodeIDParamName][0])
	}

	listener, err := h.getListenerByName(params[nodeIDParamName][0], params[listenerParamName][0])
	if err != nil {
		return nil, err
	}

	var filterChain *listenerv3.FilterChain
	if params[filterChainParamName][0] != "" {
		filterChain, err = h.getFilterChainByName(listener, params[filterChainParamName][0])
		if err != nil {
			return nil, err
		}
	} else {
		if len(listener.FilterChains) != 1 {
			return nil, fmt.Errorf("filter_chain_name is required if listener have more than one filter_chain")
		}

		filterChain = listener.FilterChains[0]
	}

	if params[filterParamName][0] != "" {
		filter, err := h.getFilterByName(filterChain, params[filterParamName][0])
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	} else {
		filters = append(filters, filterChain.Filters...)
	}

	return filters, err
}

func (h *handler) getRequestHTTPFilter(filter *listenerv3.Filter, name string) (*hcmv3.HttpFilter, error) {
	hcmConfig := resourcev3.GetHTTPConnectionManager(filter)
	if hcmConfig == nil {
		return nil, fmt.Errorf("filter is not http_connection_manager")
	}

	if name == "" {
		if len(hcmConfig.HttpFilters) != 1 {
			return nil, fmt.Errorf("http_filter_name is required if http_filters have more than one filter")
		}

		return hcmConfig.HttpFilters[0], nil
	}

	for _, hFilter := range hcmConfig.HttpFilters {
		if hFilter.Name == name {
			return hFilter, nil
		}
	}

	return nil, fmt.Errorf("http_filter not found, name: %v", name)
}

func (h *handler) getParamsForFilterRequests(queryValues url.Values) (map[string][]string, error) {
	qParams := []getParam{
		{
			name:     nodeIDParamName,
			required: true,
			onlyOne:  true,
		},
		{
			name:     listenerParamName,
			required: false,
			onlyOne:  true,
		},
		{
			name:     filterChainParamName,
			required: false,
			onlyOne:  true,
		},
		{
			name:     filterParamName,
			required: false,
			onlyOne:  true,
		},
	}

	params, err := h.getParams(queryValues, qParams)
	if err != nil {
		return nil, err
	}

	return params, nil
}
