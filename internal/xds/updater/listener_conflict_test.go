package updater

import (
	"context"
	"errors"
	"fmt"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// conflictOn returns the error the snapshot cache gives for two listeners of nodeID on one address.
func conflictOn(t *testing.T, nodeID string) error {
	t.Helper()
	listener := func(name string) *listenerv3.Listener {
		return &listenerv3.Listener{
			Name: name,
			Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{
				Address:       "0.0.0.0",
				PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 443},
			}}},
		}
	}
	snapshot, err := gcp.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.ListenerType: {listener("ns/https"), listener("ns/https-module")},
	})
	require.NoError(t, err)
	err = wrapped.NewSnapshotCache().SetSnapshot(t.Context(), nodeID, snapshot)
	require.Error(t, err)
	return err
}

func conflictNodes(conflicts []*wrapped.DuplicateListenerAddress) []string {
	nodes := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		nodes = append(nodes, conflict.NodeID())
	}
	return nodes
}

func TestToleratedListenerConflicts(t *testing.T) {
	tolerated := []string{"node-a", "node-b"}
	tests := []struct {
		name  string
		err   func(t *testing.T) error
		nodes []string
		want  []string
	}{
		{
			name: "no error",
			err:  func(*testing.T) error { return nil },
		},
		{
			name: "conflict on a tolerated node behind wrapping",
			err: func(t *testing.T) error {
				return fmt.Errorf("failed to build snapshot: %w", conflictOn(t, "node-a"))
			},
			want: []string{"node-a"},
		},
		{
			name: "conflicts on two tolerated nodes come back sorted",
			err: func(t *testing.T) error {
				return fmt.Errorf("dry run: %w", multierr.Combine(conflictOn(t, "node-b"), conflictOn(t, "node-a")))
			},
			want: []string{"node-a", "node-b"},
		},
		{
			name: "conflict on a node outside the list",
			err:  func(t *testing.T) error { return conflictOn(t, "node-c") },
		},
		{
			name: "tolerated conflict next to a node outside the list",
			err: func(t *testing.T) error {
				return multierr.Combine(conflictOn(t, "node-a"), conflictOn(t, "node-c"))
			},
		},
		{
			name: "tolerated conflict next to another error",
			err: func(t *testing.T) error {
				return multierr.Combine(conflictOn(t, "node-a"), errors.New("missing dependency"))
			},
		},
		{
			name: "tolerated conflict next to a timeout",
			err: func(t *testing.T) error {
				return errors.Join(conflictOn(t, "node-a"), context.DeadlineExceeded)
			},
		},
		{
			name:  "empty list tolerates nothing",
			err:   func(t *testing.T) error { return conflictOn(t, "node-a") },
			nodes: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := tolerated
			if tt.nodes != nil {
				nodes = tt.nodes
			}
			got := ToleratedListenerConflicts(tt.err(t), nodes)
			if tt.want == nil {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tt.want, conflictNodes(got))
		})
	}
}

// moveFixture is a node with two services on one listener and a second listener on the same address.
func moveFixture(t *testing.T) (*CacheUpdater, *v1alpha1.VirtualService) {
	t.Helper()
	s := store.New()
	s.SetListener(makeListenerCR("https", "0.0.0.0", 443))
	s.SetListener(makeListenerCR("https-module", "0.0.0.0", 443))
	s.SetVirtualService(makeVSWithListener("first", []string{"node-a"}, "https"))
	s.SetVirtualService(makeVSWithListener("second", []string{"node-a"}, "https"))
	s.SetVirtualService(makeVSWithListener("neighbour", []string{"node-b"}, "https"))
	t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		listener, err := vs.GetListenerNamespacedName()
		require.NoError(t, err)
		return &resbuilder.Resources{Listener: listener, Domains: []string{vs.Name + ".example.com"}}, nil
	}))
	return NewCacheUpdater(wrapped.NewSnapshotCache(), s), makeVSWithListener("first", []string{"node-a"}, "https-module")
}

func TestDryRunReportsTheMovingNode(t *testing.T) {
	updater, moved := moveFixture(t)

	err := updater.DryBuildSnapshotsWithVirtualService(t.Context(), moved)

	require.ErrorContains(t, err, "duplicate address")
	require.Equal(t, []string{"node-a"}, conflictNodes(ToleratedListenerConflicts(err, []string{"node-a"})))
	require.Nil(t, ToleratedListenerConflicts(err, []string{"node-b"}))
	require.Nil(t, ToleratedListenerConflicts(err, nil))
}

func TestLightCheckSkipsConflictsOnToleratedNodes(t *testing.T) {
	s := store.New()
	s.SetListener(makeListenerCR("https", "0.0.0.0", 443))
	s.SetListener(makeListenerCR("https-module", "0.0.0.0", 443))
	s.SetVirtualService(makeVSWithListener("first", []string{"node-a"}, "https"))
	s.SetVirtualService(makeVSWithListener("second", []string{"node-a"}, "https-module"))
	cache := wrapped.NewSnapshotCache()
	tolerated := map[string]struct{}{"node-a": {}}

	for _, indices := range []bool{false, true} {
		require.NoError(t, validateListenerAddresses(s, cache, indices, tolerated), "indices=%v", indices)
		require.ErrorContains(t, validateListenerAddresses(s, cache, indices, nil), "within nodeID 'node-a'",
			"indices=%v", indices)
	}

	s.SetVirtualService(makeVSWithListener("third", []string{"node-c"}, "https"))
	s.SetVirtualService(makeVSWithListener("fourth", []string{"node-c"}, "https-module"))
	for _, indices := range []bool{false, true} {
		require.ErrorContains(t, validateListenerAddresses(s, cache, indices, tolerated), "within nodeID 'node-c'",
			"indices=%v", indices)
	}
}

func TestLightValidationPassesWhileToleratedNodeIsHeld(t *testing.T) {
	updater, moved := moveFixture(t)
	require.NoError(t, updater.RebuildSnapshots(t.Context()))
	updater.store.SetVirtualService(moved)
	require.ErrorContains(t, updater.RebuildSnapshots(t.Context()), "duplicate address")
	neighbour := makeVSWithListener("neighbour", []string{"node-b"}, "https")

	require.NoError(t, updater.DryValidateVirtualServiceLight(t.Context(), neighbour, nil, false, []string{"node-a"}))
	require.ErrorContains(t, updater.DryValidateVirtualServiceLight(t.Context(), neighbour, nil, false, nil),
		"within nodeID 'node-a'")
}

// holdFixture publishes two nodes, then holds node-a on its snapshot with a listener address conflict.
// Each service uses the secret named in secrets, or none.
func holdFixture(t *testing.T, domains, secrets map[string]string) *CacheUpdater {
	t.Helper()
	s := store.New()
	s.SetListener(makeListenerCR("https", "0.0.0.0", 443))
	s.SetListener(makeListenerCR("https-module", "0.0.0.0", 443))
	s.SetVirtualService(makeVSWithListener("first", []string{"node-a"}, "https"))
	s.SetVirtualService(makeVSWithListener("second", []string{"node-a"}, "https"))
	s.SetVirtualService(makeVSWithListener("neighbour", []string{"node-b"}, "https"))
	t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		listener, err := vs.GetListenerNamespacedName()
		require.NoError(t, err)
		res := &resbuilder.Resources{Listener: listener, Domains: []string{vs.Name + ".example.com"}}
		if domain, ok := domains[vs.Name]; ok {
			res.Domains = []string{domain}
		}
		if secret, ok := secrets[vs.Name]; ok {
			res.UsedSecrets = []helpers.NamespacedName{{Namespace: testNamespace, Name: secret}}
		}
		return res, nil
	}))
	u := NewCacheUpdater(wrapped.NewSnapshotCache(), s)
	require.NoError(t, u.RebuildSnapshots(t.Context()))
	u.store.SetVirtualService(makeVSWithListener("first", []string{"node-a"}, "https-module"))
	require.ErrorContains(t, u.RebuildSnapshots(t.Context()), "duplicate address")
	return u
}

func TestDomainIndexFollowsPublishedNodesWhileAnotherIsHeld(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	u := holdFixture(t, map[string]string{"neighbour": "neighbour.example.com"}, nil)
	u.store.SetVirtualService(makeVSWithListener("added", []string{"node-b"}, "https"))
	require.ErrorContains(t, u.RebuildSnapshots(t.Context()), "duplicate address")

	index := u.store.GetNodeDomainsIndex()
	require.Contains(t, index["node-b"], testNamespace+"/https|added.example.com", "published node")
	require.Contains(t, index["node-a"], testNamespace+"/https|first.example.com", "held node keeps what it serves")
	require.Contains(t, index["node-a"], testNamespace+"/https-module|first.example.com", "and what it is about to serve")

	clash := makeVSWithListener("clash", []string{"node-b"}, "https")
	t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		listener, err := vs.GetListenerNamespacedName()
		require.NoError(t, err)
		domain := vs.Name + ".example.com"
		if vs.Name == "clash" {
			domain = "added.example.com"
		}
		return &resbuilder.Resources{Listener: listener, Domains: []string{domain}}, nil
	}))
	require.ErrorContains(t,
		u.DryValidateVirtualServiceLight(t.Context(), clash, nil, true, []string{"node-a"}), "duplicate domain")
}

func TestUsedSecretsCoverHeldAndPublishedNodes(t *testing.T) {
	u := holdFixture(t, nil, map[string]string{"second": "old-cert", "neighbour": "neighbour-cert"})
	u.store.SetVirtualService(makeVSWithListener("added", []string{"node-b"}, "https"))
	t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		listener, err := vs.GetListenerNamespacedName()
		require.NoError(t, err)
		res := &resbuilder.Resources{Listener: listener, Domains: []string{vs.Name + ".example.com"}}
		switch vs.Name {
		case "neighbour":
			res.UsedSecrets = []helpers.NamespacedName{{Namespace: testNamespace, Name: "neighbour-cert"}}
		case "added":
			res.UsedSecrets = []helpers.NamespacedName{{Namespace: testNamespace, Name: "new-cert"}}
		}
		return res, nil
	}))
	require.ErrorContains(t, u.RebuildSnapshots(t.Context()), "duplicate address")

	used := u.GetUsedSecrets()
	require.Contains(t, used, helpers.NamespacedName{Namespace: testNamespace, Name: "new-cert"}, "published node")
	require.Contains(t, used, helpers.NamespacedName{Namespace: testNamespace, Name: "old-cert"},
		"held node still serves it")
}

func TestListenerChecksStayForToleratedNodes(t *testing.T) {
	updater, moved := moveFixture(t)
	require.NoError(t, updater.DryValidateListenerAddresses(t.Context(), moved, false, []string{"node-a"}))
	require.ErrorContains(t, updater.DryValidateListenerAddresses(t.Context(), moved, false, nil),
		"within nodeID 'node-a'")

	raw, err := protoutil.Marshaler.Marshal(&listenerv3.Listener{})
	require.NoError(t, err)
	updater.store.SetListener(&v1alpha1.Listener{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "bare"},
		Spec:       &runtime.RawExtension{Raw: raw},
	})
	bare := makeVSWithListener("first", []string{"node-a"}, "bare")
	require.ErrorContains(t, updater.DryValidateListenerAddresses(t.Context(), bare, false, []string{"node-a"}),
		"has no address configured")
}

func TestLightCheckFindsListenersInTheirOwnNamespace(t *testing.T) {
	s := store.New()
	shared := makeListenerCR("other", "0.0.0.0", 443)
	shared.Namespace = "shared"
	s.SetListener(shared)
	vs := makeVSWithListener("first", []string{"node-a"}, "other")
	namespace := "shared"
	vs.Spec.Listener.Namespace = &namespace
	s.SetVirtualService(vs)

	require.NoError(t, validateListenerAddresses(s, wrapped.NewSnapshotCache(), false, nil))
}
