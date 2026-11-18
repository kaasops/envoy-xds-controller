package cache

import (
	"errors"
	"testing"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"
)

func TestLastPublicationAfterRejectedSnapshot(t *testing.T) {
	c := NewSnapshotCache()
	node := "edge"
	snapshot, err := gcp.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.ListenerType: {&listenerv3.Listener{Name: "first"}},
	})
	require.NoError(t, err)
	require.True(t, c.LastPublication(node).IsZero())
	require.NoError(t, c.SetSnapshot(t.Context(), node, snapshot))
	previous := c.LastPublication(node)
	require.False(t, previous.IsZero())
	invalid, err := gcp.NewSnapshot("2", map[resource.Type][]types.Resource{
		resource.ListenerType: {&listenerv3.Listener{Name: "first"}, &listenerv3.Listener{Name: "second"}},
	})
	require.NoError(t, err)
	for range 20 {
		err := c.SetSnapshot(t.Context(), node, invalid)
		require.EqualError(t, err, "snapshot is invalid: 'second' has duplicate address ':0' as existing listener 'first'")
		require.Equal(t, previous, c.LastPublication(node))
	}
	c.ClearSnapshot(node)
	require.True(t, c.LastPublication(node).IsZero())
}

func TestListenerConflictDetailIsScopedToItsNamespace(t *testing.T) {
	c := NewSnapshotCache()
	snap, err := gcp.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.ListenerType: {&listenerv3.Listener{Name: "team/a"}, &listenerv3.Listener{Name: "team/b"}},
	})
	require.NoError(t, err)
	var conflict *DuplicateListenerAddress
	require.True(t, errors.As(c.SetSnapshot(t.Context(), "node", snap), &conflict))
	require.Contains(t, conflict.DetailForNamespace("team"), "team/a")
	require.Contains(t, conflict.DetailForNamespace("team"), "team/b")
	require.Equal(t, "duplicate address", conflict.DetailForNamespace("other"))
	competing := &DuplicateListenerAddress{listener: "team/a", previous: "other/b"}
	require.Equal(t, "duplicate address", competing.DetailForNamespace("team"))
}
