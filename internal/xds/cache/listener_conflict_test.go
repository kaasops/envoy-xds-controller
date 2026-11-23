package cache

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"
)

func listenerAt(name, host string, port uint32) *listenerv3.Listener {
	return &listenerv3.Listener{
		Name: name,
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{
			Address:       host,
			PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: port},
		}}},
	}
}

func TestDuplicateListenerAddressNamesTheNode(t *testing.T) {
	c := NewSnapshotCache()
	snapshot, err := gcp.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.ListenerType: {listenerAt("ns/https", "0.0.0.0", 443), listenerAt("ns/https-module", "0.0.0.0", 443)},
	})
	require.NoError(t, err)

	err = c.SetSnapshot(t.Context(), "node-a", snapshot)

	var conflict *DuplicateListenerAddress
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "node-a", conflict.NodeID())
	require.Equal(t, "0.0.0.0:443", conflict.Address())
	listener, previous := conflict.Listeners()
	require.Equal(t, "ns/https-module", listener)
	require.Equal(t, "ns/https", previous)
	require.EqualError(t, err,
		"snapshot is invalid: 'ns/https-module' has duplicate address '0.0.0.0:443' as existing listener 'ns/https'")
}
