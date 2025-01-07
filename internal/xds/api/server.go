package api

import (
	"fmt"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"go.uber.org/zap"

	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"

	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/handlers"

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
	Cache   *xdscache.SnapshotCache
	cfg     *Config
	logger  *zap.Logger
	devMode bool
}

func New(cache *xdscache.SnapshotCache, cfg *Config, logger *zap.Logger, devMode bool) *Client {
	return &Client{
		Cache:   cache,
		cfg:     cfg,
		logger:  logger,
		devMode: devMode,
	}
}

func (c *Client) Run(port int, cacheAPIScheme, cacheAPIAddr string) error {
	server := gin.New()
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, _ int) {
		c.logger.Debug(fmt.Sprintf("endpoint %v %v %v", httpMethod, absolutePath, handlerName))
	}
	if c.devMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	server.Use(ginzap.Ginzap(c.logger, time.RFC3339, true))
	server.Use(ginzap.RecoveryWithZap(c.logger, true))

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
