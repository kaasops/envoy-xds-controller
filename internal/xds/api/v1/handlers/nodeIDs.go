package handlers

import (
	"sort"

	"github.com/gin-gonic/gin"
)

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
