package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/internal/buildinfo"
)

// getBuildInfo godoc
// @Summary Get build information
// @Description Get version, commit hash, and build date information
// @Tags buildinfo
// @Accept json
// @Produce json
// @Success 200 {object} buildinfo.Info
// @Router /api/v1/buildinfo [get]
func (h *handler) getBuildInfo(c *gin.Context) {
	info := buildinfo.GetInfo()
	c.JSON(http.StatusOK, info)
}