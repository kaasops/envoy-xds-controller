package snapshots

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"

	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

func listener(name string) *listenerv3.Listener {
	return &listenerv3.Listener{
		Name: name,
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{
			Address:       "0.0.0.0",
			PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 443},
		}}},
	}
}

func secret(name, key string) *tlsv3.Secret {
	return &tlsv3.Secret{Name: name, Type: &tlsv3.Secret_TlsCertificate{TlsCertificate: &tlsv3.TlsCertificate{
		PrivateKey: &corev3.DataSource{Specifier: &corev3.DataSource_InlineString{InlineString: key}},
	}}}
}

func snapshot(t *testing.T, version string, resources ...types.Resource) gcp.ResourceSnapshot {
	t.Helper()
	byType := map[resource.Type][]types.Resource{}
	for _, r := range resources {
		switch r.(type) {
		case *listenerv3.Listener:
			byType[resource.ListenerType] = append(byType[resource.ListenerType], r)
		case *tlsv3.Secret:
			byType[resource.SecretType] = append(byType[resource.SecretType], r)
		}
	}
	s, err := gcp.NewSnapshot(version, byType)
	require.NoError(t, err)
	return s
}

// fakeBuilder returns a fixed build result.
type fakeBuilder struct {
	built    gcp.ResourceSnapshot
	err      error
	rebuilds int
}

func (b *fakeBuilder) RebuildSnapshots(context.Context) error {
	b.rebuilds++
	return nil
}

func (b *fakeBuilder) DryBuildNode(context.Context, string) (gcp.ResourceSnapshot, error) {
	return b.built, b.err
}

// memoryStore records what would be kept in Kubernetes.
type memoryStore struct {
	mu    sync.Mutex
	saved map[string]wrapped.Freeze
	floor int64
	err   error
}

func (s *memoryStore) Save(_ context.Context, frozen map[string]wrapped.Freeze, floor int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.saved, s.floor = frozen, floor
	return nil
}

func (s *memoryStore) kept() map[string]wrapped.Freeze {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saved
}

var now = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

// fixture serves node-a two snapshots in a row and builds a third one.
func fixture(t *testing.T) (*Control, *wrapped.SnapshotCache, *fakeBuilder, *memoryStore) {
	t.Helper()
	cache := wrapped.NewSnapshotCache()
	cache.SetHistoryLimit(10)
	require.NoError(t, cache.SetSnapshot(t.Context(), "node-a", snapshot(t, "1", listener("ns/https"))))
	require.NoError(t, cache.SetSnapshot(t.Context(), "node-a",
		snapshot(t, "2", listener("ns/https"), secret("ns/cert", "old-key"))))
	builder := &fakeBuilder{built: snapshot(t, "1", listener("ns/https-next"), secret("ns/cert", "new-key"))}
	store := &memoryStore{}
	control := &Control{Cache: cache, Builder: builder, Store: store, Now: func() time.Time { return now }}
	return control, cache, builder, store
}

func TestStateShowsServedBuiltAndHistory(t *testing.T) {
	control, cache, _, _ := fixture(t)

	state, err := control.State(t.Context(), "node-a")

	require.NoError(t, err)
	served, _ := cache.Served("node-a")
	require.Equal(t, served.ID(), state.Served.ID)
	require.Equal(t, "2", state.Served.Versions[resource.ListenerType])
	require.Equal(t, 1, state.Served.Counts[resource.SecretType])
	require.NotEqual(t, state.Served.ID, state.Built.ID)
	require.Empty(t, state.Built.Error)
	require.Len(t, state.History, 1)
	require.False(t, state.Frozen)
	require.Nil(t, state.Freeze)

	_, err = control.State(t.Context(), "node-x")
	require.ErrorIs(t, err, ErrUnknownNode)
}

func TestFreezeIsKeptAndUnfreezePublishes(t *testing.T) {
	control, _, builder, store := fixture(t)

	state, err := control.Freeze(t.Context(), "node-a", "alice", "listener switch")
	require.NoError(t, err)
	require.True(t, state.Frozen)
	require.Equal(t, wrapped.Freeze{By: "alice", At: now, Reason: "listener switch"}, *state.Freeze)
	require.Contains(t, store.saved, "node-a")
	require.Equal(t, "2", state.Served.Versions[resource.ListenerType])
	require.Equal(t, int64(2), store.floor, "the freeze keeps the version the node was served")
	require.Equal(t, 1, builder.rebuilds, "the statuses of the node's services tell it is frozen")

	_, err = control.Freeze(t.Context(), "node-a", "bob", "again")
	require.ErrorIs(t, err, ErrAlreadyFrozen)
	_, err = control.Freeze(t.Context(), "node-a", "bob", "")
	require.Error(t, err)

	state, err = control.Unfreeze(t.Context(), "node-a", "alice", "")
	require.NoError(t, err)
	require.False(t, state.Frozen)
	require.NotContains(t, store.saved, "node-a")
	require.Equal(t, wrapped.SnapshotID(builder.built), state.Served.ID, "what was compared is served first")
	require.Equal(t, state.Served.Versions[resource.ListenerType], strconv.FormatInt(store.floor, 10),
		"the version is kept before it is served")
	require.Equal(t, 2, builder.rebuilds, "then whatever the objects build now")

	_, err = control.Unfreeze(t.Context(), "node-a", "alice", "")
	require.ErrorIs(t, err, ErrNotFrozen)
}

func TestFreezeIsUndoneWhenItCannotBeKept(t *testing.T) {
	control, cache, _, store := fixture(t)
	store.err = errors.New("forbidden")

	_, err := control.Freeze(t.Context(), "node-a", "alice", "listener switch")

	require.ErrorContains(t, err, "forbidden")
	_, frozen := cache.Frozen("node-a")
	require.False(t, frozen, "a freeze lost on restart would publish what it held back")
}

func TestRollbackServesHistoryAndFreezes(t *testing.T) {
	control, cache, builder, store := fixture(t)
	target := cache.History("node-a")[0].ID()

	state, err := control.Rollback(t.Context(), "node-a", "alice", target, "")

	require.NoError(t, err)
	require.Equal(t, target, state.Served.ID)
	require.True(t, state.Frozen)
	require.Equal(t, "rollback to "+target, state.Freeze.Reason)
	require.Contains(t, store.saved, "node-a")
	require.Equal(t, 1, builder.rebuilds, "the statuses of the node's services tell it is frozen at once")

	_, err = control.Rollback(t.Context(), "node-a", "alice", "000000000000", "")
	require.ErrorIs(t, err, wrapped.ErrUnknownSnapshot)
}

func TestDiffServedAgainstBuilt(t *testing.T) {
	control, _, _, _ := fixture(t)

	diff, err := control.Diff(t.Context(), "node-a", "served", "built", false)

	require.NoError(t, err)
	changes := map[string]string{}
	for _, r := range diff.Resources {
		changes[r.Name] = r.Change
	}
	require.Equal(t, map[string]string{"ns/https": "removed", "ns/https-next": "added", "ns/cert": "changed"}, changes)
	require.Equal(t, Summary{Added: 1, Removed: 1}, diff.Summary[resource.ListenerType])

	all, err := control.Diff(t.Context(), "node-a", "served", "served", true)
	require.NoError(t, err)
	for _, r := range all.Resources {
		require.Equal(t, "same", r.Change, r.Name)
	}
}

func TestResourceNeverShowsSecrets(t *testing.T) {
	control, _, _, _ := fixture(t)

	l, err := control.Resource(t.Context(), "node-a", "served", resource.ListenerType, "ns/https")
	require.NoError(t, err)
	require.Contains(t, string(l.JSON), `"portValue":443`)
	require.Len(t, l.SHA256, 64)

	s, err := control.Resource(t.Context(), "node-a", "served", resource.SecretType, "ns/cert")
	require.NoError(t, err)
	require.Nil(t, s.JSON)
	require.Len(t, s.SHA256, 64)

	_, err = control.Resource(t.Context(), "node-a", "served", resource.ListenerType, "ns/none")
	require.ErrorIs(t, err, ErrUnknownResource)
}

func TestWritesCheckWhatTheOperatorSaw(t *testing.T) {
	control, cache, builder, _ := fixture(t)
	_, err := control.Freeze(t.Context(), "node-a", "alice", "listener switch")
	require.NoError(t, err)
	seen := wrapped.SnapshotID(builder.built)
	builder.built = snapshot(t, "1", listener("ns/https-other"))

	_, err = control.Unfreeze(t.Context(), "node-a", "alice", seen)
	require.ErrorIs(t, err, ErrStateChanged, "the build changed after the diff was read")
	_, frozen := cache.Frozen("node-a")
	require.True(t, frozen)

	target := cache.History("node-a")[0].ID()
	_, err = control.Rollback(t.Context(), "node-a", "alice", target, "000000000000")
	require.ErrorIs(t, err, ErrStateChanged, "the node is served something else than what was read")

	served, _ := cache.Served("node-a")
	_, err = control.Rollback(t.Context(), "node-a", "alice", target, served.ID())
	require.NoError(t, err)
	_, err = control.Unfreeze(t.Context(), "node-a", "alice", wrapped.SnapshotID(builder.built))
	require.NoError(t, err)
}

func TestUnfreezeKeepsTheFreezeWhileTheBuildFails(t *testing.T) {
	control, cache, builder, _ := fixture(t)
	_, err := control.Freeze(t.Context(), "node-a", "alice", "listener switch")
	require.NoError(t, err)
	builder.err = errors.New("snapshot is invalid: duplicate address")

	_, err = control.Unfreeze(t.Context(), "node-a", "alice", "")

	require.ErrorIs(t, err, ErrBuildFails)
	require.ErrorContains(t, err, "duplicate address")
	_, frozen := cache.Frozen("node-a")
	require.True(t, frozen)
}

// Two operators at once: whatever each gets back, what is kept in Kubernetes is what is in memory.
func TestConcurrentWritesKeepTheStoreInStep(t *testing.T) {
	for range 200 {
		control, cache, _, store := fixture(t)
		target := cache.History("node-a")[0].ID()
		served, _ := cache.Served("node-a")
		var wg sync.WaitGroup
		for _, expect := range []string{served.ID(), "000000000000"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = control.Rollback(t.Context(), "node-a", "alice", target, expect)
			}()
		}
		wg.Wait()
		require.Equal(t, cache.FrozenNodes(), store.kept())
	}
}
