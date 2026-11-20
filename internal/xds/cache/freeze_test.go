package cache

import (
	"strconv"
	"testing"
	"time"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"
)

// snapshotWith returns a snapshot of one listener on 0.0.0.0:<port>; the port makes the content differ.
func snapshotWith(t *testing.T, version string, port uint32) gcp.ResourceSnapshot {
	t.Helper()
	snapshot, err := gcp.NewSnapshot(version, map[resource.Type][]types.Resource{
		resource.ListenerType: {listenerAt("ns/https", "0.0.0.0", port)},
	})
	require.NoError(t, err)
	return snapshot
}

func listenerPort(t *testing.T, c *SnapshotCache, nodeID string) uint32 {
	t.Helper()
	served, err := c.GetSnapshot(nodeID)
	require.NoError(t, err)
	l := served.GetResources(resource.ListenerType)["ns/https"].(*listenerv3.Listener)
	return l.GetAddress().GetSocketAddress().GetPortValue()
}

var freeze = Freeze{By: "alice", Reason: "listener switch", At: time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)}

func TestFrozenNodeKeepsServedSnapshot(t *testing.T) {
	c := NewSnapshotCache()
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "1", 443)))
	published := c.LastPublication("node-a")
	require.True(t, c.Freeze("node-a", freeze))
	require.False(t, c.Freeze("node-a", freeze), "already frozen")

	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "2", 8443)))

	require.Equal(t, uint32(443), listenerPort(t, c, "node-a"))
	require.Equal(t, published, c.LastPublication("node-a"))
	require.Empty(t, c.History("node-a"))
	offered, ok := c.Offered("node-a")
	require.True(t, ok)
	require.Equal(t, "2", offered.GetVersion(resource.ListenerType))
	got, frozen := c.Frozen("node-a")
	require.True(t, frozen)
	require.Equal(t, freeze, got)

	require.True(t, c.Unfreeze("node-a"))
	require.False(t, c.Unfreeze("node-a"), "not frozen")
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "2", 8443)))
	require.Equal(t, uint32(8443), listenerPort(t, c, "node-a"))
}

func TestHistoryKeepsPreviouslyServedSnapshots(t *testing.T) {
	c := NewSnapshotCache()
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "0", 999)))
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "0", 998)))
	require.Empty(t, c.History("node-a"), "no history by default")
	c.SetHistoryLimit(maxHistory + 5)
	for i := 1; i <= maxHistory+3; i++ {
		require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, strconv.Itoa(i), uint32(1000+i))))
	}
	history := c.History("node-a")
	require.Len(t, history, maxHistory)
	require.Equal(t, strconv.Itoa(maxHistory+2), history[0].Snapshot.GetVersion(resource.ListenerType), "newest first")
	served, ok := c.Served("node-a")
	require.True(t, ok)
	require.Equal(t, strconv.Itoa(maxHistory+3), served.Snapshot.GetVersion(resource.ListenerType))
	require.False(t, served.PublishedAt.IsZero())
}

func TestSnapshotIDFollowsContentNotVersions(t *testing.T) {
	require.Equal(t, SnapshotID(snapshotWith(t, "1", 443)), SnapshotID(snapshotWith(t, "7", 443)))
	require.NotEqual(t, SnapshotID(snapshotWith(t, "1", 443)), SnapshotID(snapshotWith(t, "1", 8443)))
	require.Len(t, SnapshotID(snapshotWith(t, "1", 443)), 12)
}

func TestRollbackServesOldContentUnderNewVersions(t *testing.T) {
	c := NewSnapshotCache()
	c.SetHistoryLimit(maxHistory)
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "4", 443)))
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "5", 8443)))
	target := c.History("node-a")[0]

	require.NoError(t, c.Rollback(t.Context(), "node-a", target.ID(), "", freeze, c.NextVersion("node-a")))

	served, ok := c.Served("node-a")
	require.True(t, ok)
	require.Equal(t, target.ID(), served.ID(), "content of the old snapshot")
	require.Equal(t, uint32(443), listenerPort(t, c, "node-a"))
	require.Greater(t, versionOf(t, served.Snapshot), int64(5), "a version Envoy has not seen")
	_, frozen := c.Frozen("node-a")
	require.True(t, frozen, "the next build must not overwrite the rollback")
	require.Equal(t, uint32(8443),
		c.History("node-a")[0].Snapshot.GetResources(resource.ListenerType)["ns/https"].(*listenerv3.Listener).
			GetAddress().GetSocketAddress().GetPortValue())

	require.ErrorIs(t, c.Rollback(t.Context(), "node-a", "000000000000", "", freeze, c.NextVersion("node-a")),
		ErrUnknownSnapshot)
	require.ErrorIs(t, c.Rollback(t.Context(), "node-a", target.ID(), "000000000000", freeze, c.NextVersion("node-a")),
		ErrServedChanged,
		"the check and the change are one step")
}

func TestRestoredFreezeServesNothing(t *testing.T) {
	c := NewSnapshotCache()
	c.RestoreFrozen(map[string]Freeze{"node-a": freeze})

	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "1", 443)))
	require.NoError(t, c.SetSnapshot(t.Context(), "node-b", snapshotWith(t, "1", 443)))

	_, err := c.GetSnapshot("node-a")
	require.Error(t, err, "running Envoys keep what they have, new ones wait for a decision")
	_, served := c.Served("node-a")
	require.False(t, served)
	require.Equal(t, map[string]Freeze{"node-a": freeze}, c.FrozenNodes())
	_, err = c.GetSnapshot("node-b")
	require.NoError(t, err)
}

func TestPublishFrozenServesTheGivenSnapshotAndUnfreezes(t *testing.T) {
	c := NewSnapshotCache()
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "7", 443)))
	require.True(t, c.Freeze("node-a", freeze))
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "8", 9443)))
	reviewed := snapshotWith(t, "1", 8443)

	require.NoError(t, c.PublishFrozen(t.Context(), "node-a", reviewed, c.NextVersion("node-a")))

	served, _ := c.Served("node-a")
	require.Equal(t, SnapshotID(reviewed), served.ID())
	require.Greater(t, versionOf(t, served.Snapshot), int64(8), "above what was served and offered")
	_, frozen := c.Frozen("node-a")
	require.False(t, frozen)
}

func versionOf(t *testing.T, s gcp.ResourceSnapshot) int64 {
	t.Helper()
	v, err := strconv.ParseInt(s.GetVersion(resource.ListenerType), 10, 64)
	require.NoError(t, err)
	return v
}

// A freeze or rollback publication must never reuse a version an Envoy may hold, also right after a
// restart: builds go on counting from what a freeze served, and only the freeze version is kept.
func TestControlVersionsStayAboveBuildsAcrossRestarts(t *testing.T) {
	clock := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	first := NewSnapshotCache()
	first.now = func() time.Time { return clock }
	first.RestoreFrozen(map[string]Freeze{"node-a": freeze})
	require.NoError(t, first.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "1", 443)))
	v1 := first.NextVersion("node-a")
	require.NoError(t, first.PublishFrozen(t.Context(), "node-a", snapshotWith(t, "1", 8443), v1))
	require.Equal(t, v1, first.VersionFloor(), "what the caller keeps")
	for i := int64(1); i <= 1000; i++ {
		require.NoError(t, first.SetSnapshot(t.Context(), "node-a",
			snapshotWith(t, strconv.FormatInt(v1+i, 10), uint32(9000+i))), "builds count on from the freeze version")
	}

	restarted := NewSnapshotCache()
	restarted.now = func() time.Time { return clock.Add(time.Second) }
	restarted.SetVersionFloor(first.VersionFloor())
	restarted.RestoreFrozen(map[string]Freeze{"node-a": freeze})
	require.NoError(t, restarted.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "1", 443)))
	v2 := restarted.NextVersion("node-a")

	require.Greater(t, v2, v1+1000, "a restart takes far longer than the builds can count in nanoseconds")
}

func TestRepeatedSameContentKeepsTheHistory(t *testing.T) {
	c := NewSnapshotCache()
	c.SetHistoryLimit(3)
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "1", 443)))
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "2", 8443)))
	for range 5 {
		require.NoError(t, c.SetSnapshot(t.Context(), "node-a", &gcp.Snapshot{}), "the node lost its last service")
	}

	published := c.LastPublication("node-a")
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", &gcp.Snapshot{}))
	require.Equal(t, published, c.LastPublication("node-a"), "the same snapshot again is not written")
	history := c.History("node-a")
	require.Len(t, history, 2, "an empty snapshot again is not a new entry")
	require.Equal(t, SnapshotID(snapshotWith(t, "2", 8443)), history[0].ID())
}

func TestUnfreezeOfAnEmptyNodeWritesOnce(t *testing.T) {
	c := NewSnapshotCache()
	c.SetHistoryLimit(3)
	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", snapshotWith(t, "1", 443)))
	require.True(t, c.Freeze("node-a", freeze))
	require.NoError(t, c.PublishFrozen(t.Context(), "node-a", &gcp.Snapshot{}, c.NextVersion("node-a")))
	published := c.LastPublication("node-a")

	require.NoError(t, c.SetSnapshot(t.Context(), "node-a", &gcp.Snapshot{}), "the next build of the node")

	require.Equal(t, published, c.LastPublication("node-a"), "same content, only the versions differ")
}

// The guarantee does not lean on the clock for a frozen node: the freeze raises the floor to what builds
// served it, and nothing is served to it until the next freeze or rollback publication.
func TestFreezeRaisesTheFloorToWhatTheNodeWasServed(t *testing.T) {
	clock := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	first := NewSnapshotCache()
	first.now = func() time.Time { return clock }
	served := clock.UnixNano() + 1000
	require.NoError(t, first.SetSnapshot(t.Context(), "node-a", snapshotWith(t, strconv.FormatInt(served, 10), 443)))
	require.True(t, first.Freeze("node-a", freeze))

	require.Equal(t, served, first.RaiseVersionFloor("node-a"))

	restarted := NewSnapshotCache()
	restarted.now = func() time.Time { return clock }
	restarted.SetVersionFloor(first.VersionFloor())
	restarted.RestoreFrozen(map[string]Freeze{"node-a": freeze})
	require.Greater(t, restarted.NextVersion("node-a"), served, "the clock stood still")
}
