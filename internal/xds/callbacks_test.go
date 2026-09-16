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

// Only the first request on a stream carries the node identifier, and the delta
// server backfills the remembered one only after this callback returns, so a nil
// node here is normal. Envoy started actually omitting it in 1.39, which is when
// reading req.Node.Id directly began taking the whole process down.
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

func TestStreamClosedNilNodeDoesNotPanic(t *testing.T) {
	cb := testCallbacks()

	require.NoError(t, cb.OnDeltaStreamOpen(t.Context(), 1, listenerType))
	require.NotPanics(t, func() {
		cb.OnStreamClosed(1, nil)
		cb.OnDeltaStreamClosed(1, nil)
	})

	assert.Empty(t, cb.connectedClients.List())
}
