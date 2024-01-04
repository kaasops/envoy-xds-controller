package handlers

import (
	"slices"

	"github.com/gin-gonic/gin"

	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
)

var (
	queryTypes = []string{"clusters", "endpoints", "routes", "listeners", "secrets"}
)

type GetCacheResponse struct {
	Version   int                 `json:"version"`
	Endpoints []xdscache.Endpoint `json:"endpoints"`
	Clusters  []xdscache.Cluster  `json:"clusters"`
	Routes    []xdscache.Route    `json:"routes"`
	Listeners []xdscache.Listener `json:"listeners"`
	Secrets   []xdscache.Secret   `json:"secrets"`
}

// getCache retrieves the cache for a specific node ID.
// It checks the "node_id" query parameter in the request URL and verifies its existence in the cache.
// If the node ID is found, it retrieves the associated resources from the cache and constructs a response.
// The response includes the version of the cache and the clusters, endpoints, routes, and listeners associated with the node ID.
// If any error occurs during the retrieval or construction of the response, an appropriate error message is returned.
// The response is then sent back to the client as a JSON object.
// @Summary Get cache for a specific node ID
// @Tags cache
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID"
// @Param type query string false "Resource type" Enums(clusters, endpoints, routes, listeners, secrets)
// @Success 200 {object} GetCacheResponse
// @Failure 400 {object} map[string]string
// @Router /cache [get]
func (h *handler) getCache(ctx *gin.Context) {
	queryValues := ctx.Request.URL.Query()

	// Check node_id query parameter
	nodeID, ok := queryValues["node_id"]
	if !ok {
		ctx.JSON(400, gin.H{"error": "node_id is required"})
		return
	}
	if len(nodeID) != 1 {
		ctx.JSON(400, gin.H{"error": "only 1 node_id is allowed"})
		return
	}
	// Check node_id exist in cache
	nodeIDs := h.cache.GetNodeIDs()
	if !slices.Contains(nodeIDs, nodeID[0]) {
		ctx.JSON(400, gin.H{"error": "node_id not found in cache"})
		return
	}

	// Check type query parameter
	qTypes := queryValues["type"]
	for _, t := range qTypes {
		if !slices.Contains(queryTypes, t) {
			ctx.JSON(400, gin.H{"error": "invalid type", "Allowed types": queryTypes})
			return
		}
	}

	resources, version, err := h.cache.GetResources(nodeID[0])
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := GetCacheResponse{
		Version: version,
	}

	clusters, endpoints, routes, listeners, secrets, err := xdscache.GetResources(resources)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(qTypes) == 0 {
		response.Clusters = clusters
		response.Endpoints = endpoints
		response.Routes = routes
		response.Listeners = listeners
		response.Secrets = secrets
		ctx.JSON(200, response)
		return
	}

	for _, t := range qTypes {
		switch t {
		case "clusters":
			response.Clusters = clusters
		case "endpoints":
			response.Endpoints = endpoints
		case "routes":
			response.Routes = routes
		case "listeners":
			response.Listeners = listeners
		case "secrets":
			response.Secrets = secrets
		}
	}

	ctx.JSON(200, response)
}
