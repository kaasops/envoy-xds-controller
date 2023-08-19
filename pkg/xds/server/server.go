package server

import (
	"context"
	"fmt"
	"net"
	"time"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	grpcKeepaliveTime        = 300 * time.Second
	grpcKeepaliveTimeout     = 50 * time.Second
	grpcKeepaliveMinTime     = 300 * time.Second
	grpcMaxConcurrentStreams = 10000000
)

type Server struct {
	xDSServer serverv3.Server
}

func New(cache cachev3.SnapshotCache, cb *testv3.Callbacks) *Server {
	return &Server{
		xDSServer: serverv3.NewServer(context.Background(), cache, cb),
	}
}

func (s *Server) Run(port int) {
	log := log.FromContext(context.Background()).WithValues("xDS Server", port)

	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Error(err, "Can't init xDS GRPC Server")
	}

	s.registerServer(grpcServer)

	log.Info("xDS Server started")
	if err = grpcServer.Serve(lis); err != nil {
		log.Error(err, "Can't start xDS GRPC Server")
	}

}

func (s *Server) registerServer(grpcServer *grpc.Server) {
	// register services
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, s.xDSServer)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, s.xDSServer)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, s.xDSServer)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, s.xDSServer)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, s.xDSServer)
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, s.xDSServer)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, s.xDSServer)
}
