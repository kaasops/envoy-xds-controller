package api

import (
	"fmt"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/api/v1/middlewares"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"

	"github.com/kaasops/envoy-xds-controller/pkg/xds/api/v1/handlers"

	docs "github.com/kaasops/envoy-xds-controller/docs/cacheRestAPI"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Config struct {
	EnableDevMode bool
	Auth          struct {
		Enabled   bool
		IssuerURL string
		ClientID  string
		ACL       map[string][]string
	}
}

type Client struct {
	Cache *xdscache.Cache
	cfg   *Config
}

func New(cache *xdscache.Cache, cfg *Config) *Client {
	return &Client{
		Cache: cache,
		cfg:   cfg,
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

	if c.cfg.Auth.Enabled {
		authMiddleware, err := middlewares.NewAuth(c.cfg.Auth.IssuerURL, c.cfg.Auth.ClientID, c.cfg.Auth.ACL, c.cfg.EnableDevMode)
		if err != nil {
			return fmt.Errorf("failed to create auth middleware: %w", err)
		}
		server.Use(authMiddleware.HandlerFunc)
	}

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
