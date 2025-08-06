package handlers

import (
	"sort"

	"github.com/gin-gonic/gin"
)

type nodeIDWithVersions struct {
	NodeID   string            `json:"node_id"`
	Versions map[string]string `json:"versions"`
}

// getNodeIDs retrieves the exists node ID in xDS cache.
// @Summary Get exists node ID in xDS cache
// @Tags nodeid
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/nodeIDs [get]
func (h *handler) getNodeIDs(ctx *gin.Context) {
	nodeIDs := h.getAvailableNodeIDs(ctx)
	sort.Strings(nodeIDs)

	ctx.JSON(200, nodeIDs)
}

// getNodeIDsWithResourceVersions retrieves the exists node ID in xDS cache with their resource versions.
// @Summary Get exists node ID in xDS cache with resource versions
// @Tags nodeid
// @Accept json
// @Produce json
// @Success 200 {array} nodeIDWithVersions
// @Router /api/v1/nodeIDs/versions [get]
func (h *handler) getNodeIDsWithResourceVersions(ctx *gin.Context) {
	nodeIDs := h.getAvailableNodeIDs(ctx)
	sort.Strings(nodeIDs)

	result := make([]nodeIDWithVersions, 0, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		versionsMap, err := h.cache.GetVersions(nodeID)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		result = append(result, nodeIDWithVersions{
			NodeID:   nodeID,
			Versions: versionsMap,
		})
	}
	ctx.JSON(200, result)
}
