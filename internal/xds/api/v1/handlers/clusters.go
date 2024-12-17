package handlers

import (
	"net/url"
	"slices"

	"github.com/gin-gonic/gin"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

type GetClustersResponse struct {
	Clusters []*clusterv3.Cluster `json:"clusters"`
}

// getClusters retrieves the clusters for a specific node ID.
// @Summary Get clusters for a specific node ID.
// @Tags cluster
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1") required(true) allowEmptyValue(false)
// @Param cluster_name query string false "Cluster name" format(string) example("cluster-1") required(false) allowEmptyValue(true)
// @Success 200 {object} GetClustersResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/clusters [get]
func (h *handler) getClusters(ctx *gin.Context) {
	params, err := h.getParamsForClusterRequests(ctx.Request.URL.Query())
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

	var response GetClustersResponse

	if params[clustersParamName][0] != "" {
		cluster, err := h.getClusterByName(params[nodeIDParamName][0], params[clustersParamName][0])
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		response.Clusters = []*clusterv3.Cluster{cluster}
		ctx.JSON(200, response)
		return
	}

	clusters, err := h.getClustersAll(params[nodeIDParamName][0])
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response.Clusters = clusters
	ctx.JSON(200, response)
}

func (h *handler) getParamsForClusterRequests(queryValues url.Values) (map[string][]string, error) {
	qParams := []getParam{
		{
			name:     nodeIDParamName,
			required: true,
			onlyOne:  true,
		},
		{
			name:     clustersParamName,
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
