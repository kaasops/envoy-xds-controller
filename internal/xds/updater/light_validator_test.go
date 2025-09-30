package updater

import (
	"context"
	"errors"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// helper to create minimal VS with ns/name and nodeIDs via annotation
func makeVS(name string, nodeIDs []string) *v1alpha1.VirtualService {
	vs := &v1alpha1.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name}}
	// ensure annotations map is non-nil for SetNodeIDs
	vs.SetAnnotations(map[string]string{})
	vs.SetNodeIDs(nodeIDs)
	return vs
}

// helper to create Listener CR with given host:port
func makeListenerCR(ns, name, host string, port uint32) *v1alpha1.Listener {
	l := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: host, PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: port}}}},
	}
	b, _ := protoutil.Marshaler.Marshal(l)
	return &v1alpha1.Listener{
		TypeMeta:   metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "Listener"},
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       &runtime.RawExtension{Raw: b},
	}
}

// helper to create VirtualService with listener reference
func makeVSWithListener(name string, nodeIDs []string, listenerName string) *v1alpha1.VirtualService {
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "ns",
			Name:        name,
			Annotations: make(map[string]string),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener: &v1alpha1.ResourceRef{
					Name: listenerName,
				},
			},
		},
	}
	vs.SetNodeIDs(nodeIDs)
	return vs
}

func withStubbedBuilder(t *testing.T, f func(vs *v1alpha1.VirtualService, store store.Store) (*resbuilder.Resources, error)) func() {
	t.Helper()
	prev := buildVSResources
	buildVSResources = f
	return func() { buildVSResources = prev }
}

func TestLightValidator_CoverageMiss_WithIndices(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	// Stub builder to return one domain without touching listener/template/etc.
	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"example.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); !errors.Is(err, ErrLightValidationInsufficientCoverage) {
		t.Fatalf("expected ErrLightValidationInsufficientCoverage, got %v", err)
	}
}

func TestLightValidator_DuplicateWithinVS(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	// Provide coverage for nodeA with empty set to avoid coverage miss
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"a.com", "a.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err == nil || err.Error() != "duplicate domain 'a.com' within VirtualService" {
		t.Fatalf("expected duplicate within VS error, got %v", err)
	}
}

func TestLightValidator_DomainCollisionAcrossNodes(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {"b.com": {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"b.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() != "duplicate domain 'b.com' for node nodeA" {
		t.Fatalf("expected duplicate domain across nodes error, got %v", err)
	}
}

func TestLightValidator_UpdatePrevVSExcluded(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	// Index includes domain that belongs to previous version of the same VS
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"node1": {"b.com": {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		// Return domains based on VS name to differentiate prev/new
		switch vs.Name {
		case "prev":
			return &resbuilder.Resources{Domains: []string{"b.com"}}, nil
		default:
			return &resbuilder.Resources{Domains: []string{"b.com"}}, nil
		}
	})
	defer restore()

	prev := makeVS("prev", []string{"node1"})
	vs := makeVS("vs1", []string{"node1"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, prev, true); err != nil {
		t.Fatalf("expected no error because prevVS domains should be excluded, got %v", err)
	}
}

func TestLightValidator_ListenerDuplicateDetected(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	// Two listeners with the same host:port
	st.SetListener(makeListenerCR("ns", "l1", "127.0.0.1", 9090))
	st.SetListener(makeListenerCR("ns", "l2", "127.0.0.1", 9090))

	// Create VirtualServices that use these listeners for the SAME nodeID
	// This should trigger a duplicate detection within the nodeID
	vs1 := makeVSWithListener("vs1", []string{"n"}, "l1")
	vs2 := makeVSWithListener("vs2", []string{"n"}, "l2")
	st.SetVirtualService(vs1)
	st.SetVirtualService(vs2)

	// Provide node coverage to not trip coverage miss
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"n": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	// Stub builder to return any domain (not important here)
	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"x"}}, nil
	})
	defer restore()

	// Test with either of the VirtualServices - both should detect the conflict
	vs := makeVSWithListener("new-vs", []string{"n"}, "l1")
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() == "" {
		t.Fatalf("expected listener duplicate error, got %v", err)
	}
	// Ensure error message format follows the documented pattern when index is enabled
	e := err.Error()
	want := "within nodeID 'n'"
	if !contains(e, want) {
		// Try to be resilient in case of different listener names order; just ensure it mentions duplicate address and nodeID
		if !containsAll(e, []string{"duplicate address", "127.0.0.1:9090", "nodeID"}) {
			t.Fatalf("unexpected error message: %v", err)
		}
	}
}

// small helper to check substrings
func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (index(s, sub) >= 0)))
}

// naive substring search to avoid importing strings to keep imports tidy
func index(s, sub string) int {
	// simple implementation
	n := len(s)
	m := len(sub)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}

func TestLightValidator_CommonVS_NoCollision_OK(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	// Provide index entries for both nodes with empty sets (coverage present)
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"n1": {}, "n2": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	// Register nodes in snapshot cache so common VS expands to these nodes
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n1", &cachev3.Snapshot{})
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n2", &cachev3.Snapshot{})

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"a.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs-common", []string{"*"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err != nil {
		t.Fatalf("expected no error for common VS with empty domain sets, got %v", err)
	}
}

func TestLightValidator_CommonVS_DomainCollisionDetected(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"n1": {}, "n2": {"a.com": {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n1", &cachev3.Snapshot{})
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n2", &cachev3.Snapshot{})

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"a.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs-common", []string{"*"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() != "duplicate domain 'a.com' for node n2" {
		t.Fatalf("expected collision on n2 for 'a.com', got %v", err)
	}
}

func TestLightValidator_MultiNode_UpdatePrevExclusionPerNode(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	// Both nodes have x.com currently
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"n1": {"x.com": {}}, "n2": {"x.com": {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"x.com"}}, nil
	})
	defer restore()

	prev := makeVS("prev", []string{"n1"}) // prevVS affected only n1
	vs := makeVS("new", []string{"n1", "n2"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, prev, true)
	if err == nil || err.Error() != "duplicate domain 'x.com' for node n2" {
		t.Fatalf("expected duplicate only on n2 (no exclusion there), got %v", err)
	}
}

func TestLightValidator_NoFalseFallback_WithEmptyNodes(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.NewStoreAdapter()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"a": {}, "b": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, st store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"z.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"a", "b"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err != nil {
		t.Fatalf("expected success with empty domain sets for nodes a,b, got %v", err)
	}
}
