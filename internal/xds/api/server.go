package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kaasops/envoy-xds-controller/internal/filewatcher"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/permissions/v1/permissionsv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/util/v1/utilv1connect"

	"github.com/kaasops/envoy-xds-controller/internal/grpcapi/virtualservice"

	"connectrpc.com/grpcreflect"
	"github.com/casbin/casbin/v2"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/access_group/v1/access_groupv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/access_log_config/v1/access_log_configv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/cluster/v1/clusterv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/http_filter/v1/http_filterv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/listener/v1/listenerv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/node/v1/nodev1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/policy/v1/policyv1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/route/v1/routev1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1/virtual_servicev1connect"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1/virtual_service_templatev1connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ginzap "github.com/gin-contrib/zap"
	"go.uber.org/zap"

	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"

	gincors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"

	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/handlers"

	docs "github.com/kaasops/envoy-xds-controller/docs/api/cacheRestAPI"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Config struct {
	EnableDevMode bool
	Auth          struct {
		Enabled             bool
		IssuerURL           string
		ClientID            string
		ACL                 map[string][]string
		AccessControlModel  string
		AccessControlPolicy string
	}
	StaticResources struct {
		AccessGroups []string `json:"accessGroups"`
		NodeIDs      []string `json:"nodeIds"`
	}
}

type Client struct {
	Cache    *xdscache.SnapshotCache
	logger   *zap.Logger
	devMode  bool
	fWatcher *filewatcher.FileWatcher
	mu       *sync.RWMutex
	cfg      *Config
}

func New(
	cache *xdscache.SnapshotCache,
	cfg *Config,
	logger *zap.Logger,
	devMode bool,
	watcher *filewatcher.FileWatcher,
	staticResourcesPath string,
) (*Client, error) {

	mu := &sync.RWMutex{}

	err := watcher.Add(staticResourcesPath, func(_ string) {
		logger.Info("static resources config changed")

		data, err := os.ReadFile(staticResourcesPath)
		if err != nil {
			logger.Error("failed to read static resources config", zap.Error(err))
		}
		if err = json.Unmarshal(data, &cfg.StaticResources); err != nil {
			logger.Error("failed to unmarshal static resources config", zap.Error(err))
		}
		logger.Info("static resources config reloaded")
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		Cache:    cache,
		mu:       mu,
		cfg:      cfg,
		logger:   logger,
		devMode:  devMode,
		fWatcher: watcher,
	}, nil
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
	server.Use(gincors.New(gincors.Config{
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

func (c *Client) RunGRPC(port int, s *store.Store, mgrClient client.Client, targetNs string) error {
	mux := http.NewServeMux()

	path, handler := virtual_servicev1connect.NewVirtualServiceStoreServiceHandler(virtualservice.NewVirtualServiceStore(s, mgrClient, targetNs))
	mux.Handle(path, handler)
	path, handler = virtual_service_templatev1connect.NewVirtualServiceTemplateStoreServiceHandler(grpcapi.NewVirtualServiceTemplateStore(s))
	mux.Handle(path, handler)
	path, handler = listenerv1connect.NewListenerStoreServiceHandler(grpcapi.NewListenerStore(s))
	mux.Handle(path, handler)
	path, handler = access_log_configv1connect.NewAccessLogConfigStoreServiceHandler(grpcapi.NewAccessLogConfigStore(s))
	mux.Handle(path, handler)
	path, handler = routev1connect.NewRouteStoreServiceHandler(grpcapi.NewRouteStore(s))
	mux.Handle(path, handler)
	path, handler = http_filterv1connect.NewHTTPFilterStoreServiceHandler(grpcapi.NewHTTPFilterStore(s))
	mux.Handle(path, handler)
	path, handler = policyv1connect.NewPolicyStoreServiceHandler(grpcapi.NewPolicyStore(s))
	mux.Handle(path, handler)
	path, handler = clusterv1connect.NewClusterStoreServiceHandler(grpcapi.NewClusterStore(s))
	mux.Handle(path, handler)
	path, handler = nodev1connect.NewNodeStoreServiceHandler(grpcapi.NewNodeStore(c))
	mux.Handle(path, handler)
	path, handler = access_groupv1connect.NewAccessGroupStoreServiceHandler(grpcapi.NewAccessGroupStore(c))
	mux.Handle(path, handler)
	path, handler = utilv1connect.NewUtilsServiceHandler(grpcapi.NewUtilsService(s))
	mux.Handle(path, handler)

	reflector := grpcreflect.NewStaticReflector()
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	if c.cfg.Auth.Enabled {
		enforcer, err := casbin.NewEnforcer(c.cfg.Auth.AccessControlModel, c.cfg.Auth.AccessControlPolicy)
		if err != nil {
			return err
		}
		path, handler = permissionsv1connect.NewPermissionsServiceHandler(grpcapi.NewPermissionsService(enforcer, c))
		mux.Handle(path, handler)
		handler = mux

		if err := c.fWatcher.Add(c.cfg.Auth.AccessControlModel, func(_ string) {
			c.logger.Info("rbac model changed")

			if err := enforcer.LoadModel(); err != nil {
				c.logger.Error("failed to load rbac model", zap.Error(err))
				return
			}
			if err := enforcer.LoadPolicy(); err != nil {
				c.logger.Error("failed to load rbac policy", zap.Error(err))
				return
			}
			c.logger.Info("rbac policy and model reloaded")
		}); err != nil {
			return err
		}
		middleware, err := grpcapi.NewAuthMiddleware(c.cfg.Auth.IssuerURL, c.cfg.Auth.ClientID, enforcer)
		if err != nil {
			return err
		}
		handler = middleware.Wrap(mux)
	} else {
		handler = mux
	}

	go func() {
		_ = http.ListenAndServe(
			net.JoinHostPort("", strconv.Itoa(port)),
			// Use h2c so we can serve HTTP/2 without TLS.
			h2c.NewHandler(cors.AllowAll().Handler(handler), &http2.Server{}),
		)
	}()
	return nil
}

func (c *Client) GetNodeIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg.StaticResources.NodeIDs
}

func (c *Client) GetAccessGroups() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg.StaticResources.AccessGroups
}
