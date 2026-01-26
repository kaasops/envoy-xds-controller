package updater

import (
	"context"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mockStore wraps a real store but tracks method calls
type mockStore struct {
	store.Store
	setVSCalls       int
	setVSTCalls      int
	setVSStatusCalls int
}

func (m *mockStore) SetVirtualService(vs *v1alpha1.VirtualService) {
	m.setVSCalls++
	m.Store.SetVirtualService(vs)
}

func (m *mockStore) SetVirtualServiceTemplate(vst *v1alpha1.VirtualServiceTemplate) {
	m.setVSTCalls++
	m.Store.SetVirtualServiceTemplate(vst)
}

// SetVirtualServiceStatus implements the status-only update interface
// This prevents rebuildSnapshots from calling SetVirtualService for status updates
func (m *mockStore) SetVirtualServiceStatus(nn helpers.NamespacedName, invalid bool, message string) {
	m.setVSStatusCalls++
	// Delegate to real store if it supports this method
	if statusStore, ok := m.Store.(interface {
		SetVirtualServiceStatus(helpers.NamespacedName, bool, string)
	}); ok {
		statusStore.SetVirtualServiceStatus(nn, invalid, message)
	}
}

func TestApplyVirtualService_SkipsRebuildWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	ms := &mockStore{Store: realStore}
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, ms)

	// Create a VirtualService
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "test-node"},
		},
		Spec: v1alpha1.VirtualServiceSpec{},
	}

	// First apply - should call SetVirtualService
	updater.ApplyVirtualService(ctx, vs)
	if ms.setVSCalls != 1 {
		t.Errorf("Expected 1 SetVirtualService call after first apply, got %d", ms.setVSCalls)
	}

	// Second apply with same data - should NOT call SetVirtualService
	vs2 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "test-node"},
		},
		Spec: v1alpha1.VirtualServiceSpec{},
	}
	updater.ApplyVirtualService(ctx, vs2)
	if ms.setVSCalls != 1 {
		t.Errorf("Expected still 1 SetVirtualService call after second apply (unchanged), got %d", ms.setVSCalls)
	}

	// Third apply with different node IDs - should call SetVirtualService
	vs3 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "different-node"},
		},
		Spec: v1alpha1.VirtualServiceSpec{},
	}
	updater.ApplyVirtualService(ctx, vs3)
	if ms.setVSCalls != 2 {
		t.Errorf("Expected 2 SetVirtualService calls after third apply (changed), got %d", ms.setVSCalls)
	}
}

func TestApplyVirtualService_HandlesNewVirtualService(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	ms := &mockStore{Store: realStore}
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, ms)

	// Apply new VirtualService - should call SetVirtualService
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "new-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "node1"},
		},
	}

	updater.ApplyVirtualService(ctx, vs)
	if ms.setVSCalls != 1 {
		t.Errorf("Expected 1 SetVirtualService call for new VS, got %d", ms.setVSCalls)
	}

	// Verify VS is in store
	storedVS := realStore.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "new-vs"})
	if storedVS == nil {
		t.Error("Expected VS to be stored")
	}
}

func TestApplyVirtualServiceTemplate_SkipsRebuildWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	ms := &mockStore{Store: realStore}
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, ms)

	// Create a VirtualServiceTemplate
	vst := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{},
	}

	// First apply - should call SetVirtualServiceTemplate
	updater.ApplyVirtualServiceTemplate(ctx, vst)
	if ms.setVSTCalls != 1 {
		t.Errorf("Expected 1 SetVirtualServiceTemplate call after first apply, got %d", ms.setVSTCalls)
	}

	// Second apply with same data - should NOT call SetVirtualServiceTemplate
	vst2 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{},
	}
	updater.ApplyVirtualServiceTemplate(ctx, vst2)
	if ms.setVSTCalls != 1 {
		t.Errorf("Expected still 1 SetVirtualServiceTemplate call after second apply (unchanged), got %d", ms.setVSTCalls)
	}

	// Third apply with different ExtraFields - should call SetVirtualServiceTemplate
	vst3 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{
				{Name: "newField", Type: "string"},
			},
		},
	}
	updater.ApplyVirtualServiceTemplate(ctx, vst3)
	if ms.setVSTCalls != 2 {
		t.Errorf("Expected 2 SetVirtualServiceTemplate calls after third apply (changed), got %d", ms.setVSTCalls)
	}
}

func TestApplyVirtualServiceTemplate_ReturnsCorrectUpdateFlag(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, realStore)

	// Create initial VST
	vst := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{},
	}

	// First apply - should return false (new, not update)
	isUpdate := updater.ApplyVirtualServiceTemplate(ctx, vst)
	if isUpdate {
		t.Error("Expected isUpdate=false for new VST")
	}

	// Second apply unchanged - should return false (no change)
	vst2 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{},
	}
	isUpdate = updater.ApplyVirtualServiceTemplate(ctx, vst2)
	if isUpdate {
		t.Error("Expected isUpdate=false for unchanged VST")
	}

	// Third apply with changes - should return true (actual update)
	vst3 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{
				{Name: "field1", Type: "string"},
			},
		},
	}
	isUpdate = updater.ApplyVirtualServiceTemplate(ctx, vst3)
	if !isUpdate {
		t.Error("Expected isUpdate=true for changed VST")
	}
}

// TestApplyVirtualService_StoreContainsCorrectData verifies that the store
// contains the correct data after apply operations
func TestApplyVirtualService_StoreContainsCorrectData(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, realStore)

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vs"}

	// Apply initial VS
	vs1 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "node1"},
		},
	}
	updater.ApplyVirtualService(ctx, vs1)

	// Verify stored data
	stored := realStore.GetVirtualService(nn)
	if stored == nil {
		t.Fatal("VS should be in store after first apply")
	}
	if stored.Annotations[v1alpha1.AnnotationNodeIDs] != "node1" {
		t.Errorf("Expected nodeIDs 'node1', got '%s'", stored.Annotations[v1alpha1.AnnotationNodeIDs])
	}

	// Apply unchanged - store should still have same data
	vs2 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "node1"},
		},
	}
	updater.ApplyVirtualService(ctx, vs2)

	stored = realStore.GetVirtualService(nn)
	if stored == nil {
		t.Fatal("VS should still be in store after unchanged apply")
	}

	// Apply with changes - store should have updated data
	vs3 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "node2"},
		},
	}
	updater.ApplyVirtualService(ctx, vs3)

	stored = realStore.GetVirtualService(nn)
	if stored == nil {
		t.Fatal("VS should be in store after changed apply")
	}
	if stored.Annotations[v1alpha1.AnnotationNodeIDs] != "node2" {
		t.Errorf("Expected nodeIDs 'node2' after update, got '%s'", stored.Annotations[v1alpha1.AnnotationNodeIDs])
	}
}

// TestApplyVirtualServiceTemplate_StoreContainsCorrectData verifies that the store
// contains the correct data after apply operations
func TestApplyVirtualServiceTemplate_StoreContainsCorrectData(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, realStore)

	nn := helpers.NamespacedName{Namespace: "default", Name: "test-vst"}

	// Apply initial VST
	vst1 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{},
	}
	updater.ApplyVirtualServiceTemplate(ctx, vst1)

	// Verify stored data
	stored := realStore.GetVirtualServiceTemplate(nn)
	if stored == nil {
		t.Fatal("VST should be in store after first apply")
	}
	if len(stored.Spec.ExtraFields) != 0 {
		t.Errorf("Expected no ExtraFields, got %d", len(stored.Spec.ExtraFields))
	}

	// Apply unchanged - store should still have same data
	vst2 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{},
	}
	updater.ApplyVirtualServiceTemplate(ctx, vst2)

	stored = realStore.GetVirtualServiceTemplate(nn)
	if stored == nil {
		t.Fatal("VST should still be in store after unchanged apply")
	}

	// Apply with changes - store should have updated data
	vst3 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vst",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{
				{Name: "field1", Type: "string", Required: true},
			},
		},
	}
	updater.ApplyVirtualServiceTemplate(ctx, vst3)

	stored = realStore.GetVirtualServiceTemplate(nn)
	if stored == nil {
		t.Fatal("VST should be in store after changed apply")
	}
	if len(stored.Spec.ExtraFields) != 1 {
		t.Fatalf("Expected 1 ExtraField after update, got %d", len(stored.Spec.ExtraFields))
	}
	if stored.Spec.ExtraFields[0].Name != "field1" {
		t.Errorf("Expected ExtraField name 'field1', got '%s'", stored.Spec.ExtraFields[0].Name)
	}
}

// TestApplyVirtualServiceTemplate_MultipleUpdates verifies correct behavior
// across multiple sequential updates
func TestApplyVirtualServiceTemplate_MultipleUpdates(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	ms := &mockStore{Store: realStore}
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, ms)

	// Sequence: new -> unchanged -> changed -> unchanged -> changed
	// Expected: call, skip, call, skip, call
	// isUpdate: false, false, true, false, true

	vst := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec:       v1alpha1.VirtualServiceTemplateSpec{},
	}

	// 1. New VST
	isUpdate := updater.ApplyVirtualServiceTemplate(ctx, vst)
	if isUpdate {
		t.Error("Step 1: Expected isUpdate=false for new VST")
	}
	if ms.setVSTCalls != 1 {
		t.Errorf("Step 1: Expected 1 SetVST call, got %d", ms.setVSTCalls)
	}

	// 2. Unchanged
	vst2 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec:       v1alpha1.VirtualServiceTemplateSpec{},
	}
	isUpdate = updater.ApplyVirtualServiceTemplate(ctx, vst2)
	if isUpdate {
		t.Error("Step 2: Expected isUpdate=false for unchanged VST")
	}
	if ms.setVSTCalls != 1 {
		t.Errorf("Step 2: Expected still 1 SetVST call, got %d", ms.setVSTCalls)
	}

	// 3. Changed (add field)
	vst3 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{{Name: "f1", Type: "string"}},
		},
	}
	isUpdate = updater.ApplyVirtualServiceTemplate(ctx, vst3)
	if !isUpdate {
		t.Error("Step 3: Expected isUpdate=true for changed VST")
	}
	if ms.setVSTCalls != 2 {
		t.Errorf("Step 3: Expected 2 SetVST calls, got %d", ms.setVSTCalls)
	}

	// 4. Unchanged (same as step 3)
	vst4 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{{Name: "f1", Type: "string"}},
		},
	}
	isUpdate = updater.ApplyVirtualServiceTemplate(ctx, vst4)
	if isUpdate {
		t.Error("Step 4: Expected isUpdate=false for unchanged VST")
	}
	if ms.setVSTCalls != 2 {
		t.Errorf("Step 4: Expected still 2 SetVST calls, got %d", ms.setVSTCalls)
	}

	// 5. Changed (modify field)
	vst5 := &v1alpha1.VirtualServiceTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{{Name: "f1", Type: "enum", Enum: []string{"a", "b"}}},
		},
	}
	isUpdate = updater.ApplyVirtualServiceTemplate(ctx, vst5)
	if !isUpdate {
		t.Error("Step 5: Expected isUpdate=true for changed VST")
	}
	if ms.setVSTCalls != 3 {
		t.Errorf("Step 5: Expected 3 SetVST calls, got %d", ms.setVSTCalls)
	}
}

// TestApplyVirtualService_NilAnnotationsHandling verifies correct handling
// of nil vs empty annotations
func TestApplyVirtualService_NilAnnotationsHandling(t *testing.T) {
	ctx := context.Background()
	realStore := store.New()
	ms := &mockStore{Store: realStore}
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, ms)

	// Apply VS with nil annotations
	vs1 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: nil,
		},
	}
	updater.ApplyVirtualService(ctx, vs1)
	if ms.setVSCalls != 1 {
		t.Errorf("Expected 1 SetVS call, got %d", ms.setVSCalls)
	}

	// Apply VS with empty annotations - should be considered equal
	vs2 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}
	updater.ApplyVirtualService(ctx, vs2)
	if ms.setVSCalls != 1 {
		t.Errorf("Expected still 1 SetVS call (nil == empty annotations), got %d", ms.setVSCalls)
	}

	// Apply VS with actual node IDs - should be different
	vs3 := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-vs",
			Namespace:   "default",
			Annotations: map[string]string{v1alpha1.AnnotationNodeIDs: "node1"},
		},
	}
	updater.ApplyVirtualService(ctx, vs3)
	if ms.setVSCalls != 2 {
		t.Errorf("Expected 2 SetVS calls (added nodeIDs), got %d", ms.setVSCalls)
	}
}
