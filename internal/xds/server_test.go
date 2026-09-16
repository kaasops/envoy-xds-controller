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
	ctrmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
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
	method := discovery.AggregatedDiscoveryService_DeltaAggregatedResources_FullMethodName
	before := recoveredPanicsFor(t, method)

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

	assert.InDelta(t, before+2, recoveredPanicsFor(t, method), 0)
}

func TestServerRecoversFromPanicInFetch(t *testing.T) {
	client := clusterservice.NewClusterDiscoveryServiceClient(startPanickingXDSServer(t))
	method := clusterservice.ClusterDiscoveryService_FetchClusters_FullMethodName
	before := recoveredPanicsFor(t, method)

	for range 2 {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		_, err := client.FetchClusters(ctx, &discovery.DiscoveryRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), err)
		cancel()
	}

	assert.InDelta(t, before+2, recoveredPanicsFor(t, method), 0)
}

// Reads the counter from the registry the metrics endpoint serves, so the test
// also fails if the metric is registered elsewhere or under another name. The
// series must exist before the first panic: increase() misses one that starts at 1.
func recoveredPanicsFor(t *testing.T, method string) float64 {
	t.Helper()
	families, err := ctrmetrics.Registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "exc_xds_recovered_panics_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "method" && label.GetValue() == method {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	require.Failf(t, "series not exported", "exc_xds_recovered_panics_total{method=%q}", method)
	return 0
}
