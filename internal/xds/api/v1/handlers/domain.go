package handlers

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"github.com/gin-gonic/gin"
)

type getDomainLocationResponse struct {
	Locations []Location `json:"locations"`
}

type Location struct {
	Listener           string `json:"listener"`
	FilterChain        string `json:"filter_chain"`
	Filter             string `json:"filter"`
	RouteConfiguration string `json:"route_configuration"`
}

// @Summary Get domain locations. Find filter chain, filter and route configuration. If Filter Chain don't have Filter Chain Match - ignored.
// @Tags domain
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string false "Listener name" format(string) example("listener-1") required(false) allowEmptyValue(false)
// @Param domain_name query string true "Domain name" format(string) example("example-domain") required(true) allowEmptyValue(false)
// @Success 200 {object} getDomainLocationResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/domainLocations [get]
func (h *handler) getDomainLocations(ctx *gin.Context) {
	params, err := h.getParamsForDomainRequests(ctx.Request.URL.Query())
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	domainName, err := h.getRequiredOnlyOneParam(ctx.Request.URL.Query(), domainParamName)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check node_id exist in cache
	nodeIDs := h.getAvailableNodeIDs(ctx)
	if !slices.Contains(nodeIDs, params[nodeIDParamName][0]) {
		ctx.JSON(400, gin.H{"error": "node_id not found in cache", "node_id": params[nodeIDParamName][0]})
		return
	}

	response := getDomainLocationResponse{}

	var listeners []*listenerv3.Listener

	if params[listenerParamName][0] == "" {
		listeners, err = h.getListenersAll(params[nodeIDParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
	} else {
		listener, err := h.getListenerByName(params[nodeIDParamName][0], params[listenerParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		listeners = []*listenerv3.Listener{listener}
	}

	for _, listener := range listeners {
		location := Location{
			Listener: listener.Name,
		}

		// find FilterChain for domain
		filterChain := h.getFilterChainForDomainByServerName(listener, domainName)
		if filterChain == nil {
			continue
		}

		location.FilterChain = filterChain.Name

		filter, err := h.findFilterByDomain(params[nodeIDParamName][0], filterChain, domainName)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if filter != nil {

			location.Filter = filter.Name

			rdsName := h.getRDSNameForFilter(filter)
			if rdsName != "" {
				location.RouteConfiguration = rdsName
			}
		}

		response.Locations = append(response.Locations, location)
	}

	if len(response.Locations) == 0 {
		ctx.JSON(500, gin.H{"error": "domain not found", "domain": domainName})
		return
	}

	ctx.JSON(200, response)
}

type getDomainsResponse struct {
	Domains []string `json:"domains"`
}

// @Summary Get domains for node_id and listener_name (Find in Filter Chain Match)
// @Tags domain
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string false "Listener name" format(string) example("listener-1") required(false) allowEmptyValue(false)
// @Success 200 {object} getDomainsResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/domains [get]
func (h *handler) getDomains(ctx *gin.Context) {
	params, err := h.getParamsForDomainRequests(ctx.Request.URL.Query())
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check node_id exist in cache
	nodeIDs := h.getAvailableNodeIDs(ctx)
	if !slices.Contains(nodeIDs, params[nodeIDParamName][0]) {
		ctx.JSON(400, gin.H{"error": "node_id not found in cache", "node_id": params[nodeIDParamName][0]})
		return
	}

	var listeners []*listenerv3.Listener

	if params[listenerParamName][0] == "" {
		listeners, err = h.getListenersAll(params[nodeIDParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
	} else {
		listener, err := h.getListenerByName(params[nodeIDParamName][0], params[listenerParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		listeners = []*listenerv3.Listener{listener}
	}

	response := getDomainsResponse{}

	for _, listener := range listeners {
		for _, filterChain := range listener.FilterChains {
			if filterChain.FilterChainMatch != nil {
				response.Domains = append(response.Domains, filterChain.FilterChainMatch.ServerNames...)
			}
		}
	}

	if len(response.Domains) == 0 {
		ctx.JSON(500, gin.H{"error": "domains not found"})
		return
	}

	ctx.JSON(200, response)
}

// getFilterChainForDomainByServerName returns filter chain for domain by server names
// if filter chain don't have Filter Chain Match - ignored
func (h *handler) getFilterChainForDomainByServerName(listener *listenerv3.Listener, domain string) *listenerv3.FilterChain {
	var lastFindServerName string

	var findFilterChain *listenerv3.FilterChain

	for _, filterChain := range listener.FilterChains {
		if filterChain.FilterChainMatch == nil {
			continue
		}

		serversNames := filterChain.FilterChainMatch.GetServerNames()

		if len(serversNames) > 0 {
			findServerName := findServerNameForDomain(serversNames, domain)
			if findServerName == "" {
				continue
			}
			if len(findServerName) >= len(lastFindServerName) {
				findFilterChain = filterChain
				lastFindServerName = findServerName
			}
		}
	}

	return findFilterChain
}

// findServerNameForDomain checks if server name exist in server names list
// ServerName can be wildcard (https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener_components.proto#config-listener-v3-filterchainmatch)
func findServerNameForDomain(serverNames []string, domain string) string {
	var findServerName string

	domainParts := strings.Split(domain, ".")

	// Reverse domain parts
	for i, j := 0, len(domainParts)-1; i < j; i, j = i+1, j-1 {
		domainParts[i], domainParts[j] = domainParts[j], domainParts[i]
	}

L1:
	for _, serverName := range serverNames {
		serverNameParts := strings.Split(serverName, ".")

		if len(serverNameParts) > len(domainParts) {
			continue
		}

		// Reverse server name parts
		for i, j := 0, len(serverNameParts)-1; i < j; i, j = i+1, j-1 {
			serverNameParts[i], serverNameParts[j] = serverNameParts[j], serverNameParts[i]
		}

		// Check if server name parts equal domain parts
		for i, serverNamePart := range serverNameParts {
			if serverNameParts[i] == "*" {
				if len(serverName) >= len(findServerName) {
					findServerName = serverName
				}
			}
			if serverNamePart != domainParts[i] {
				continue L1
			}

			if i == len(serverNameParts)-1 {
				if len(serverName) >= len(findServerName) {
					findServerName = serverName
				}
			}
		}

	}

	return findServerName
}

// findFilterByDomain returns filter for domain
// If filter not HTTP Connection Manager - ignored
func (h *handler) findFilterByDomain(nodeid string, filterChain *listenerv3.FilterChain, domain string) (*listenerv3.Filter, error) {
	for _, filter := range filterChain.Filters {
		hcmConfig := resourcev3.GetHTTPConnectionManager(filter)
		// Skip if filter not HttpConnectionManager
		if hcmConfig == nil {
			continue
		}

		switch hcmConfig.RouteSpecifier.(type) {
		case *hcmv3.HttpConnectionManager_Rds:
			RDSName := hcmConfig.GetRds().GetRouteConfigName()
			routeConfigurations, err := h.getRouteConfigurationByName(nodeid, RDSName)
			if err != nil {
				return nil, err
			}
			if existDomainInVirtualHosts(routeConfigurations.VirtualHosts, domain) {
				return filter, nil
			}
		case *hcmv3.HttpConnectionManager_RouteConfig:
			routeConfig := hcmConfig.GetRouteConfig()
			if existDomainInVirtualHosts(routeConfig.VirtualHosts, domain) {
				return filter, nil
			}
		case *hcmv3.HttpConnectionManager_ScopedRoutes:
			return nil, fmt.Errorf("ScopedRoutes not supported")
		}
	}
	return nil, fmt.Errorf("filter for domain %v not found", domain)
}

func existDomainInVirtualHosts(vhs []*routev3.VirtualHost, domain string) bool {
	for _, vh := range vhs {
		if slices.Contains(vh.Domains, "*") || slices.Contains(vh.Domains, domain) {
			return true
		}
	}

	return false
}

func (h *handler) getParamsForDomainRequests(queryValues url.Values) (map[string][]string, error) {
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
	}

	params, err := h.getParams(queryValues, qParams)
	if err != nil {
		return nil, err
	}

	return params, nil
}
