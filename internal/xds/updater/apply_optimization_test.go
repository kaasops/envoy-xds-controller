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
