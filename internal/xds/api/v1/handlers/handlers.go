package handlers

import (
	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

// @version 1.0
// @title Envoy XDS Cache Rest API
// @description This is a sample server for Envoy XDS Cache Rest API.
// @BasePath /
// @schemes http

type handler struct {
	cache *xdscache.SnapshotCache
}

var (
	version = "/api/v1"
)

func RegisterRoutes(r *gin.Engine, cache *xdscache.SnapshotCache) {
	h := &handler{cache: cache}

	routes := r.Group(version)

	routes.GET("/nodeIDs", h.getNodeIDs)

	// ********** Get Listeners **********
	// Get Listeners
	routes.GET("/listeners", h.getListeners)

	// ********** Get Filters **********
	// Get Filter Type
	routes.GET("/filterType", h.getFilterType)

	// Get Http Connection Manager Filters
	routes.GET("/filters", h.getFilter)
	routes.GET("/filtersHCM", h.getFilter)
	// Get Http Filters
	routes.GET("/httpFilters", h.getHTTPFilters)
	routes.GET("/httpFilterRouter", h.getHTTPFilterRouter)
	routes.GET("/httpFilterRBAC", h.getHTTPFilterRBAC)

	// TCP Filters
	routes.GET("/filtersTCPProxy", h.getTCPProxyFilters)

	// ********** Get RouteConfigurations **********
	routes.GET("/routeConfigurations", h.getRouteConfigurations)

	// ********** Get Clusters **********
	routes.GET("/clusters", h.getClusters)

	// ********** Get Secrets **********
	routes.GET("/secrets", h.getSecrets)
	routes.GET("/secrets/:namespace/:name", h.getSecretByNamespacedName)

	// ********** Get Domain info **********
	routes.GET("/domainLocations", h.getDomainLocations)
	routes.GET("/domains", h.getDomains)
}
