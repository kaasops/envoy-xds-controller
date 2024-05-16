package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/kube/api/v1/handlers"
	"github.com/kaasops/envoy-xds-controller/pkg/kube/client"
)

func RegisterRoutes(router *gin.RouterGroup, client *client.VirtualServiceClient, cfg *config.Config) {
	handler := handlers.NewVirtualServiceHandler(client, cfg)

	router.GET("/virtualservices", handler.GetAllVirtualServices)
	router.GET("/virtualservices/wrong-state", handler.GetAllVirtualServicesWithWrongState)
	router.GET("/virtualservices/:name", handler.GetVirtualService)
	router.GET("/virtualservices/search", handler.GetVirtualServiceByNameAndNodeId)
	router.POST("/virtualservices", handler.CreateVirtualService)
	router.PUT("/virtualservices", handler.UpdateVirtualService)
	router.DELETE("/virtualservices/:name", handler.DeleteVirtualService)
}
