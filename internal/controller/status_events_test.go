package controller

import (
	"testing"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestVirtualServiceStatusWritesDoNotWakeOtherWriters(t *testing.T) {
	before := &api.VirtualService{ObjectMeta: metav1.ObjectMeta{UID: "one", Generation: 4, ResourceVersion: "17"}}
	after := before.DeepCopy()
	after.ResourceVersion = "18"
	after.Status.Message = "another replica reported its local publication time"
	update := event.UpdateEvent{ObjectOld: before, ObjectNew: after}
	require.False(t, virtualServiceInputChanged(update))
	after.Annotations = map[string]string{api.AnnotationNodeIDs: "two"}
	require.True(t, virtualServiceInputChanged(update))
	after.Annotations = nil
	after.Generation++
	require.True(t, virtualServiceInputChanged(update))
	after.Generation = before.Generation
	after.UID = "replacement"
	require.True(t, virtualServiceInputChanged(update))
}
