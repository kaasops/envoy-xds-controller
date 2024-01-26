package api

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"

	"github.com/kaasops/envoy-xds-controller/pkg/xds/api/v1/handlers"

	docs "github.com/kaasops/envoy-xds-controller/docs/cacheRestAPI"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Client struct {
	Cache *xdscache.Cache
}

func New(cache *xdscache.Cache) *Client {
	return &Client{
		Cache: cache,
	}
}

func (c *Client) Run(port int, cacheAPIScheme, cacheAPIAddr string) error {
	server := gin.Default()

	// TODO: Fix CORS policy (don't enable for all origins)
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	handlers.RegisterRoutes(server, c.Cache)

	// Register swagger
	docs.SwaggerInfo.Schemes = []string{cacheAPIScheme}
	docs.SwaggerInfo.Host = cacheAPIAddr
	url := ginSwagger.URL(fmt.Sprintf("%v://%v/swagger/doc.json", cacheAPIScheme, cacheAPIAddr))
	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	// Run server
	if err := server.Run(fmt.Sprintf(":%d", port)); err != nil {
		return err
	}

	return nil
}
