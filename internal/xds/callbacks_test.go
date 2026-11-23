package xds

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	listenerType = "type.googleapis.com/envoy.config.listener.v3.Listener"
	routeType    = "type.googleapis.com/envoy.config.route.v3.RouteConfiguration"

	nackText = "Error adding/updating listener(s) default/https: " +
		"unable to open file '/tmp/nodir/access.log': No such file or directory\n"
)

// logLines collects what the callbacks log. The xDS server calls them from
// its own goroutines, so the sink is guarded.
type logLines struct {
	mu    sync.Mutex
	lines []map[string]any
}

func (l *logLines) add(line string) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		fields = map[string]any{"msg": "unparsable: " + line}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fields)
}

func (l *logLines) withMessage(msg string) []map[string]any {
	l.mu.Lock()
	defer l.mu.Unlock()
	var found []map[string]any
	for _, line := range l.lines {
		if line["msg"] == msg {
			found = append(found, line)
		}
	}
	return found
}

func loggingCallbacks() (*Callbacks, *logLines) {
	lines := &logLines{}
	return NewCallbacks(funcr.NewJSON(lines.add, funcr.Options{}), clients.NewRegistry()), lines
}

func peerContext(t *testing.T) context.Context {
	t.Helper()
	return peer.NewContext(t.Context(), &peer.Peer{Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 7), Port: 41000}})
}

// openDeltaStream opens stream 1 for node "one" the way Envoy 1.40 does: the
// node comes with the first request only.
func openDeltaStream(t *testing.T, cb *Callbacks) {
	t.Helper()
	require.NoError(t, cb.OnDeltaStreamOpen(peerContext(t), 1, ""))
	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("one"),
	}))
}

func deltaResponse(typeURL, nonce, version string, resourceNames ...string) *discovery.DeltaDiscoveryResponse {
	resp := &discovery.DeltaDiscoveryResponse{TypeUrl: typeURL, Nonce: nonce, SystemVersionInfo: version}
	for _, name := range resourceNames {
		resp.Resources = append(resp.Resources, &discovery.Resource{Name: name, Version: "hash-of-" + name})
	}
	return resp
}

func deltaNack(typeURL, nonce, text string) *discovery.DeltaDiscoveryRequest {
	return &discovery.DeltaDiscoveryRequest{
		TypeUrl:       typeURL,
		ResponseNonce: nonce,
		ErrorDetail:   status.New(codes.Internal, text).Proto(),
	}
}

func onlyClient(t *testing.T, cb *Callbacks) clients.Info {
	t.Helper()
	list := cb.connectedClients.List()
	require.Len(t, list, 1)
	return list[0]
}

func onlyStatus(t *testing.T, cb *Callbacks) clients.TypeStatus {
	t.Helper()
	types := onlyClient(t, cb).Types
	require.Len(t, types, 1)
	return types[0]
}

func testCallbacks() *Callbacks {
	return NewCallbacks(logr.Discard(), clients.NewRegistry())
}

func nodeWithVersion(id string) *core.Node {
	return &core.Node{
		Id: id,
		UserAgentVersionType: &core.Node_UserAgentBuildVersion{
			UserAgentBuildVersion: &core.BuildVersion{
				Version: &typev3.SemanticVersion{MajorNumber: 1, MinorNumber: 39, Patch: 0},
			},
		},
	}
}

// Only the first request on a stream is guaranteed to carry the node, and the delta
// server backfills the remembered one only after this callback returns, so a nil
// node here is normal. Envoy 1.38+ omits it when set_node_on_first_message_only is
// enabled, which is when reading req.Node.Id directly began taking the process down.
func TestOnStreamDeltaRequestNilNodeDoesNotPanic(t *testing.T) {
	cb := testCallbacks()

	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("node-1"),
	}))

	require.NotPanics(t, func() {
		require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
			TypeUrl:       listenerType,
			ResponseNonce: "1",
		}))
	})
}

func TestOnStreamDeltaRequestNilNodeKeepsClientInfo(t *testing.T) {
	cb := testCallbacks()

	require.NoError(t, cb.OnDeltaStreamOpen(t.Context(), 1, listenerType))
	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("node-1"),
	}))
	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl:       listenerType,
		ResponseNonce: "1",
	}))

	list := cb.connectedClients.List()
	require.Len(t, list, 1)
	assert.Equal(t, "node-1", list[0].NodeID)
	assert.Equal(t, "v1.39.0", list[0].Version)
}

func TestOnStreamRequestNilNodeDoesNotPanic(t *testing.T) {
	cb := testCallbacks()
	req := &discovery.DiscoveryRequest{TypeUrl: listenerType}

	require.NotPanics(t, func() {
		require.NoError(t, cb.OnStreamRequest(1, req))
		cb.OnStreamResponse(t.Context(), 1, req, &discovery.DiscoveryResponse{})
	})

	assert.Empty(t, cb.connectedClients.List())
}

func TestStreamClosedNilNodeRemovesClient(t *testing.T) {
	for _, tc := range []struct {
		name  string
		open  func(cb *Callbacks, ctx context.Context, id int64) error
		close func(cb *Callbacks, id int64)
	}{
		{
			name: "sotw",
			open: func(cb *Callbacks, ctx context.Context, id int64) error {
				return cb.OnStreamOpen(ctx, id, listenerType)
			},
			close: func(cb *Callbacks, id int64) { cb.OnStreamClosed(id, nil) },
		},
		{
			name: "delta",
			open: func(cb *Callbacks, ctx context.Context, id int64) error {
				return cb.OnDeltaStreamOpen(ctx, id, listenerType)
			},
			close: func(cb *Callbacks, id int64) { cb.OnDeltaStreamClosed(id, nil) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := testCallbacks()
			require.NoError(t, tc.open(cb, t.Context(), 1))
			require.Len(t, cb.connectedClients.List(), 1)

			require.NotPanics(t, func() { tc.close(cb, 1) })
			assert.Empty(t, cb.connectedClients.List())
		})
	}
}

// The peer is only reachable from the stream context, which the request and
// response callbacks never see.
func TestStreamOpenRecordsPeerAddress(t *testing.T) {
	ctx := peer.NewContext(t.Context(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 7), Port: 41000},
	})
	for _, tc := range []struct {
		name string
		open func(cb *Callbacks) error
	}{
		{"sotw", func(cb *Callbacks) error { return cb.OnStreamOpen(ctx, 1, "") }},
		{"delta", func(cb *Callbacks) error { return cb.OnDeltaStreamOpen(ctx, 1, "") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := testCallbacks()
			require.NoError(t, tc.open(cb))

			list := cb.connectedClients.List()
			require.Len(t, list, 1)
			assert.Equal(t, "10.0.0.7:41000", list[0].Address)
		})
	}
}

// The SotW and Delta servers count streams separately, so both hand out id 1.
func TestStreamsOfDifferentProtocolsWithSameIDAreSeparateClients(t *testing.T) {
	cb := testCallbacks()
	require.NoError(t, cb.OnDeltaStreamOpen(t.Context(), 1, ""))
	require.NoError(t, cb.OnStreamOpen(t.Context(), 1, ""))
	require.NoError(t, cb.OnStreamRequest(1, &discovery.DiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("sotw-node"),
	}))
	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("delta-node"),
	}))

	nodeIDs := make([]string, 0, 2)
	for _, c := range cb.connectedClients.List() {
		nodeIDs = append(nodeIDs, c.NodeID)
	}
	assert.ElementsMatch(t, []string{"sotw-node", "delta-node"}, nodeIDs)

	cb.OnStreamClosed(1, nil)

	list := cb.connectedClients.List()
	require.Len(t, list, 1)
	assert.Equal(t, "delta-node", list[0].NodeID)
}

// Envoy 1.40 sends the node once per stream and go-control-plane restores it
// only after the callback, so the NACK itself says nothing about who sent it.
func TestDeltaNackIsLoggedWithTheIdentityOfTheStream(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(listenerType, "1", "7", "default/https"))

	require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(listenerType, "1", nackText)))

	logged := lines.withMessage("Envoy NACKed xDS response")
	require.Len(t, logged, 1)
	assert.Equal(t, map[string]any{
		"logger":          "",
		"msg":             "Envoy NACKed xDS response",
		"error":           strings.TrimSuffix(nackText, "\n"),
		"nodeId":          "one",
		"id":              float64(1),
		"protocol":        "delta",
		"typeUrl":         listenerType,
		"address":         "10.0.0.7:41000",
		"envoyVersion":    "v1.39.0",
		"nonce":           "1",
		"responseVersion": "7",
		"resources":       []any{"default/https"},
		"resourceCount":   float64(1),
	}, logged[0])

	rejections := onlyStatus(t, cb).Rejections
	require.Len(t, rejections, 1)
	want := []clients.ResourceRef{{Name: "default/https", Version: "hash-of-default/https"}}
	assert.Equal(t, want, rejections[0].Resources)
}

// The server answers a SotW NACK with the same snapshot at once; on the test
// bench that was 1400 rounds a second from one Envoy.
func TestSotWNackIsLoggedOncePerRejectedVersion(t *testing.T) {
	cb, lines := loggingCallbacks()
	require.NoError(t, cb.OnStreamOpen(peerContext(t), 1, ""))
	require.NoError(t, cb.OnStreamRequest(1, &discovery.DiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("one"),
	}))

	for _, nonce := range []string{"1", "2", "3"} {
		cb.OnStreamResponse(t.Context(), 1, nil, &discovery.DiscoveryResponse{
			TypeUrl:     listenerType,
			Nonce:       nonce,
			VersionInfo: "7",
		})
		require.NoError(t, cb.OnStreamRequest(1, &discovery.DiscoveryRequest{
			TypeUrl:       listenerType,
			ResponseNonce: nonce,
			VersionInfo:   "6",
			ErrorDetail:   status.New(codes.Internal, nackText).Proto(),
		}))
	}

	logged := lines.withMessage("Envoy NACKed xDS response")
	require.Len(t, logged, 1)
	assert.Equal(t, "sotw", logged[0]["protocol"])
	assert.Equal(t, "10.0.0.7:41000", logged[0]["address"])
	rejections := onlyStatus(t, cb).Rejections
	require.Len(t, rejections, 1)
	assert.Equal(t, 3, rejections[0].Count)
}

// The request handed to a response callback is the one that opened the watch.
// After a NACK that is the NACK itself, error and all.
func TestResponseCallbackIgnoresTheRequestThatOpenedTheWatch(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(listenerType, "1", "7", "default/https"))
	rejecting := deltaNack(listenerType, "1", nackText)
	require.NoError(t, cb.OnStreamDeltaRequest(1, rejecting))

	cb.OnStreamDeltaResponse(1, rejecting, deltaResponse(listenerType, "2", "8", "default/other"))

	assert.Len(t, lines.withMessage("Envoy NACKed xDS response"), 1)
	assert.Empty(t, lines.withMessage("Envoy NACKed an xDS response unknown to this stream"))
	assert.Zero(t, onlyClient(t, cb).UnknownNacks)
	status := onlyStatus(t, cb)
	require.Len(t, status.Rejections, 1)
	assert.Equal(t, "1", status.Rejections[0].Nonce)
	assert.False(t, status.AwaitingAckSince.IsZero())
}

func TestClearedRejectionIsLogged(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(listenerType, "1", "7", "default/https"))
	require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(listenerType, "1", nackText)))

	cb.OnStreamDeltaResponse(1, nil, deltaResponse(listenerType, "2", "8", "default/https"))
	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl:       listenerType,
		ResponseNonce: "2",
	}))

	logged := lines.withMessage("xDS rejection cleared")
	require.Len(t, logged, 1)
	assert.Equal(t, map[string]any{
		"logger":        "",
		"level":         float64(0),
		"msg":           "xDS rejection cleared",
		"nodeId":        "one",
		"id":            float64(1),
		"protocol":      "delta",
		"typeUrl":       listenerType,
		"reason":        "accepted",
		"resources":     []any{"default/https"},
		"resourceCount": float64(1),
	}, logged[0])
	assert.Empty(t, onlyStatus(t, cb).Rejections)
}

func TestUnsubscribeClearsRejection(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(routeType, "1", "3", "r1"))
	require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(routeType, "1", "bad route")))

	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl:                  routeType,
		ResourceNamesUnsubscribe: []string{"r1"},
	}))

	logged := lines.withMessage("xDS rejection cleared")
	require.Len(t, logged, 1)
	assert.Equal(t, "unsubscribed", logged[0]["reason"])
	assert.Empty(t, onlyStatus(t, cb).Rejections)
}

// Anyone who reaches the port can send such requests, as fast as they like
// and with fields megabytes long.
func TestNackOfUnknownResponseIsLoggedOncePerStream(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	longType := "type.googleapis.com/" + strings.Repeat("x", 1<<20)
	longNonce := strings.Repeat("9", 1<<20)

	for range 3 {
		require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(longType, longNonce, "made up")))
	}

	logged := lines.withMessage("Envoy NACKed an xDS response unknown to this stream")
	require.Len(t, logged, 1)
	assert.Equal(t, "one", logged[0]["nodeId"])
	assert.Equal(t, "made up", logged[0]["error"])
	assert.Contains(t, logged[0], "level", "a NACK nobody can act on is not an error")
	assert.Len(t, logged[0]["typeUrl"], 256)
	assert.Len(t, logged[0]["nonce"], 64)
	assert.Empty(t, lines.withMessage("Envoy NACKed xDS response"))
	client := onlyClient(t, cb)
	assert.Equal(t, 3, client.UnknownNacks)
	assert.Empty(t, client.Types)
}

// The Delta cache answers a wildcard request for any type URL, so a client
// could otherwise grow the registry one made-up type at a time.
func TestResponsesOfUnknownTypesAreNotTracked(t *testing.T) {
	const madeUpType = "type.googleapis.com/made.up.Type"
	cb, _ := loggingCallbacks()
	openDeltaStream(t, cb)
	require.NoError(t, cb.OnStreamOpen(peerContext(t), 1, ""))

	cb.OnStreamDeltaResponse(1, nil, deltaResponse(madeUpType, "1", "7"))
	cb.OnStreamResponse(t.Context(), 1, nil, &discovery.DiscoveryResponse{TypeUrl: madeUpType, Nonce: "1"})

	list := cb.connectedClients.List()
	require.Len(t, list, 2)
	for _, client := range list {
		assert.Empty(t, client.Types, client.Protocol)
	}
}

// The version Envoy last accepted reaches the registry only through this
// callback; without it no SotW response would ever count as acknowledged.
func TestSotWAckThroughCallbacks(t *testing.T) {
	cb, lines := loggingCallbacks()
	require.NoError(t, cb.OnStreamOpen(peerContext(t), 1, ""))
	require.NoError(t, cb.OnStreamRequest(1, &discovery.DiscoveryRequest{
		TypeUrl: listenerType,
		Node:    nodeWithVersion("one"),
	}))
	cb.OnStreamResponse(t.Context(), 1, nil, &discovery.DiscoveryResponse{
		TypeUrl: listenerType, Nonce: "1", VersionInfo: "7",
	})
	require.NoError(t, cb.OnStreamRequest(1, &discovery.DiscoveryRequest{
		TypeUrl: listenerType, ResponseNonce: "1", VersionInfo: "6",
		ErrorDetail: status.New(codes.Internal, nackText).Proto(),
	}))
	cb.OnStreamResponse(t.Context(), 1, nil, &discovery.DiscoveryResponse{
		TypeUrl: listenerType, Nonce: "2", VersionInfo: "8",
	})
	require.NoError(t, cb.OnStreamRequest(1, &discovery.DiscoveryRequest{
		TypeUrl: listenerType, ResponseNonce: "2", VersionInfo: "8",
	}))

	acked := onlyStatus(t, cb)
	assert.Equal(t, "8", acked.AckedVersion)
	assert.Empty(t, acked.Rejections)
	logged := lines.withMessage("Envoy NACKed xDS response")
	require.Len(t, logged, 1)
	assert.Equal(t, "7", logged[0]["responseVersion"])
	assert.Len(t, lines.withMessage("xDS rejection cleared"), 1)
}

func TestRemovalClearsRejectionThroughCallbacks(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(listenerType, "1", "7", "default/https"))
	require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(listenerType, "1", nackText)))

	cb.OnStreamDeltaResponse(1, nil, &discovery.DeltaDiscoveryResponse{
		TypeUrl: listenerType, Nonce: "2", SystemVersionInfo: "8", RemovedResources: []string{"default/https"},
	})
	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl:       listenerType,
		ResponseNonce: "2",
	}))

	logged := lines.withMessage("xDS rejection cleared")
	require.Len(t, logged, 1)
	assert.Equal(t, "removed", logged[0]["reason"])
	assert.Empty(t, onlyStatus(t, cb).Rejections)
}

// An unsubscribe list is as long as the client makes it.
func TestClearedLogNamesFewResourcesAndCountsAll(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	routes := make([]string, 0, 25)
	for i := range 25 {
		routes = append(routes, fmt.Sprintf("route-%02d", i))
	}
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(routeType, "1", "3", routes...))
	require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(routeType, "1", "bad route")))

	require.NoError(t, cb.OnStreamDeltaRequest(1, &discovery.DeltaDiscoveryRequest{
		TypeUrl:                  routeType,
		ResourceNamesUnsubscribe: routes,
	}))

	logged := lines.withMessage("xDS rejection cleared")
	require.Len(t, logged, 1)
	assert.Len(t, logged[0]["resources"], 20)
	assert.InDelta(t, 25, logged[0]["resourceCount"], 0)
}

// After a restart the first response carries every resource of the type.
func TestNackLogNamesFewResourcesAndCountsAll(t *testing.T) {
	cb, lines := loggingCallbacks()
	openDeltaStream(t, cb)
	routes := make([]string, 0, 25)
	for i := range 25 {
		routes = append(routes, fmt.Sprintf("route-%02d", i))
	}
	cb.OnStreamDeltaResponse(1, nil, deltaResponse(routeType, "1", "3", routes...))

	require.NoError(t, cb.OnStreamDeltaRequest(1, deltaNack(routeType, "1", "bad route")))

	logged := lines.withMessage("Envoy NACKed xDS response")
	require.Len(t, logged, 1)
	assert.Len(t, logged[0]["resources"], 20)
	assert.InDelta(t, 25, logged[0]["resourceCount"], 0)
}

// Behind a balancer the address is the balancer's. The open line is where an
// operator finds out which of the two the controller sees.
func TestStreamOpenLogsThePeerAddress(t *testing.T) {
	cb, lines := loggingCallbacks()

	require.NoError(t, cb.OnStreamOpen(peerContext(t), 1, ""))
	require.NoError(t, cb.OnDeltaStreamOpen(peerContext(t), 1, ""))

	for _, msg := range []string{"stream open", "delta stream open"} {
		logged := lines.withMessage(msg)
		require.Len(t, logged, 1, msg)
		assert.Equal(t, "10.0.0.7:41000", logged[0]["address"], msg)
	}
}

// Everything above calls the callbacks by hand. This drives them through the
// real server, which decides the order of the calls and what each one carries:
// the response callback runs before the client can answer, and the NACK
// reaches the request callback without the node.
func TestNackOverARealDeltaStream(t *testing.T) {
	snapshot, err := cache.NewSnapshot("7", map[resource.Type][]types.Resource{
		resource.ListenerType: {&listenerv3.Listener{Name: "default/https"}},
		resource.ClusterType:  {&clusterv3.Cluster{Name: "backend"}},
	})
	require.NoError(t, err)
	snapshots := cache.NewSnapshotCache(true, cache.IDHash{}, nil)
	require.NoError(t, snapshots.SetSnapshot(t.Context(), "one", snapshot))

	cb, lines := loggingCallbacks()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := newGRPCServer(server.NewServer(t.Context(), snapshots, cb), logr.Discard())
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	stream, err := discovery.NewAggregatedDiscoveryServiceClient(conn).DeltaAggregatedResources(ctx)
	require.NoError(t, err)

	require.NoError(t, stream.Send(&discovery.DeltaDiscoveryRequest{
		TypeUrl: resource.ListenerType,
		Node:    nodeWithVersion("one"),
	}))
	listeners, err := stream.Recv()
	require.NoError(t, err)
	require.NoError(t, stream.Send(deltaNack(resource.ListenerType, listeners.GetNonce(), nackText)))
	// The server reads a stream in order and answers nothing to a NACK, so the
	// reply to this next request proves the NACK has been handled.
	require.NoError(t, stream.Send(&discovery.DeltaDiscoveryRequest{TypeUrl: resource.ClusterType}))
	_, err = stream.Recv()
	require.NoError(t, err)

	client := onlyClient(t, cb)
	assert.Equal(t, "one", client.NodeID)
	assert.Equal(t, "v1.39.0", client.Version)
	assert.True(t, strings.HasPrefix(client.Address, "127.0.0.1:"), client.Address)
	require.Len(t, client.Types, 2)
	clusters, rejectedListeners := client.Types[0], client.Types[1]
	assert.Equal(t, resource.ClusterType, clusters.TypeURL)
	assert.False(t, clusters.AwaitingAckSince.IsZero())
	assert.Equal(t, resource.ListenerType, rejectedListeners.TypeURL)
	assert.True(t, rejectedListeners.AwaitingAckSince.IsZero())
	require.Len(t, rejectedListeners.Rejections, 1)
	rejection := rejectedListeners.Rejections[0]
	assert.Equal(t, listeners.GetNonce(), rejection.Nonce)
	assert.Equal(t, "7", rejection.ResponseVersion)
	require.Len(t, rejection.Resources, 1)
	assert.Equal(t, "default/https", rejection.Resources[0].Name)

	logged := lines.withMessage("Envoy NACKed xDS response")
	require.Len(t, logged, 1)
	assert.Equal(t, "one", logged[0]["nodeId"])
	assert.Equal(t, client.Address, logged[0]["address"])
}
