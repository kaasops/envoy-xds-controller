package updater

import (
	"errors"
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
)

func TestDryBuildNodeShowsWhatAFrozenNodeWaitsFor(t *testing.T) {
	updater, moved := moveFixture(t)
	require.NoError(t, updater.RebuildSnapshots(t.Context()))
	served, ok := updater.snapshotCache.Served("node-a")
	require.True(t, ok)
	require.True(t, updater.snapshotCache.Freeze("node-a", wrapped.Freeze{By: "alice", Reason: "check"}))

	updater.store.SetListener(makeListenerCR("http", "0.0.0.0", 80))
	updater.store.SetVirtualService(makeVSWithListener("added", []string{"node-a"}, "http"))
	require.NoError(t, updater.RebuildSnapshots(t.Context()))

	built, err := updater.DryBuildNode(t.Context(), "node-a")
	require.NoError(t, err)
	require.NotEqual(t, served.ID(), wrapped.SnapshotID(built), "built has the added listener")
	still, _ := updater.snapshotCache.Served("node-a")
	require.Equal(t, served.ID(), still.ID(), "the frozen node is served what it had")

	updater.store.SetVirtualService(moved)
	built, err = updater.DryBuildNode(t.Context(), "node-a")
	require.ErrorContains(t, err, "duplicate address", "the conflict of this node")
	require.Len(t, built.GetResources(resource.ListenerType), 3, "the snapshot that was refused")
	_, err = updater.DryBuildNode(t.Context(), "node-b")
	require.NoError(t, err, "a conflict on another node does not concern node-b")

	empty, err := updater.DryBuildNode(t.Context(), "node-gone")
	require.NoError(t, err)
	require.Empty(t, empty.GetResources(resource.ListenerType))
}

func TestDryBuildNodeReportsBuildFailures(t *testing.T) {
	updater, _ := moveFixture(t)
	t.Cleanup(withStubbedBuilder(t, func(*v1alpha1.VirtualService, store.Store) (*resbuilder.Resources, error) {
		return nil, errBroken
	}))

	built, err := updater.DryBuildNode(t.Context(), "node-a")
	require.ErrorIs(t, err, errBroken)
	require.Nil(t, built)
}

var errBroken = errors.New("missing dependency")

func TestDryBuildNodeReportsTheConflictOfEachHeldNode(t *testing.T) {
	updater, moved := moveFixture(t)
	updater.store.SetVirtualService(moved)
	updater.store.SetVirtualService(makeVSWithListener("third", []string{"node-c"}, "https"))
	updater.store.SetVirtualService(makeVSWithListener("fourth", []string{"node-c"}, "https-module"))

	for _, node := range []string{"node-a", "node-c"} {
		_, err := updater.DryBuildNode(t.Context(), node)
		var conflict *wrapped.DuplicateListenerAddress
		require.ErrorAs(t, err, &conflict, node)
		require.Equal(t, node, conflict.NodeID())
	}
	_, err := updater.DryBuildNode(t.Context(), "node-b")
	require.NoError(t, err)
}

func TestFrozenNodesAreListedForAdmission(t *testing.T) {
	updater, _ := moveFixture(t)
	require.Empty(t, updater.FrozenNodeIDs())
	updater.snapshotCache.RestoreFrozen(map[string]wrapped.Freeze{"node-b": {}, "node-a": {}})

	require.Equal(t, []string{"node-a", "node-b"}, updater.FrozenNodeIDs())
}

func TestStatusOfAFrozenNodeSaysSo(t *testing.T) {
	updater, _ := moveFixture(t)
	require.NoError(t, updater.RebuildSnapshots(t.Context()))
	at := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	updater.snapshotCache.Freeze("node-a", wrapped.Freeze{By: "alice@example.com", At: at, Reason: "listener switch"})

	require.NoError(t, updater.RebuildSnapshots(t.Context()))

	frozen, _ := updater.PublicationStatusFor(makeVSWithListener("first", []string{"node-a"}, "https"))
	require.False(t, frozen.Invalid)
	require.Contains(t, frozen.Message, "node node-a is frozen since 2026-11-23T09:00:00Z (listener switch)")
	require.NotContains(t, frozen.Message, "alice", "who froze it is in the API, not in every status")
	neighbour, _ := updater.PublicationStatusFor(makeVSWithListener("neighbour", []string{"node-b"}, "https"))
	require.Empty(t, neighbour.Message)

	updater.snapshotCache.Unfreeze("node-a")
	require.NoError(t, updater.RebuildSnapshots(t.Context()))
	cleared, _ := updater.PublicationStatusFor(makeVSWithListener("first", []string{"node-a"}, "https"))
	require.Empty(t, cleared.Message)
}
