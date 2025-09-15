package updater

import (
	"context"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func makeTestListenerCR(ns, name string, l *listenerv3.Listener) *v1alpha1.Listener {
	b, _ := protoutil.Marshaler.Marshal(l)
	return &v1alpha1.Listener{
		TypeMeta:   metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "Listener"},
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       &runtime.RawExtension{Raw: b},
	}
}

func makeTestVirtualServiceCR(ns, name string, nodeIDs []string, listenerName string) *v1alpha1.VirtualService {
	vs := &v1alpha1.VirtualService{
		TypeMeta: metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "VirtualService"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
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
	// Set nodeIDs using the proper method
	vs.SetNodeIDs(nodeIDs)
	return vs
}

func TestValidateListenerAddresses_SameAddressDifferentNodeIDs(t *testing.T) {
	// Test that same address:port is allowed across different nodeIDs
	s := store.New()

	// Create two listeners with the same address:port
	l1 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "127.0.0.1",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
	}

	l2 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "127.0.0.1",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
	}

	// Add listeners to store
	s.SetListener(makeTestListenerCR("default", "listener1", l1))
	s.SetListener(makeTestListenerCR("default", "listener2", l2))

	// Create VirtualServices that use these listeners for DIFFERENT nodeIDs
	vs1 := makeTestVirtualServiceCR("default", "vs1", []string{"node1"}, "listener1")
	vs2 := makeTestVirtualServiceCR("default", "vs2", []string{"node2"}, "listener2")

	s.SetVirtualService(vs1)
	s.SetVirtualService(vs2)

	// This should NOT produce an error since listeners belong to different nodeIDs
	snapshotCache := wrapped.NewSnapshotCache()
	err := validateListenerAddresses(s, snapshotCache, false)
	if err != nil {
		t.Fatalf("Expected no error when same address:port used by different nodeIDs, got: %v", err)
	}
}

func TestValidateListenerAddresses_SameAddressSameNodeID(t *testing.T) {
	// Test that same address:port is NOT allowed within the same nodeID
	s := store.New()

	// Create two listeners with the same address:port
	l1 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "127.0.0.1",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
	}

	l2 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "127.0.0.1",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
	}

	// Add listeners to store
	s.SetListener(makeTestListenerCR("default", "listener1", l1))
	s.SetListener(makeTestListenerCR("default", "listener2", l2))

	// Create VirtualServices that use these listeners for the SAME nodeID
	vs1 := makeTestVirtualServiceCR("default", "vs1", []string{"same-node"}, "listener1")
	vs2 := makeTestVirtualServiceCR("default", "vs2", []string{"same-node"}, "listener2")

	s.SetVirtualService(vs1)
	s.SetVirtualService(vs2)

	// This SHOULD produce an error since listeners belong to the same nodeID
	snapshotCache := wrapped.NewSnapshotCache()
	err := validateListenerAddresses(s, snapshotCache, false)
	if err == nil {
		t.Fatalf("Expected error when same address:port used within same nodeID, got no error")
	}

	expectedSubstring := "within nodeID 'same-node'"
	if !stringContains(err.Error(), expectedSubstring) {
		t.Fatalf("Expected error message to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestValidateListenerAddresses_MultipleNodeIDsOverlap(t *testing.T) {
	// Test complex scenario: listener1 used by [node1, node2], listener2 used by [node2, node3]
	// Same address should fail only where nodeIDs overlap (node2)
	s := store.New()

	// Create two listeners with the same address:port
	l1 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 9000,
					},
				},
			},
		},
	}

	l2 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 9000,
					},
				},
			},
		},
	}

	// Add listeners to store
	s.SetListener(makeTestListenerCR("default", "listener1", l1))
	s.SetListener(makeTestListenerCR("default", "listener2", l2))

	// listener1 is used by node1 and node2
	vs1 := makeTestVirtualServiceCR("default", "vs1", []string{"node1", "node2"}, "listener1")
	// listener2 is used by node2 and node3
	vs2 := makeTestVirtualServiceCR("default", "vs2", []string{"node2", "node3"}, "listener2")

	s.SetVirtualService(vs1)
	s.SetVirtualService(vs2)

	// This SHOULD produce an error because both listeners are used by node2
	snapshotCache := wrapped.NewSnapshotCache()
	err := validateListenerAddresses(s, snapshotCache, false)
	if err == nil {
		t.Fatalf("Expected error when same address:port used within overlapping nodeID, got no error")
	}

	expectedSubstring := "within nodeID 'node2'"
	if !stringContains(err.Error(), expectedSubstring) {
		t.Fatalf("Expected error message to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestBuildListenerToNodeIDsMapping(t *testing.T) {
	s := store.New()

	// Create listeners
	l1 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "127.0.0.1",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
	}

	s.SetListener(makeTestListenerCR("default", "listener1", l1))

	// Create VirtualServices
	vs1 := makeTestVirtualServiceCR("default", "vs1", []string{"node1", "node2"}, "listener1")
	vs2 := makeTestVirtualServiceCR("default", "vs2", []string{"node2", "node3"}, "listener1")

	s.SetVirtualService(vs1)
	s.SetVirtualService(vs2)

	// Build mapping
	snapshotCache := wrapped.NewSnapshotCache()
	mapping, err := buildListenerToNodeIDsMapping(s, snapshotCache)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	listenerNN := helpers.NamespacedName{Namespace: "default", Name: "listener1"}
	nodeIDs := mapping[listenerNN]

	if len(nodeIDs) != 3 {
		t.Fatalf("Expected 3 nodeIDs for listener1, got %d: %v", len(nodeIDs), nodeIDs)
	}

	// Check that all expected nodeIDs are present
	expectedNodeIDs := []string{"node1", "node2", "node3"}
	for _, expected := range expectedNodeIDs {
		found := false
		for _, actual := range nodeIDs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected nodeID '%s' not found in mapping: %v", expected, nodeIDs)
		}
	}
}

func TestBuildListenerToNodeIDsMapping_WithCommonVS(t *testing.T) {
	// Test that VirtualService with nodeID = "*" is properly expanded to all known nodeIDs
	s := store.New()

	// Create listener
	l1 := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: 9090,
					},
				},
			},
		},
	}

	s.SetListener(makeTestListenerCR("default", "common-listener", l1))

	// Create a common VirtualService with nodeID = "*"
	commonVS := makeTestVirtualServiceCR("default", "common-vs", []string{"*"}, "common-listener")
	s.SetVirtualService(commonVS)

	// Create snapshotCache with some known nodeIDs
	snapshotCache := wrapped.NewSnapshotCache()
	// Simulate known nodeIDs in the snapshot cache
	ctx := context.Background()
	_ = snapshotCache.SetSnapshot(ctx, "node-a", &cachev3.Snapshot{})
	_ = snapshotCache.SetSnapshot(ctx, "node-b", &cachev3.Snapshot{})
	_ = snapshotCache.SetSnapshot(ctx, "node-c", &cachev3.Snapshot{})

	// Build mapping
	mapping, err := buildListenerToNodeIDsMapping(s, snapshotCache)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	listenerNN := helpers.NamespacedName{Namespace: "default", Name: "common-listener"}
	nodeIDs := mapping[listenerNN]

	// Should expand "*" to all known nodeIDs in the snapshot cache
	expectedNodeIDs := []string{"node-a", "node-b", "node-c"}
	if len(nodeIDs) != len(expectedNodeIDs) {
		t.Fatalf("Expected %d nodeIDs for common-listener, got %d: %v", len(expectedNodeIDs), len(nodeIDs), nodeIDs)
	}

	// Check that all expected nodeIDs are present
	for _, expected := range expectedNodeIDs {
		found := false
		for _, actual := range nodeIDs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected nodeID '%s' not found in mapping: %v", expected, nodeIDs)
		}
	}
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
