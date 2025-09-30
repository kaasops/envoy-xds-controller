package store

import (
	"context"
	"fmt"
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

// TestOptimizedStore_DryRunPatterns tests Copy() behavior matching real DryRun usage from updater.go
func TestOptimizedStore_DryRunPatterns(t *testing.T) {
	t.Run("DryBuildSnapshotsWithVirtualService pattern", func(t *testing.T) {
		// Setup: Original store with existing data
		original := NewOptimizedStore()

		vs1 := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-vs",
				Namespace: "default",
				UID:       types.UID("existing-uid"),
			},
		}
		original.SetVirtualService(vs1)

		listener := &v1alpha1.Listener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-listener",
				Namespace: "default",
				UID:       types.UID("listener-uid"),
			},
		}
		original.SetListener(listener)

		// Simulate DryRun: Copy store, add candidate VS, verify isolation
		storeCopy := original.Copy()
		candidateVS := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "candidate-vs",
				Namespace: "default",
				UID:       types.UID("candidate-uid"),
			},
		}
		storeCopy.SetVirtualService(candidateVS)

		// Verify: Original unchanged
		assert.Nil(t, original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "candidate-vs"}),
			"Original store should not have candidate VS")
		assert.Nil(t, original.GetVirtualServiceByUID("candidate-uid"),
			"Original store UID index should not have candidate VS")

		// Verify: Copy has both
		assert.NotNil(t, storeCopy.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "existing-vs"}),
			"Copy should have existing VS")
		assert.NotNil(t, storeCopy.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "candidate-vs"}),
			"Copy should have candidate VS")
		assert.NotNil(t, storeCopy.GetVirtualServiceByUID("candidate-uid"),
			"Copy UID index should have candidate VS")

		// Verify: Listener unaffected
		assert.NotNil(t, original.GetListener(helpers.NamespacedName{Namespace: "default", Name: "test-listener"}),
			"Original should still have listener")
		assert.NotNil(t, storeCopy.GetListener(helpers.NamespacedName{Namespace: "default", Name: "test-listener"}),
			"Copy should have listener")
	})

	t.Run("DryValidateVirtualServiceLight pattern", func(t *testing.T) {
		// Setup: Store with multiple resources
		original := NewOptimizedStore()

		for i := 0; i < 10; i++ {
			vs := &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("vs-%d", i),
					Namespace: "default",
					UID:       types.UID(fmt.Sprintf("uid-%d", i)),
				},
			}
			original.SetVirtualService(vs)
		}

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "default",
				UID:       types.UID("cluster-uid"),
			},
		}
		original.SetCluster(cluster)

		// Simulate lightweight validation: Copy, overlay candidate, read resources
		storeCopy := original.Copy()
		candidateVS := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vs-5", // Update existing
				Namespace: "default",
				UID:       types.UID("uid-5-updated"),
			},
		}
		storeCopy.SetVirtualService(candidateVS)

		// Verify: Original VS-5 unchanged
		origVS5 := original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs-5"})
		assert.Equal(t, types.UID("uid-5"), origVS5.UID, "Original VS-5 UID unchanged")

		// Verify: Copy has updated VS-5
		copyVS5 := storeCopy.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs-5"})
		assert.Equal(t, types.UID("uid-5-updated"), copyVS5.UID, "Copy VS-5 has new UID")

		// Verify: UID indices independent
		assert.NotNil(t, original.GetVirtualServiceByUID("uid-5"), "Original has old UID")
		assert.Nil(t, original.GetVirtualServiceByUID("uid-5-updated"), "Original doesn't have new UID")
		assert.NotNil(t, storeCopy.GetVirtualServiceByUID("uid-5-updated"), "Copy has new UID")

		// Verify: Other resources unaffected
		assert.Equal(t, 10, len(original.MapVirtualServices()), "Original still has 10 VS")
		assert.Equal(t, 10, len(storeCopy.MapVirtualServices()), "Copy still has 10 VS")
		assert.NotNil(t, storeCopy.GetCluster(helpers.NamespacedName{Namespace: "default", Name: "test-cluster"}),
			"Copy has cluster")
	})

	t.Run("DryBuildSnapshotsWithVirtualServiceTemplate pattern", func(t *testing.T) {
		// Setup: Store with templates
		original := NewOptimizedStore()

		existingTemplate := &v1alpha1.VirtualServiceTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-template",
				Namespace: "default",
				UID:       types.UID("template-uid-1"),
			},
		}
		original.SetVirtualServiceTemplate(existingTemplate)

		// Simulate DryRun with template
		storeCopy := original.Copy()
		candidateTemplate := &v1alpha1.VirtualServiceTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "candidate-template",
				Namespace: "default",
				UID:       types.UID("template-uid-2"),
			},
		}
		storeCopy.SetVirtualServiceTemplate(candidateTemplate)

		// Verify isolation
		assert.Nil(t, original.GetVirtualServiceTemplate(helpers.NamespacedName{Namespace: "default", Name: "candidate-template"}),
			"Original doesn't have candidate template")
		assert.NotNil(t, storeCopy.GetVirtualServiceTemplate(helpers.NamespacedName{Namespace: "default", Name: "candidate-template"}),
			"Copy has candidate template")
		assert.NotNil(t, storeCopy.GetVirtualServiceTemplate(helpers.NamespacedName{Namespace: "default", Name: "existing-template"}),
			"Copy has existing template")
	})

	t.Run("excludePreviousVSDomains pattern", func(t *testing.T) {
		// Setup: Store representing current state
		original := NewOptimizedStore()

		currentVS := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-vs",
				Namespace: "default",
				UID:       types.UID("current-uid"),
			},
		}
		original.SetVirtualService(currentVS)

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{"key": []byte("value")},
		}
		original.SetSecret(secret)

		// Simulate: Copy store to analyze previous state
		origStoreCopy := original.Copy()

		// Verify: Can read previous state from copy without affecting original
		prevVS := origStoreCopy.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "my-vs"})
		assert.NotNil(t, prevVS, "Copy has previous VS")
		assert.Equal(t, types.UID("current-uid"), prevVS.UID)

		// Verify: Modifications to copy don't affect original
		origStoreCopy.DeleteVirtualService(helpers.NamespacedName{Namespace: "default", Name: "my-vs"})
		assert.Nil(t, origStoreCopy.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "my-vs"}),
			"Deleted from copy")
		assert.NotNil(t, original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "my-vs"}),
			"Original still has VS")

		// Verify: Secrets accessible in both
		assert.NotNil(t, original.GetSecret(helpers.NamespacedName{Namespace: "default", Name: "my-secret"}))
		assert.NotNil(t, origStoreCopy.GetSecret(helpers.NamespacedName{Namespace: "default", Name: "my-secret"}))
	})
}

// TestOptimizedStore_DryRunAllResourceTypes verifies Copy() isolation for all resource types
func TestOptimizedStore_DryRunAllResourceTypes(t *testing.T) {
	original := NewOptimizedStore()

	// Populate with one of each resource type
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{Name: "vs", Namespace: "default", UID: "vs-uid"},
	}
	original.SetVirtualService(vs)

	listener := &v1alpha1.Listener{
		ObjectMeta: metav1.ObjectMeta{Name: "listener", Namespace: "default", UID: "listener-uid"},
	}
	original.SetListener(listener)

	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default", UID: "cluster-uid"},
	}
	original.SetCluster(cluster)

	route := &v1alpha1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: "route", Namespace: "default", UID: "route-uid"},
	}
	original.SetRoute(route)

	httpFilter := &v1alpha1.HttpFilter{
		ObjectMeta: metav1.ObjectMeta{Name: "filter", Namespace: "default", UID: "filter-uid"},
	}
	original.SetHTTPFilter(httpFilter)

	policy := &v1alpha1.Policy{
		ObjectMeta: metav1.ObjectMeta{Name: "policy", Namespace: "default", UID: "policy-uid"},
	}
	original.SetPolicy(policy)

	accessLog := &v1alpha1.AccessLogConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "accesslog", Namespace: "default", UID: "accesslog-uid"},
	}
	original.SetAccessLog(accessLog)

	tracing := &v1alpha1.Tracing{
		ObjectMeta: metav1.ObjectMeta{Name: "tracing", Namespace: "default", UID: "tracing-uid"},
	}
	original.SetTracing(tracing)

	vst := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "template", Namespace: "default", UID: "template-uid"},
	}
	original.SetVirtualServiceTemplate(vst)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
	}
	original.SetSecret(secret)

	// DryRun: Copy and delete all resources
	storeCopy := original.Copy()
	storeCopy.DeleteVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs"})
	storeCopy.DeleteListener(helpers.NamespacedName{Namespace: "default", Name: "listener"})
	storeCopy.DeleteCluster(helpers.NamespacedName{Namespace: "default", Name: "cluster"})
	storeCopy.DeleteRoute(helpers.NamespacedName{Namespace: "default", Name: "route"})
	storeCopy.DeleteHTTPFilter(helpers.NamespacedName{Namespace: "default", Name: "filter"})
	storeCopy.DeletePolicy(helpers.NamespacedName{Namespace: "default", Name: "policy"})
	storeCopy.DeleteAccessLog(helpers.NamespacedName{Namespace: "default", Name: "accesslog"})
	storeCopy.DeleteTracing(helpers.NamespacedName{Namespace: "default", Name: "tracing"})
	storeCopy.DeleteVirtualServiceTemplate(helpers.NamespacedName{Namespace: "default", Name: "template"})
	storeCopy.DeleteSecret(helpers.NamespacedName{Namespace: "default", Name: "secret"})

	// Verify: Copy is empty
	assert.Equal(t, 0, len(storeCopy.MapVirtualServices()), "Copy has no VS")
	assert.Equal(t, 0, len(storeCopy.MapListeners()), "Copy has no listeners")
	assert.Equal(t, 0, len(storeCopy.MapClusters()), "Copy has no clusters")
	assert.Equal(t, 0, len(storeCopy.MapRoutes()), "Copy has no routes")
	assert.Equal(t, 0, len(storeCopy.MapHTTPFilters()), "Copy has no filters")
	assert.Equal(t, 0, len(storeCopy.MapPolicies()), "Copy has no policies")
	assert.Equal(t, 0, len(storeCopy.MapAccessLogs()), "Copy has no accesslogs")
	assert.Equal(t, 0, len(storeCopy.MapTracings()), "Copy has no tracings")
	assert.Equal(t, 0, len(storeCopy.MapVirtualServiceTemplates()), "Copy has no templates")
	assert.Equal(t, 0, len(storeCopy.MapSecrets()), "Copy has no secrets")

	// Verify: Original unchanged
	assert.NotNil(t, original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs"}))
	assert.NotNil(t, original.GetListener(helpers.NamespacedName{Namespace: "default", Name: "listener"}))
	assert.NotNil(t, original.GetCluster(helpers.NamespacedName{Namespace: "default", Name: "cluster"}))
	assert.NotNil(t, original.GetRoute(helpers.NamespacedName{Namespace: "default", Name: "route"}))
	assert.NotNil(t, original.GetHTTPFilter(helpers.NamespacedName{Namespace: "default", Name: "filter"}))
	assert.NotNil(t, original.GetPolicy(helpers.NamespacedName{Namespace: "default", Name: "policy"}))
	assert.NotNil(t, original.GetAccessLog(helpers.NamespacedName{Namespace: "default", Name: "accesslog"}))
	assert.NotNil(t, original.GetTracing(helpers.NamespacedName{Namespace: "default", Name: "tracing"}))
	assert.NotNil(t, original.GetVirtualServiceTemplate(helpers.NamespacedName{Namespace: "default", Name: "template"}))
	assert.NotNil(t, original.GetSecret(helpers.NamespacedName{Namespace: "default", Name: "secret"}))

	// Verify: UID indices unchanged in original
	assert.NotNil(t, original.GetVirtualServiceByUID("vs-uid"))
	assert.NotNil(t, original.GetListenerByUID("listener-uid"))
	assert.NotNil(t, original.GetVirtualServiceTemplateByUID("template-uid"))
	assert.NotNil(t, original.GetRouteByUID("route-uid"))
	assert.NotNil(t, original.GetHTTPFilterByUID("filter-uid"))
	assert.NotNil(t, original.GetAccessLogByUID("accesslog-uid"))

	// Verify: UID indices empty in copy
	assert.Nil(t, storeCopy.GetVirtualServiceByUID("vs-uid"))
	assert.Nil(t, storeCopy.GetListenerByUID("listener-uid"))
	assert.Nil(t, storeCopy.GetVirtualServiceTemplateByUID("template-uid"))
	assert.Nil(t, storeCopy.GetRouteByUID("route-uid"))
	assert.Nil(t, storeCopy.GetHTTPFilterByUID("filter-uid"))
	assert.Nil(t, storeCopy.GetAccessLogByUID("accesslog-uid"))
}

// TestOptimizedStore_ConcurrentDryRuns simulates concurrent DryRun operations
func TestOptimizedStore_ConcurrentDryRuns(t *testing.T) {
	original := NewOptimizedStore()

	// Setup: Populate with initial data
	for i := 0; i < 50; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("vs-%d", i),
				Namespace: "default",
				UID:       types.UID(fmt.Sprintf("uid-%d", i)),
			},
		}
		original.SetVirtualService(vs)
	}

	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster",
			Namespace: "default",
			UID:       types.UID("cluster-uid"),
		},
	}
	original.SetCluster(cluster)

	// Simulate 10 concurrent DryRun operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			// Each goroutine creates a copy and modifies it
			storeCopy := original.Copy()

			// Add new VS
			candidateVS := &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("candidate-%d", id),
					Namespace: "default",
					UID:       types.UID(fmt.Sprintf("candidate-uid-%d", id)),
				},
			}
			storeCopy.SetVirtualService(candidateVS)

			// Update existing VS
			updateVS := &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vs-10",
					Namespace: "default",
					UID:       types.UID(fmt.Sprintf("updated-uid-%d", id)),
				},
			}
			storeCopy.SetVirtualService(updateVS)

			// Delete some VS
			storeCopy.DeleteVirtualService(helpers.NamespacedName{Namespace: "default", Name: fmt.Sprintf("vs-%d", id)})

			// Read cluster
			assert.NotNil(t, storeCopy.GetCluster(helpers.NamespacedName{Namespace: "default", Name: "cluster"}))

			// Verify candidate exists in this copy
			assert.NotNil(t, storeCopy.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: fmt.Sprintf("candidate-%d", id)}))

			done <- true
		}(i)
	}

	// Wait for all DryRuns to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify: Original unchanged
	assert.Equal(t, 50, len(original.MapVirtualServices()), "Original should still have 50 VS")
	for i := 0; i < 50; i++ {
		assert.NotNil(t, original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: fmt.Sprintf("vs-%d", i)}),
			fmt.Sprintf("Original should have vs-%d", i))
	}

	// Verify: No candidates leaked to original
	for i := 0; i < 10; i++ {
		assert.Nil(t, original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: fmt.Sprintf("candidate-%d", i)}),
			fmt.Sprintf("Original should not have candidate-%d", i))
	}

	// Verify: Original vs-10 unchanged
	vs10 := original.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs-10"})
	assert.Equal(t, types.UID("uid-10"), vs10.UID, "Original vs-10 UID unchanged")

	// Verify: Cluster unchanged
	assert.NotNil(t, original.GetCluster(helpers.NamespacedName{Namespace: "default", Name: "cluster"}))
}

// TestOptimizedStore_DryRunTemplateIndices tests template index isolation in Copy()
func TestOptimizedStore_DryRunTemplateIndices(t *testing.T) {
	original := NewOptimizedStore()

	// Setup: Template and VSes using it
	template := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-template",
			Namespace: "default",
			UID:       types.UID("template-uid"),
		},
	}
	original.SetVirtualServiceTemplate(template)

	vs1 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs-1",
			Namespace: "default",
			UID:       types.UID("vs-1-uid"),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			Template: &v1alpha1.ResourceRef{
				Name: "my-template",
			},
		},
	}
	original.SetVirtualService(vs1)

	vs2 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs-2",
			Namespace: "default",
			UID:       types.UID("vs-2-uid"),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			Template: &v1alpha1.ResourceRef{
				Name: "my-template",
			},
		},
	}
	original.SetVirtualService(vs2)

	// Verify initial state
	templateNN := helpers.NamespacedName{Namespace: "default", Name: "my-template"}
	originalVSList := original.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, originalVSList, 2, "Original has 2 VS for template")

	// DryRun: Copy and add new VS with same template
	storeCopy := original.Copy()
	vs3 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs-3",
			Namespace: "default",
			UID:       types.UID("vs-3-uid"),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			Template: &v1alpha1.ResourceRef{
				Name: "my-template",
			},
		},
	}
	storeCopy.SetVirtualService(vs3)

	// Verify: Copy has 3 VS for template
	copyVSList := storeCopy.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, copyVSList, 3, "Copy has 3 VS for template")
	assert.Contains(t, copyVSList, vs3, "Copy includes new VS")

	// Verify: Original still has 2
	originalVSList = original.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, originalVSList, 2, "Original still has 2 VS for template")
	assert.NotContains(t, originalVSList, vs3, "Original doesn't have new VS")

	// DryRun: Delete VS from copy
	storeCopy.DeleteVirtualService(helpers.NamespacedName{Namespace: "default", Name: "vs-1"})

	// Verify: Copy has 2 VS for template (vs-2 and vs-3)
	copyVSList = storeCopy.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, copyVSList, 2, "Copy has 2 VS after delete")
	vsNames := make([]string, 0, len(copyVSList))
	for _, vs := range copyVSList {
		vsNames = append(vsNames, vs.Name)
	}
	assert.Contains(t, vsNames, "vs-2")
	assert.Contains(t, vsNames, "vs-3")
	assert.NotContains(t, vsNames, "vs-1")

	// Verify: Original still has vs-1 and vs-2
	originalVSList = original.GetVirtualServicesByTemplateNN(templateNN)
	assert.Len(t, originalVSList, 2, "Original still has 2 VS")
	origVSNames := make([]string, 0, len(originalVSList))
	for _, vs := range originalVSList {
		origVSNames = append(origVSNames, vs.Name)
	}
	assert.Contains(t, origVSNames, "vs-1")
	assert.Contains(t, origVSNames, "vs-2")
}

// TestOptimizedStore_DryRunNodeDomainsIndex tests node domains index isolation
func TestOptimizedStore_DryRunNodeDomainsIndex(t *testing.T) {
	original := NewOptimizedStore()

	// Setup: Node domains index
	originalIndex := map[string]map[string]struct{}{
		"node1": {
			"domain1.com": {},
			"domain2.com": {},
		},
		"node2": {
			"domain3.com": {},
		},
	}
	original.ReplaceNodeDomainsIndex(originalIndex)

	// Verify initial state
	domains, missing := original.GetNodeDomainsForNodes([]string{"node1", "node2"})
	assert.Len(t, domains, 2)
	assert.Len(t, missing, 0)

	// DryRun: Copy and modify index
	storeCopy := original.Copy()
	modifiedIndex := map[string]map[string]struct{}{
		"node1": {
			"domain1.com":    {},
			"domain2.com":    {},
			"domain-new.com": {}, // Added
		},
		"node3": { // New node
			"domain4.com": {},
		},
	}
	storeCopy.ReplaceNodeDomainsIndex(modifiedIndex)

	// Verify: Copy has modified index
	copyDomains, copyMissing := storeCopy.GetNodeDomainsForNodes([]string{"node1", "node2", "node3"})
	assert.Len(t, copyDomains, 2, "Copy has node1 and node3")
	assert.Contains(t, copyDomains["node1"], "domain-new.com", "Copy has new domain")
	assert.Contains(t, copyDomains, "node3", "Copy has node3")
	assert.Contains(t, copyMissing, "node2", "Copy doesn't have node2")

	// Verify: Original unchanged
	origDomains, origMissing := original.GetNodeDomainsForNodes([]string{"node1", "node2", "node3"})
	assert.Len(t, origDomains, 2, "Original has node1 and node2")
	assert.NotContains(t, origDomains["node1"], "domain-new.com", "Original doesn't have new domain")
	assert.NotContains(t, origDomains, "node3", "Original doesn't have node3")
	assert.Contains(t, origMissing, "node3", "Original is missing node3")
	assert.NotContains(t, origMissing, "node2", "Original has node2")
}
