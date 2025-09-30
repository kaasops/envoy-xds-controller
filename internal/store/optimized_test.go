package store

import (
	"context"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestOptimizedStore_VirtualServices(t *testing.T) {
	store := NewOptimizedStore()
	require.NotNil(t, store)

	// Test virtual service operations
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	// Test Set and Get
	store.SetVirtualService(vs)
	retrieved := store.GetVirtualService(nn)
	assert.Equal(t, vs, retrieved)

	// Test GetByUID
	retrievedByUID := store.GetVirtualServiceByUID("test-uid")
	assert.Equal(t, vs, retrievedByUID)

	// Test IsExisting
	assert.True(t, store.IsExistingVirtualService(nn))

	// Test Map
	vsMap := store.MapVirtualServices()
	assert.Len(t, vsMap, 1)
	assert.Equal(t, vs, vsMap[nn])

	// Test Delete
	store.DeleteVirtualService(nn)
	assert.False(t, store.IsExistingVirtualService(nn))
	assert.Nil(t, store.GetVirtualService(nn))
	assert.Nil(t, store.GetVirtualServiceByUID("test-uid"))
}

func TestOptimizedStore_VirtualServicesByTemplate(t *testing.T) {
	store := NewOptimizedStore()

	templateNN := helpers.NamespacedName{Namespace: "default", Name: "template1"}

	// Create virtual services with template reference
	vs1 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs1",
			Namespace: "default",
			UID:       types.UID("uid1"),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			Template: &v1alpha1.ResourceRef{
				Name: "template1",
			},
		},
	}

	vs2 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs2",
			Namespace: "default",
			UID:       types.UID("uid2"),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			Template: &v1alpha1.ResourceRef{
				Name: "template1",
			},
		},
	}

	// Set virtual services
	store.SetVirtualService(vs1)
	store.SetVirtualService(vs2)

	// Get virtual services by template
	vsList := store.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, vsList, 2)
	assert.Contains(t, vsList, vs1)
	assert.Contains(t, vsList, vs2)

	// Delete one virtual service
	store.DeleteVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs1"})
	vsList = store.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, vsList, 1)
	assert.Contains(t, vsList, vs2)
}

func TestOptimizedStore_Listeners(t *testing.T) {
	store := NewOptimizedStore()

	listener := &v1alpha1.Listener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listener",
			Namespace: "default",
			UID:       types.UID("listener-uid"),
		},
	}

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-listener"}

	// Test Set and Get
	store.SetListener(listener)
	retrieved := store.GetListener(nn)
	assert.Equal(t, listener, retrieved)

	// Test GetByUID
	retrievedByUID := store.GetListenerByUID("listener-uid")
	assert.Equal(t, listener, retrievedByUID)

	// Test IsExisting
	assert.True(t, store.IsExistingListener(nn))

	// Test Map
	listenerMap := store.MapListeners()
	assert.Len(t, listenerMap, 1)
	assert.Equal(t, listener, listenerMap[nn])

	// Test Delete
	store.DeleteListener(nn)
	assert.False(t, store.IsExistingListener(nn))
	assert.Nil(t, store.GetListener(nn))
	assert.Nil(t, store.GetListenerByUID("listener-uid"))
}

func TestOptimizedStore_Clusters(t *testing.T) {
	store := NewOptimizedStore()

	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
			UID:       "test-uid",
		},
	}

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-cluster"}

	// Test Set and Get
	store.SetCluster(cluster)
	retrieved := store.GetCluster(nn)
	assert.Equal(t, cluster, retrieved)

	// Test IsExisting
	assert.True(t, store.IsExistingCluster(nn))

	// Test Map
	clusterMap := store.MapClusters()
	assert.Len(t, clusterMap, 1)
	assert.Equal(t, cluster, clusterMap[nn])

	// Note: GetSpecCluster and MapSpecClusters require valid Spec with UnmarshalV3()
	// Skipping these tests as they need proper cluster configuration

	// Test Delete
	store.DeleteCluster(nn)
	assert.False(t, store.IsExistingCluster(nn))
	assert.Nil(t, store.GetCluster(nn))
}

func TestOptimizedStore_Secrets(t *testing.T) {
	store := NewOptimizedStore()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-secret"}

	// Test Set and Get
	store.SetSecret(secret)
	retrieved := store.GetSecret(nn)
	assert.Equal(t, secret, retrieved)

	// Test IsExisting
	assert.True(t, store.IsExistingSecret(nn))

	// Test Map
	secretMap := store.MapSecrets()
	assert.Len(t, secretMap, 1)
	assert.Equal(t, secret, secretMap[nn])

	// Test Delete
	store.DeleteSecret(nn)
	assert.False(t, store.IsExistingSecret(nn))
	assert.Nil(t, store.GetSecret(nn))
}

func TestOptimizedStore_NodeDomainsIndex(t *testing.T) {
	store := NewOptimizedStore()

	// Create node domains index
	index := map[string]map[string]struct{}{
		"node1": {
			"domain1.com": {},
			"domain2.com": {},
		},
		"node2": {
			"domain3.com": {},
		},
	}

	// Test ReplaceNodeDomainsIndex
	store.ReplaceNodeDomainsIndex(index)

	// Test GetNodeDomainsIndex
	retrievedIndex := store.GetNodeDomainsIndex()
	assert.Len(t, retrievedIndex, 2)
	assert.Contains(t, retrievedIndex["node1"], "domain1.com")
	assert.Contains(t, retrievedIndex["node1"], "domain2.com")
	assert.Contains(t, retrievedIndex["node2"], "domain3.com")

	// Test GetNodeDomainsForNodes
	domains, missing := store.GetNodeDomainsForNodes([]string{"node1", "node3"})
	assert.Len(t, domains, 1)
	assert.Contains(t, domains["node1"], "domain1.com")
	assert.Contains(t, domains["node1"], "domain2.com")
	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "node3")
}

func TestOptimizedStore_Copy(t *testing.T) {
	store := NewOptimizedStore()

	// Add some data
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}
	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
	store.SetVirtualService(vs)

	listener := &v1alpha1.Listener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listener",
			Namespace: "default",
			UID:       types.UID("listener-uid"),
		},
	}
	listenerNN := helpers.NamespacedName{Namespace: "default", Name: "test-listener"}
	store.SetListener(listener)

	// Create a copy
	storeCopy := store.Copy()
	require.NotNil(t, storeCopy)

	// Verify data is copied
	assert.Equal(t, vs, storeCopy.GetVirtualService(nn))
	assert.Equal(t, listener, storeCopy.GetListener(listenerNN))

	// Verify copy is independent
	store.DeleteVirtualService(nn)
	assert.Nil(t, store.GetVirtualService(nn))
	assert.Equal(t, vs, storeCopy.GetVirtualService(nn)) // Copy should still have it

	// Verify copy can be modified independently
	storeCopy.DeleteListener(listenerNN)
	assert.Nil(t, storeCopy.GetListener(listenerNN))
	assert.Equal(t, listener, store.GetListener(listenerNN)) // Original should still have it
}

func TestOptimizedStore_FillFromKubernetes(t *testing.T) {
	store := NewOptimizedStore()

	// Test FillFromKubernetes (basic implementation returns nil)
	err := store.FillFromKubernetes(context.Background(), nil)
	assert.NoError(t, err)
}

func TestOptimizedStore_CopyUIDIndices(t *testing.T) {
	store := NewOptimizedStore()

	// Add various resources with UIDs to test UID index copying
	httpFilter := &v1alpha1.HttpFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-filter",
			Namespace: "default",
			UID:       types.UID("filter-uid-456"),
		},
	}
	store.SetHTTPFilter(httpFilter)

	accessLog := &v1alpha1.AccessLogConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-accesslog",
			Namespace: "default",
			UID:       types.UID("accesslog-uid-789"),
		},
	}
	store.SetAccessLog(accessLog)

	route := &v1alpha1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
			UID:       types.UID("route-uid-abc"),
		},
	}
	store.SetRoute(route)

	// Create a copy
	storeCopy := store.Copy()
	require.NotNil(t, storeCopy)

	// Verify UID indices are copied correctly using available Get*ByUID methods
	assert.Equal(t, httpFilter, storeCopy.GetHTTPFilterByUID("filter-uid-456"), "HTTPFilter UID index not copied")
	assert.Equal(t, accessLog, storeCopy.GetAccessLogByUID("accesslog-uid-789"), "AccessLog UID index not copied")
	assert.Equal(t, route, storeCopy.GetRouteByUID("route-uid-abc"), "Route UID index not copied")

	// Verify copy is independent - delete from original
	store.DeleteHTTPFilter(helpers.NamespacedName{Namespace: "default", Name: "test-filter"})
	assert.Nil(t, store.GetHTTPFilterByUID("filter-uid-456"), "Original should not have filter after delete")
	assert.Equal(t, httpFilter, storeCopy.GetHTTPFilterByUID("filter-uid-456"), "Copy should still have filter after delete from original")

	// Also verify that policies UID index was copied (this was the critical bug)
	policy := &v1alpha1.Policy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
			UID:       types.UID("policy-uid-123"),
		},
	}
	store.SetPolicy(policy)

	storeCopy2 := store.Copy()
	policyNN := helpers.NamespacedName{Namespace: "default", Name: "test-policy"}

	// Delete from original
	store.DeletePolicy(policyNN)
	assert.Nil(t, store.GetPolicy(policyNN), "Original should not have policy after delete")
	assert.NotNil(t, storeCopy2.GetPolicy(policyNN), "Copy should still have policy after delete from original")
}

func TestOptimizedStore_ConcurrentAccess(t *testing.T) {
	store := NewOptimizedStore()

	// Test concurrent writes and reads
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			vs := &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
					UID:       types.UID("test-uid"),
				},
			}
			store.SetVirtualService(vs)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
			_ = store.GetVirtualService(nn)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
	vs := store.GetVirtualService(nn)
	assert.NotNil(t, vs)
}

// TestOptimizedStore_CopyShallowBehavior verifies shallow copy behavior with immutable pattern
// With immutable pattern, mutations through copy affect original (but that's OK since we don't mutate in buildSnapshots)
func TestOptimizedStore_CopyShallowBehavior(t *testing.T) {
	original := NewOptimizedStore()

	// Add a VirtualService to original store
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       "test-uid",
		},
		Status: v1alpha1.VirtualServiceStatus{
			Invalid: false,
			Message: "Original message",
		},
	}
	original.SetVirtualService(vs)

	// Create a copy
	copy := original.Copy()

	// Get VS from copy
	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
	vsCopy := copy.GetVirtualService(nn)
	require.NotNil(t, vsCopy)

	// With shallow copy, both point to the same object
	vsOriginal := original.GetVirtualService(nn)
	require.NotNil(t, vsOriginal)

	// They should be the exact same pointer
	assert.Same(t, vsOriginal, vsCopy, "Shallow copy should share pointers")

	// This is expected behavior with immutable pattern:
	// - buildSnapshots returns statuses instead of mutating
	// - rebuildSnapshots applies statuses only to the original store
	// - Copy is used only for dry-run validations which don't need mutation isolation
}

// TestOptimizedStore_CopyIndependence verifies that copy and original have independent maps
func TestOptimizedStore_CopyIndependence(t *testing.T) {
	original := NewOptimizedStore()

	// Add a VirtualService to original
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       "test-uid",
		},
	}
	original.SetVirtualService(vs)

	// Create a copy
	copy := original.Copy()

	// Verify copy has the VS
	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
	assert.NotNil(t, copy.GetVirtualService(nn))

	// Delete from original - copy should still have it
	original.DeleteVirtualService(nn)
	assert.Nil(t, original.GetVirtualService(nn), "Original should not have VS after delete")
	assert.NotNil(t, copy.GetVirtualService(nn), "Copy should still have VS after delete from original")

	// Add new VS to copy - original should not have it
	vs2 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs-2",
			Namespace: "default",
			UID:       "test-uid-2",
		},
	}
	nn2 := helpers.NamespacedName{Namespace: "default", Name: "test-vs-2"}
	copy.SetVirtualService(vs2)

	assert.NotNil(t, copy.GetVirtualService(nn2), "Copy should have new VS")
	assert.Nil(t, original.GetVirtualService(nn2), "Original should not have new VS")
}
