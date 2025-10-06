package store

import (
	"sync"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestOptimizedStore_FirstTimeVSCreation simulates the exact flow when VS is created for the first time
// This tests the scenario mentioned: "ресурс только создается, в store его нет"
func TestOptimizedStore_FirstTimeVSCreation(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)
	nn := helpers.NamespacedName{Namespace: "default", Name: "new-vs"}

	// Step 1: Verify VS doesn't exist yet
	assert.Nil(t, store.GetVirtualService(nn), "VS should not exist initially")
	assert.Nil(t, store.GetVirtualServiceWithStatus(nn), "VS with status should not exist initially")

	// Step 2: Controller calls ApplyVirtualService
	// This is what happens in controller: store.SetVirtualService(vs)
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-vs",
			Namespace: "default",
			UID:       types.UID("new-uid"),
		},
	}
	store.SetVirtualService(vs)

	// Step 3: Verify VS is now in store
	vsFromStore := store.GetVirtualService(nn)
	require.NotNil(t, vsFromStore, "VS should exist after SetVirtualService")
	assert.Equal(t, "new-vs", vsFromStore.Name)

	// Step 4: rebuildSnapshots() sets status
	// This is what happens in rebuildSnapshots after processing
	store.SetVirtualServiceStatus(nn, false, "")

	// Step 5: Controller calls GetVirtualServiceWithStatus
	// This is what happens at line 76 of virtualservice_controller.go
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus, "GetVirtualServiceWithStatus should return VS")
	assert.Equal(t, "new-vs", vsWithStatus.Name)
	assert.False(t, vsWithStatus.Status.Invalid)
	assert.Equal(t, "", vsWithStatus.Status.Message)
}

// TestOptimizedStore_FirstTimeVSCreationWithError simulates VS creation that fails validation
func TestOptimizedStore_FirstTimeVSCreationWithError(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)
	nn := helpers.NamespacedName{Namespace: "default", Name: "invalid-vs"}

	// Step 1: VS doesn't exist
	assert.Nil(t, store.GetVirtualService(nn))

	// Step 2: VS created
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-vs",
			Namespace: "default",
			UID:       types.UID("invalid-uid"),
		},
	}
	store.SetVirtualService(vs)

	// Step 3: rebuildSnapshots() finds validation error
	store.SetVirtualServiceStatus(nn, true, "invalid configuration: missing listener")

	// Step 4: Controller gets VS with error status
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus, "VS should exist even with error")
	assert.True(t, vsWithStatus.Status.Invalid)
	assert.Equal(t, "invalid configuration: missing listener", vsWithStatus.Status.Message)

	// Step 5: Verify original VS is not mutated
	vsOriginal := store.GetVirtualService(nn)
	assert.False(t, vsOriginal.Status.Invalid, "Original VS should not have status")
	assert.Equal(t, "", vsOriginal.Status.Message)
}

// TestOptimizedStore_GetVirtualServiceWithStatus_Nil tests nil handling
func TestOptimizedStore_GetVirtualServiceWithStatus_Nil(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)
	nn := helpers.NamespacedName{Namespace: "default", Name: "nonexistent"}

	// Case 1: VS doesn't exist, no status
	vs := store.GetVirtualServiceWithStatus(nn)
	assert.Nil(t, vs, "Should return nil for nonexistent VS")

	// Case 2: Status exists but VS doesn't (orphaned status)
	store.SetVirtualServiceStatus(nn, true, "orphaned error")
	vs = store.GetVirtualServiceWithStatus(nn)
	assert.Nil(t, vs, "Should return nil when VS doesn't exist even if status exists")
}

// TestOptimizedStore_GetVirtualServiceWithStatus_ZeroStatus tests VS without status
func TestOptimizedStore_GetVirtualServiceWithStatus_ZeroStatus(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs-no-status",
			Namespace: "default",
			UID:       types.UID("uid-123"),
		},
	}
	nn := helpers.NamespacedName{Namespace: "default", Name: "vs-no-status"}

	store.SetVirtualService(vs)

	// Don't set any status - should get zero value (Invalid=false, Message="")
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus)
	assert.False(t, vsWithStatus.Status.Invalid, "Should have zero status")
	assert.Equal(t, "", vsWithStatus.Status.Message, "Should have empty message")
}

// TestOptimizedStore_ControllerFlowSimulation simulates exact controller flow
func TestOptimizedStore_ControllerFlowSimulation(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	// === Reconcile iteration 1: Create VS ===

	// Controller receives CREATE event from Kubernetes
	vsFromK8s := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-service",
			Namespace: "production",
			UID:       types.UID("vs-12345"),
		},
		Spec: v1alpha1.VirtualServiceSpec{
			// ... spec fields
		},
		Status: v1alpha1.VirtualServiceStatus{
			// K8s CR initially has no status
			Invalid: false,
			Message: "",
		},
	}

	nn := helpers.NamespacedName{Namespace: "production", Name: "my-service"}

	// Controller calls ApplyVirtualService
	// Inside: store.SetVirtualService(vs) + rebuildSnapshots()
	store.SetVirtualService(vsFromK8s)

	// rebuildSnapshots() processes VS and sets status
	// Simulating successful validation
	store.SetVirtualServiceStatus(nn, false, "")

	// Controller gets status to sync back to K8s
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus, "VS should exist after creation")

	// prevStatus.Invalid != vs.Status.Invalid check
	// In this case both are false, so no update needed

	// === Reconcile iteration 2: VS becomes invalid ===

	// Something changes (e.g., referenced listener deleted)
	// rebuildSnapshots() runs again and finds error
	store.SetVirtualServiceStatus(nn, true, "referenced listener not found")

	// Controller detects change and syncs to K8s
	vsWithStatus = store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus)
	assert.True(t, vsWithStatus.Status.Invalid)
	assert.Equal(t, "referenced listener not found", vsWithStatus.Status.Message)

	// Verify original VS is still clean
	vsOriginal := store.GetVirtualService(nn)
	assert.False(t, vsOriginal.Status.Invalid, "Original VS not mutated")
}

// TestOptimizedStore_ConcurrentCreationAndStatus tests race during initial creation
func TestOptimizedStore_ConcurrentCreationAndStatus(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	// This tests a potential race:
	// - Goroutine 1: Controller creates VS
	// - Goroutine 2: rebuildSnapshots tries to set status
	// - Goroutine 3: Another controller reads status

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "concurrent-vs",
			Namespace: "default",
			UID:       types.UID("concurrent-uid"),
		},
	}
	nn := helpers.NamespacedName{Namespace: "default", Name: "concurrent-vs"}

	// All operations should be safe
	var wg sync.WaitGroup

	// Goroutine 1: Create VS
	wg.Add(1)
	go func() {
		defer wg.Done()
		store.SetVirtualService(vs)
	}()

	// Goroutine 2: Set status
	wg.Add(1)
	go func() {
		defer wg.Done()
		store.SetVirtualServiceStatus(nn, false, "validated")
	}()

	// Goroutine 3: Read with status
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = store.GetVirtualServiceWithStatus(nn)
	}()

	// Goroutine 4: Read without status
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = store.GetVirtualService(nn)
	}()

	wg.Wait()

	// After all operations, should be consistent
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus, "VS should exist after concurrent operations")
}

// TestOptimizedStore_StatusBeforeVS tests orphaned status scenario
func TestOptimizedStore_StatusBeforeVS(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)
	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	// Set status before VS exists (shouldn't happen in practice, but let's test)
	store.SetVirtualServiceStatus(nn, true, "premature error")

	// Verify GetVirtualService returns nil
	assert.Nil(t, store.GetVirtualService(nn))

	// Verify GetVirtualServiceWithStatus returns nil (VS doesn't exist)
	assert.Nil(t, store.GetVirtualServiceWithStatus(nn))

	// Now add VS
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
	}
	store.SetVirtualService(vs)

	// Now GetVirtualServiceWithStatus should work and use the previously set status
	vsWithStatus := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus)
	assert.True(t, vsWithStatus.Status.Invalid, "Should use previously set status")
	assert.Equal(t, "premature error", vsWithStatus.Status.Message)
}

// TestOptimizedStore_UpdateExistingVSStatus tests updating status of existing VS
func TestOptimizedStore_UpdateExistingVSStatus(t *testing.T) {
	store := NewOptimizedStore().(*OptimizedStore)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-vs",
			Namespace: "default",
			UID:       types.UID("existing-uid"),
		},
	}
	nn := helpers.NamespacedName{Namespace: "default", Name: "existing-vs"}

	// Create VS with initial status
	store.SetVirtualService(vs)
	store.SetVirtualServiceStatus(nn, false, "initial: ok")

	vsWithStatus1 := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus1)
	assert.False(t, vsWithStatus1.Status.Invalid)
	assert.Equal(t, "initial: ok", vsWithStatus1.Status.Message)

	// Update status
	store.SetVirtualServiceStatus(nn, true, "updated: error found")

	vsWithStatus2 := store.GetVirtualServiceWithStatus(nn)
	require.NotNil(t, vsWithStatus2)
	assert.True(t, vsWithStatus2.Status.Invalid)
	assert.Equal(t, "updated: error found", vsWithStatus2.Status.Message)

	// Verify original VS is still clean
	vsOriginal := store.GetVirtualService(nn)
	assert.False(t, vsOriginal.Status.Invalid)
	assert.Equal(t, "", vsOriginal.Status.Message)
}
