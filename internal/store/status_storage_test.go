package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestStatusStorage_BasicOperations tests basic status storage functionality
func TestStatusStorage_BasicOperations(t *testing.T) {
	storage := NewStatusStorage()

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	// Set status
	storage.SetStatus(nn, true, "error message")

	// Get status
	status := storage.GetStatus(nn)
	assert.True(t, status.Invalid)
	assert.Equal(t, "error message", status.Message)

	// Delete status
	storage.DeleteStatus(nn)
	status = storage.GetStatus(nn)
	assert.False(t, status.Invalid)
	assert.Equal(t, "", status.Message)
}

// TestStatusStorage_ImmutabilityPattern tests that status storage maintains immutability
func TestStatusStorage_ImmutabilityPattern(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	// Add VirtualService
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}
	store.SetVirtualService(vs)

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	// Set status separately
	store.SetVirtualServiceStatus(nn, true, "validation error")

	// Get VS - should NOT have status
	vsFromStore := store.GetVirtualService(nn)
	require.NotNil(t, vsFromStore)
	assert.False(t, vsFromStore.Status.Invalid, "VS object should not have status")
	assert.Equal(t, "", vsFromStore.Status.Message, "VS object should not have status")

	// Get VS with status - should have status
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus)
	assert.True(t, vsWithStatus.Status.Invalid, "VS with status should have status")
	assert.Equal(t, "validation error", vsWithStatus.Status.Message)

	// Verify they are different objects
	assert.NotSame(t, vsFromStore, vsWithStatus, "GetVirtualServiceWithStatus should return a copy")
}

// TestStatusStorage_ConcurrentAccess tests concurrent status updates
func TestStatusStorage_ConcurrentAccess(t *testing.T) {
	storage := NewStatusStorage()

	var wg sync.WaitGroup
	iterations := 1000

	// Goroutine 1: Write statuses
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			nn := helpers.NamespacedName{
				Namespace: "default",
				Name:      fmt.Sprintf("vs-%d", i%10),
			}
			storage.SetStatus(nn, i%2 == 0, fmt.Sprintf("message-%d", i))
		}
	}()

	// Goroutine 2: Read statuses
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			nn := helpers.NamespacedName{
				Namespace: "default",
				Name:      fmt.Sprintf("vs-%d", i%10),
			}
			_ = storage.GetStatus(nn)
		}
	}()

	wg.Wait()
}

// TestStatusStorage_CopyIsolation tests that copied status storage is isolated
func TestStatusStorage_CopyIsolation(t *testing.T) {
	original := NewStatusStorage()

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
	original.SetStatus(nn, false, "initial")

	// Copy
	copy := original.Copy()

	// Verify copy has original value
	assert.Equal(t, "initial", copy.GetStatus(nn).Message)

	// Modify original
	original.SetStatus(nn, true, "modified")

	// Verify copy is not affected
	copyStatus := copy.GetStatus(nn)
	assert.False(t, copyStatus.Invalid, "Copy should not be affected by original modifications")
	assert.Equal(t, "initial", copyStatus.Message)

	// Verify original has new value
	originalStatus := original.GetStatus(nn)
	assert.True(t, originalStatus.Invalid)
	assert.Equal(t, "modified", originalStatus.Message)
}

// TestOptimizedStore_StatusStorageInCopy tests that Copy() includes status storage
func TestOptimizedStore_StatusStorageInCopy(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}
	store.SetVirtualService(vs)

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}
	store.SetVirtualServiceStatus(nn, true, "error in original")

	// Copy store
	storeCopy := store.Copy().(*OptimizedStore)

	// Verify copy has status
	copyStatus := storeCopy.GetVirtualServiceStatus(nn)
	assert.True(t, copyStatus.Invalid)
	assert.Equal(t, "error in original", copyStatus.Message)

	// Modify original status
	store.SetVirtualServiceStatus(nn, false, "fixed in original")

	// Verify copy is not affected
	copyStatus = storeCopy.GetVirtualServiceStatus(nn)
	assert.True(t, copyStatus.Invalid, "Copy status should not be affected")
	assert.Equal(t, "error in original", copyStatus.Message)

	// Verify original has new status
	originalStatus := store.GetVirtualServiceStatus(nn)
	assert.False(t, originalStatus.Invalid)
	assert.Equal(t, "fixed in original", originalStatus.Message)
}

// TestOptimizedStore_NoMutationWithStatusStorage tests that VS objects are never mutated
func TestOptimizedStore_NoMutationWithStatusStorage(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Status: v1alpha1.VirtualServiceStatus{
			Invalid: false,
			Message: "initial",
		},
	}
	store.SetVirtualService(vs)

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	// Get VS and keep reference
	vsRef1 := store.GetVirtualService(nn)
	require.NotNil(t, vsRef1)
	initialMessage := vsRef1.Status.Message

	// Set status separately
	store.SetVirtualServiceStatus(nn, true, "new error")

	// Get VS again
	vsRef2 := store.GetVirtualService(nn)
	require.NotNil(t, vsRef2)

	// Verify VS object was NOT mutated
	assert.Equal(t, initialMessage, vsRef1.Status.Message, "Original VS object should not be mutated")
	assert.Equal(t, initialMessage, vsRef2.Status.Message, "VS object from store should not have status")

	// But GetVirtualServiceWithStatus should have new status
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus)
	assert.True(t, vsWithStatus.Status.Invalid)
	assert.Equal(t, "new error", vsWithStatus.Status.Message)
}

// TestOptimizedStore_StatusStorageRealWorldFlow simulates real controller flow
func TestOptimizedStore_StatusStorageRealWorldFlow(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	// Setup: Multiple VirtualServices
	for i := 0; i < 20; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("vs-%d", i),
				Namespace: "default",
				UID:       types.UID(fmt.Sprintf("uid-%d", i)),
			},
		}
		store.SetVirtualService(vs)
	}

	var wg sync.WaitGroup
	corruptionDetected := false
	corruptionMutex := sync.Mutex{}

	// Goroutine 1: DryRun validation (copies store and reads)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			// Copy store (like webhook does)
			storeCopy := store.Copy().(*OptimizedStore)

			// Read VirtualServices from copy
			for j := 0; j < 20; j++ {
				nn := helpers.NamespacedName{
					Namespace: "default",
					Name:      fmt.Sprintf("vs-%d", j),
				}
				vs := storeCopy.GetVirtualService(nn)
				if vs != nil {
					// Check if VS object has status (it shouldn't!)
					if vs.Status.Message != "" && vs.Status.Message != "initial" {
						corruptionMutex.Lock()
						corruptionDetected = true
						corruptionMutex.Unlock()
						t.Logf("Corruption: VS object has status: %s", vs.Status.Message)
					}
				}
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Goroutine 2: Status updates (like rebuildSnapshots does)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			// Update statuses using NEW API (no mutation)
			for j := 0; j < 20; j++ {
				nn := helpers.NamespacedName{
					Namespace: "default",
					Name:      fmt.Sprintf("vs-%d", j),
				}
				// SetVirtualServiceStatus does NOT mutate VS objects
				store.SetVirtualServiceStatus(nn, i%2 == 0, fmt.Sprintf("status-%d", i))
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()

	assert.False(t, corruptionDetected, "VS objects should never be mutated")
}

// TestOptimizedStore_StatusStorageDeleteCleanup tests that status is cleaned up on delete
func TestOptimizedStore_StatusStorageDeleteCleanup(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}
	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	store.SetVirtualService(vs)
	store.SetVirtualServiceStatus(nn, true, "error")

	// Verify status exists
	status := store.GetVirtualServiceStatus(nn)
	assert.True(t, status.Invalid)

	// Delete VS
	store.DeleteVirtualService(nn)

	// Verify status is also deleted
	status = store.GetVirtualServiceStatus(nn)
	assert.False(t, status.Invalid, "Status should be deleted with VS")
	assert.Equal(t, "", status.Message)
}
