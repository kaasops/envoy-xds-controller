package updater

import (
	"context"
	"errors"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Domains are unique per listener, so tests that seed the domain index have to use the
// same composite key the validator builds.
const testNamespace = "ns"

var (
	testListenerA = helpers.NamespacedName{Namespace: testNamespace, Name: "listener-a"}
	testListenerB = helpers.NamespacedName{Namespace: testNamespace, Name: "listener-b"}
)

func ldk(listener helpers.NamespacedName, domain string) string {
	return listenerDomain(listener, domain)
}

// helper to create minimal VS with ns/name and nodeIDs via annotation
func makeVS(name string, nodeIDs []string) *v1alpha1.VirtualService {
	vs := &v1alpha1.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name}}
	// ensure annotations map is non-nil for SetNodeIDs
	vs.SetAnnotations(map[string]string{})
	vs.SetNodeIDs(nodeIDs)
	return vs
}

// helper to create Listener CR with given host:port
func makeListenerCR(name, host string, port uint32) *v1alpha1.Listener {
	l := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address:       host,
					PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: port},
				},
			},
		},
	}
	b, _ := protoutil.Marshaler.Marshal(l)
	return &v1alpha1.Listener{
		TypeMeta:   metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "Listener"},
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: name},
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

// snapshotServing builds a snapshot where a listener serves one VirtualService: the
// filter chain and the route configuration share the VirtualService name, which is how
// the validator attributes a route configuration to its listener.
func snapshotServing(t *testing.T, listener helpers.NamespacedName, vsName, domain string) *cachev3.Snapshot {
	t.Helper()
	l := &listenerv3.Listener{
		Name:         listener.String(),
		FilterChains: []*listenerv3.FilterChain{{Name: vsName}},
	}
	rc := &routev3.RouteConfiguration{
		Name: vsName,
		VirtualHosts: []*routev3.VirtualHost{
			{Name: vsName + "-vh", Domains: []string{domain}},
		},
	}
	snap, err := cachev3.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.ListenerType: {l},
		resource.RouteType:    {rc},
	})
	if err != nil {
		t.Fatalf("failed to build snapshot: %v", err)
	}
	return snap
}

func withStubbedBuilder(
	t *testing.T,
	f func(vs *v1alpha1.VirtualService, store store.Store) (*resbuilder.Resources, error),
) func() {
	t.Helper()
	prev := buildVSResources
	buildVSResources = f
	return func() { buildVSResources = prev }
}

func TestLightValidator_CoverageMiss_WithIndices(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	// Stub builder to return one domain without touching listener/template/etc.
	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"example.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if !errors.Is(err, ErrLightValidationInsufficientCoverage) {
		t.Fatalf("expected ErrLightValidationInsufficientCoverage, got %v", err)
	}
}

func TestLightValidator_DuplicateWithinVS(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	// Provide coverage for nodeA with empty set to avoid coverage miss
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"a.com", "a.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() != "duplicate domain 'a.com' within VirtualService" {
		t.Fatalf("expected duplicate within VS error, got %v", err)
	}
}

func TestLightValidator_DomainCollisionAcrossNodes(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {ldk(testListenerA, "b.com"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerA}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() != "duplicate domain 'b.com' for node nodeA on listener ns/listener-a" {
		t.Fatalf("expected duplicate domain across nodes error, got %v", err)
	}
}

// TestLightValidator_SameDomainDifferentListener is the regression test for the
// false positive: the same domain on two different listeners must be allowed.
func TestLightValidator_SameDomainDifferentListener(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	// listener-a already serves b.com on this node
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {ldk(testListenerA, "b.com"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	// the candidate serves the same domain, but on listener-b
	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerB}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err != nil {
		t.Fatalf("same domain on a different listener must be allowed, got %v", err)
	}
}

// The two tests below drive the snapshot branch of buildExistingDomainsMap - the one
// that has to recover the listener from the snapshot itself - by passing
// validationIndices=false. The index branch is covered by the tests around them.

func TestLightValidator_Snapshot_SameDomainDifferentListener(t *testing.T) {
	st := store.New()
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)
	// listener-a serves b.com on this node
	snap := snapshotServing(t, testListenerA, "ns/vs-existing", "b.com")
	if err := cu.snapshotCache.SetSnapshot(context.Background(), "nodeA", snap); err != nil {
		t.Fatalf("failed to set snapshot: %v", err)
	}

	// the candidate serves the same domain on listener-b
	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerB}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, false); err != nil {
		t.Fatalf("same domain on a different listener must be allowed, got %v", err)
	}
}

func TestLightValidator_Snapshot_SameDomainSameListener(t *testing.T) {
	st := store.New()
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)
	snap := snapshotServing(t, testListenerA, "ns/vs-existing", "b.com")
	if err := cu.snapshotCache.SetSnapshot(context.Background(), "nodeA", snap); err != nil {
		t.Fatalf("failed to set snapshot: %v", err)
	}

	// the candidate claims the same domain on the very same listener
	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerA}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, false)
	if err == nil || err.Error() != "duplicate domain 'b.com' for node nodeA on listener ns/listener-a" {
		t.Fatalf("expected duplicate on the same listener, got %v", err)
	}
}

func TestRouteConfigToListener_DuplicateFilterChainName(t *testing.T) {
	listeners := []*listenerv3.Listener{
		{Name: testListenerA.String(), FilterChains: []*listenerv3.FilterChain{{Name: "ns/vs"}}},
		{Name: testListenerB.String(), FilterChains: []*listenerv3.FilterChain{{Name: "ns/vs"}}},
	}
	if _, err := routeConfigToListener(listeners); err == nil {
		t.Fatal("expected an error when one filter chain name appears on two listeners")
	}
}

// TestLightValidator_SameDomainListenersSharingPort covers the requirement that two
// listeners stay independent even when they share a port and only differ by address.
// It also exercises the listener address check, which keys on host:port and therefore
// has to accept both listeners in the first place.
func TestLightValidator_SameDomainListenersSharingPort(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	st.SetListener(makeListenerCR("listener-a", "0.0.0.0", 80))
	st.SetListener(makeListenerCR("listener-b", "10.0.0.1", 80))
	// a VirtualService on listener-a so the listener address check actually sees both
	st.SetVirtualService(makeVSWithListener("vs-a", []string{"nodeA"}, "listener-a"))
	// listener-a already serves the domain on this node
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {ldk(testListenerA, "b.com"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerB}, nil
	})
	defer restore()

	vs := makeVSWithListener("vs1", []string{"nodeA"}, "listener-b")
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err != nil {
		t.Fatalf("listeners sharing a port must stay independent, got %v", err)
	}
}

// TestLightValidator_WildcardDifferentListener covers the same case for '*', which is
// the catch-all domain and therefore the one most likely to be reused across listeners.
func TestLightValidator_WildcardDifferentListener(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {ldk(testListenerA, "*"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"*"}, Listener: testListenerB}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err != nil {
		t.Fatalf("'*' on a different listener must be allowed, got %v", err)
	}
}

// TestLightValidator_WildcardSameListener makes sure relaxing the scope did not
// disable the check that actually matters.
func TestLightValidator_WildcardSameListener(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"nodeA": {ldk(testListenerA, "*"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"*"}, Listener: testListenerA}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"nodeA"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() != "duplicate domain '*' for node nodeA on listener ns/listener-a" {
		t.Fatalf("expected duplicate '*' on the same listener, got %v", err)
	}
}

func TestLightValidator_UpdatePrevVSExcluded(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	// Index includes domain that belongs to previous version of the same VS
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"node1": {ldk(testListenerA, "b.com"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		// Return domains based on VS name to differentiate prev/new
		switch vs.Name {
		case "prev":
			return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerA}, nil
		default:
			return &resbuilder.Resources{Domains: []string{"b.com"}, Listener: testListenerA}, nil
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
	st := store.New()
	// Two listeners with the same host:port
	st.SetListener(makeListenerCR("l1", "127.0.0.1", 9090))
	st.SetListener(makeListenerCR("l2", "127.0.0.1", 9090))

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
	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
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
	st := store.New()
	// Provide index entries for both nodes with empty sets (coverage present)
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"n1": {}, "n2": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	// Register nodes in snapshot cache so common VS expands to these nodes
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n1", &cachev3.Snapshot{})
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n2", &cachev3.Snapshot{})

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
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
	st := store.New()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"n1": {}, "n2": {ldk(testListenerA, "a.com"): {}}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n1", &cachev3.Snapshot{})
	_ = cu.snapshotCache.SetSnapshot(context.Background(), "n2", &cachev3.Snapshot{})

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"a.com"}, Listener: testListenerA}, nil
	})
	defer restore()

	vs := makeVS("vs-common", []string{"*"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true)
	if err == nil || err.Error() != "duplicate domain 'a.com' for node n2 on listener ns/listener-a" {
		t.Fatalf("expected collision on n2 for 'a.com', got %v", err)
	}
}

func TestLightValidator_MultiNode_UpdatePrevExclusionPerNode(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	// Both nodes have x.com currently
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{
		"n1": {ldk(testListenerA, "x.com"): {}},
		"n2": {ldk(testListenerA, "x.com"): {}},
	})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"x.com"}, Listener: testListenerA}, nil
	})
	defer restore()

	prev := makeVS("prev", []string{"n1"}) // prevVS affected only n1
	vs := makeVS("new", []string{"n1", "n2"})
	err := cu.DryValidateVirtualServiceLight(context.Background(), vs, prev, true)
	if err == nil || err.Error() != "duplicate domain 'x.com' for node n2 on listener ns/listener-a" {
		t.Fatalf("expected duplicate only on n2 (no exclusion there), got %v", err)
	}
}

func TestLightValidator_NoFalseFallback_WithEmptyNodes(t *testing.T) {
	t.Setenv("WEBHOOK_VALIDATION_INDICES", "1")
	st := store.New()
	st.ReplaceNodeDomainsIndex(map[string]map[string]struct{}{"a": {}, "b": {}})
	cu := NewCacheUpdater(wrapped.NewSnapshotCache(), st)

	restore := withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		return &resbuilder.Resources{Domains: []string{"z.com"}}, nil
	})
	defer restore()

	vs := makeVS("vs1", []string{"a", "b"})
	if err := cu.DryValidateVirtualServiceLight(context.Background(), vs, nil, true); err != nil {
		t.Fatalf("expected success with empty domain sets for nodes a,b, got %v", err)
	}
}
