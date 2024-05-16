package api

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/kaasops/envoy-xds-controller/docs/kubeRestAPI"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/kube/api/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/kube/client"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"log"
	"time"
)

type Server struct {
	VirtualServiceClient *client.VirtualServiceClient
	Config               *config.Config
}

func NewServer(virtualServiceClient *client.VirtualServiceClient, config *config.Config) *Server {
	return &Server{VirtualServiceClient: virtualServiceClient, Config: config}
}

func (s *Server) Run(port int, scheme, addr string) error {
	server := gin.Default()
	// CORS configuration
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	apiV1 := server.Group("/api/v1")
	v1.RegisterRoutes(apiV1, s.VirtualServiceClient, s.Config)
	server.Static("/docs", "./docs")

	kubeAPIUrl := ginSwagger.URL(fmt.Sprintf("%v://%v/docs/kubeRestAPI/kube_swagger.json", scheme, addr))
	server.GET("/swagger/kube/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, kubeAPIUrl, ginSwagger.InstanceName("kubeSwagger")))

	if err := server.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

	return nil
}
