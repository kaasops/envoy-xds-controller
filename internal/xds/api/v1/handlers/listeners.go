package handlers

import (
	"slices"

	"github.com/gin-gonic/gin"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type GetListenersResponse struct {
	Listeners []*listenerv3.Listener `json:"listeners"`
}

// getListeners retrieves the listeners for a specific node ID.
// @Summary Get listeners for a specific node ID
// @Tags listener
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param listener_name query string false "Listener name" format(string) example("listener-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetListenersResponse
// @Failure 400 {object} map[string]string "error": "node_id not found in cache", "node_id": nodeID
// @Router /api/v1/listeners [get]
func (h *handler) getListeners(ctx *gin.Context) {
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

	params, err := h.getParams(ctx.Request.URL.Query(), qParams)
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

	listeners, err := h.cache.GetListeners(params[nodeIDParamName][0])
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := GetListenersResponse{}

	if params[listenerParamName][0] != "" {
		for _, listener := range listeners {
			if listener.Name != params[listenerParamName][0] {
				continue
			}
			response.Listeners = []*listenerv3.Listener{listener}
			ctx.JSON(200, response)
			return
		}

		ctx.JSON(500, gin.H{"error": "listener not found", "name": params[listenerParamName][0]})
		return
	}

	response.Listeners = listeners

	ctx.JSON(200, response)
}
