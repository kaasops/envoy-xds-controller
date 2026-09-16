package xds

import (
	"context"
	"net"
	"testing"
	"time"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func startPanickingXDSServer(t *testing.T) *grpc.ClientConn {
	t.Helper()

	callbacks := server.CallbackFuncs{
		StreamDeltaRequestFunc: func(int64, *discovery.DeltaDiscoveryRequest) error { panic("boom") },
		FetchRequestFunc:       func(context.Context, *discovery.DiscoveryRequest) error { panic("boom") },
	}
	srv := server.NewServer(t.Context(), cache.NewSnapshotCache(false, cache.IDHash{}, nil), callbacks)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := newGRPCServer(srv, logr.Discard())
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// The second call proves the server survived the first panic.
func TestServerRecoversFromPanicInDeltaStream(t *testing.T) {
	client := discovery.NewAggregatedDiscoveryServiceClient(startPanickingXDSServer(t))

	for range 2 {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		stream, err := client.DeltaAggregatedResources(ctx)
		require.NoError(t, err)
		require.NoError(t, stream.Send(&discovery.DeltaDiscoveryRequest{
			TypeUrl: "type.googleapis.com/envoy.config.listener.v3.Listener",
		}))
		_, err = stream.Recv()
		assert.Equal(t, codes.Internal, status.Code(err), err)
		cancel()
	}
}

func TestServerRecoversFromPanicInFetch(t *testing.T) {
	client := clusterservice.NewClusterDiscoveryServiceClient(startPanickingXDSServer(t))

	for range 2 {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		_, err := client.FetchClusters(ctx, &discovery.DiscoveryRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), err)
		cancel()
	}
}
