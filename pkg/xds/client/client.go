package client

import (
	"fmt"

	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"

	"github.com/kaasops/envoy-xds-controller/pkg/xds/client/handlers"

	_ "github.com/kaasops/envoy-xds-controller/docs/cacheRestAPI"
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

func (c *Client) Run(port int) error {
	server := gin.Default()

	handlers.RegisterRoutes(server, c.Cache)

	// Register swagger docs
	url := ginSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", port))
	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	if err := server.Run(fmt.Sprintf(":%d", port)); err != nil {
		return err
	}

	return nil
}
