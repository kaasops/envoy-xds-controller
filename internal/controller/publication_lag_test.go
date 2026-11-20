package controller

import (
	"context"
	"testing"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPublicationStatusWithLaggingInformer(t *testing.T) {
	ctx := t.Context()
	inputs, err := testStore()
	require.NoError(t, err)
	u := updater.NewCacheUpdater(cache.NewSnapshotCache(), inputs)
	vs := &api.VirtualService{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", UID: "original", Generation: 1,
			Annotations: map[string]string{api.AnnotationNodeIDs: "local"}},
		Spec: api.VirtualServiceSpec{VirtualServiceCommonSpec: api.VirtualServiceCommonSpec{
			Listener: &api.ResourceRef{Name: "http"},
			HTTPFilters: []*runtime.RawExtension{{Raw: []byte(`{"name":"envoy.filters.http.router",
				"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}}`)}},
			VirtualHost: &runtime.RawExtension{Raw: []byte(`{"domains":["app.example"],"routes":[
				{"match":{"prefix":"/"},"direct_response":{"status":200}}]}`)},
		}},
		Status: api.VirtualServiceStatus{ObservedGeneration: 1},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, api.AddToScheme(scheme))
	live := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(vs).WithObjects(vs).Build()
	key := client.ObjectKeyFromObject(vs)
	var cached api.VirtualService
	require.NoError(t, live.Get(ctx, key, &cached))
	missingFromCache := false
	cachedReads := 0
	stale := interceptor.NewClient(live, interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey,
			obj client.Object, _ ...client.GetOption) error {
			cachedReads++
			if missingFromCache {
				return apierrors.NewNotFound(api.GroupVersion.WithResource("virtualservices").GroupResource(), key.Name)
			}
			cached.DeepCopyInto(obj.(*api.VirtualService))
			return nil
		},
	})
	liveReads := 0
	var afterRead func()
	apiReader := interceptor.NewClient(live, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, k client.ObjectKey,
			obj client.Object, opts ...client.GetOption) error {
			liveReads++
			err := c.Get(ctx, k, obj, opts...)
			if afterRead != nil {
				afterRead()
				afterRead = nil
			}
			return err
		},
	})
	ready := make(chan struct{})
	close(ready)
	r := &VirtualServiceReconciler{Client: stale, Updater: u, CacheReadyChan: ready, apiReader: apiReader}
	request := ctrl.Request{NamespacedName: key}
	reconciles := 0
	run := func() error {
		reconciles++
		_, err := r.Reconcile(ctx, request)
		return err
	}
	reconcile := func() {
		t.Helper()
		require.NoError(t, run())
	}
	current := func() *api.VirtualService {
		t.Helper()
		var obj api.VirtualService
		require.NoError(t, live.Get(ctx, key, &obj))
		return &obj
	}

	reconcile()
	require.False(t, current().Status.Invalid)
	listenerKey := client.ObjectKey{Namespace: "default", Name: "http"}
	listener := inputs.GetListener(helpers.NamespacedName(listenerKey)).DeepCopy()
	u.DeleteListener(ctx, listenerKey)
	reconcile()
	require.True(t, current().Status.Invalid)
	u.ApplyListener(ctx, listener)
	reconcile()
	require.False(t, current().Status.Invalid, "after recovery the API must contain A while the informer still shows A")

	// A competing status writer wins between the authoritative GET and Update.
	afterRead = func() {
		obj := current()
		obj.Status.UsedSecrets = []api.ResourceRef{{Name: "another-writer"}}
		require.NoError(t, live.Status().Update(ctx, obj))
	}
	u.DeleteListener(ctx, listenerKey)
	err = run()
	require.True(t, apierrors.IsConflict(err), "got %v", err)
	reconcile()
	require.True(t, current().Status.Invalid)
	require.Equal(t, []api.ResourceRef{{Name: "another-writer"}}, current().Status.UsedSecrets)

	// The recovery of generation 1 must not attest a new spec that the informer has not delivered.
	u.ApplyListener(ctx, listener)
	newer := current()
	newer.Generation++
	newer.Spec.Listener.Name = "missing-v2"
	require.NoError(t, live.Update(ctx, newer))
	reconcile()
	require.True(t, current().Status.Invalid)
	require.Equal(t, newer.Generation, current().Status.ObservedGeneration)
	require.Contains(t, current().Status.Message, "missing-v2")

	// After replacement the informer can still report the old UID, then NotFound before the add.
	require.NoError(t, live.Delete(ctx, current()))
	replacement := vs.DeepCopy()
	replacement.UID = "replacement"
	replacement.ResourceVersion = ""
	require.NoError(t, live.Create(ctx, replacement))
	reconcile()
	missingFromCache = true
	reconcile()
	_, observed := u.PublicationStatusFor(replacement)
	require.True(t, observed)
	require.False(t, current().Status.Invalid)
	require.Equal(t, replacement.Generation, current().Status.ObservedGeneration)
	require.Equal(t, reconciles, liveReads)
	require.Zero(t, cachedReads)
}
