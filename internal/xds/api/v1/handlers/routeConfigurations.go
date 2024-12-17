package handlers

import (
	"net/url"
	"slices"

	"github.com/gin-gonic/gin"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type GetRouteConfigurationsResponse struct {
	RouteConfigurations []*routev3.RouteConfiguration `json:"routeConfigurations"`
}

// getRouteConfigurations retrieves the routesConfigurations for a specific node ID.
// If set param name, return only one route configuration
// @Summary Get routesConfigurations for a specific node ID. If set param name, return only one route configuration.
// @Tags routeConfiguration
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param route_configuration_name query string false "RouteConfiguration name" format(string) example("route-config-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetRouteConfigurationsResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/routeConfigurations [get]
func (h *handler) getRouteConfigurations(ctx *gin.Context) {
	params, err := h.getParamsForRouteConfigurationRequests(ctx.Request.URL.Query())
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

	var response GetRouteConfigurationsResponse

	// If param name set, return only one route configuration
	if params[routeConfigurationParamName][0] != "" {
		rc, err := h.getRouteConfigurationByName(params[nodeIDParamName][0], params[routeConfigurationParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		response.RouteConfigurations = []*routev3.RouteConfiguration{rc}
		ctx.JSON(200, response)
		return
	}

	rcs, err := h.getRouteConfigurationsAll(params[nodeIDParamName][0])
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	response.RouteConfigurations = rcs
	ctx.JSON(200, response)
}

func (h *handler) getParamsForRouteConfigurationRequests(queryValues url.Values) (map[string][]string, error) {
	qParams := []getParam{
		{
			name:     nodeIDParamName,
			required: true,
			onlyOne:  true,
		},
		{
			name:     routeConfigurationParamName,
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
