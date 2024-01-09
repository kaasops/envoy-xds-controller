package handlers

import (
	"github.com/gin-gonic/gin"
)

// getNodeIDs retrieves the exists node ID in xDS cache.
// @Summary Get exists node ID in xDS cache
// @Tags nodeids
// @Tags cache
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/nodeIDs [get]
func (h *handler) getNodeIDs(ctx *gin.Context) {
	nodeIDs := h.cache.GetNodeIDs()
	ctx.JSON(200, nodeIDs)
}
