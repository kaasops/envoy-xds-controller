package xds

import (
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/go-logr/logr"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const listenerType = "type.googleapis.com/envoy.config.listener.v3.Listener"

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
		close func(cb *Callbacks, id int64)
	}{
		{"sotw", func(cb *Callbacks, id int64) { cb.OnStreamClosed(id, nil) }},
		{"delta", func(cb *Callbacks, id int64) { cb.OnDeltaStreamClosed(id, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := testCallbacks()
			require.NoError(t, cb.OnDeltaStreamOpen(t.Context(), 1, listenerType))
			require.Len(t, cb.connectedClients.List(), 1)

			require.NotPanics(t, func() { tc.close(cb, 1) })
			assert.Empty(t, cb.connectedClients.List())
		})
	}
}
