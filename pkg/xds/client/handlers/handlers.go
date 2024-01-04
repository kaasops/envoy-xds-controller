package handlers

import (
	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
)

// @version 1.0
// @title Envoy XDS Cache Rest API
// @description This is a sample server for Envoy XDS Cache Rest API.
// @BasePath /
// @schemes http

type handler struct {
	cache *xdscache.Cache
}

func RegisterRoutes(r *gin.Engine, cache *xdscache.Cache) {
	h := &handler{cache: cache}

	routes := r.Group("/")
	routes.GET("/cache", h.getCache)
}
