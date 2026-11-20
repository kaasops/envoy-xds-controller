package virtualservice

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
)

type updateFixture struct {
	t    *testing.T
	kube client.WithWatch
	vs   *api.VirtualService
	s    store.Store
}

func newUpdateFixture(t *testing.T) *updateFixture {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, api.AddToScheme(scheme))
	kube := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&api.VirtualService{}).Build()

	vs := &api.VirtualService{ObjectMeta: metav1.ObjectMeta{Name: "shop", Namespace: "edge", UID: "vs-uid"}}
	vs.SetEditable(true)
	vs.SetNodeIDs([]string{"old-node"})
	vs.SetDescription("previous description")
	vs.Spec.Template = &api.ResourceRef{Name: "http"}
	require.NoError(t, kube.Create(t.Context(), vs))

	s := store.New()
	s.SetVirtualService(vs.DeepCopy())
	s.SetListener(&api.Listener{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: vs.Namespace},
		Spec:       &runtime.RawExtension{Raw: []byte(`{"address":{"socket_address":{"address":"127.0.0.1","port_value":8080}}}`)},
	})
	template := &api.VirtualServiceTemplate{ObjectMeta: metav1.ObjectMeta{Name: "http", Namespace: vs.Namespace, UID: "template-uid"}}
	template.Spec.Listener = &api.ResourceRef{Name: "web"}
	template.Spec.HTTPFilters = []*runtime.RawExtension{{Raw: []byte(`{"name":"envoy.filters.http.router","typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}}`)}}
	template.Spec.VirtualHost = &runtime.RawExtension{Raw: []byte(`{"domains":["shop.example.test"],"routes":[{"match":{"prefix":"/"},"direct_response":{"status":200}}]}`)}
	s.SetVirtualServiceTemplate(template)
	return &updateFixture{t: t, kube: kube, vs: vs, s: s}
}

func (f *updateFixture) update(writer client.Writer) error {
	service := NewVirtualServiceStore(f.s, writer, f.kube, f.vs.Namespace, nil)
	_, err := service.UpdateVirtualService(f.t.Context(), connect.NewRequest(&v1.UpdateVirtualServiceRequest{
		Uid: string(f.vs.UID), TemplateUid: "template-uid", NodeIds: []string{"next-node"}, Description: "updated",
	}))
	return err
}

func (f *updateFixture) current() *api.VirtualService {
	f.t.Helper()
	var vs api.VirtualService
	require.NoError(f.t, f.kube.Get(f.t.Context(), client.ObjectKeyFromObject(f.vs), &vs))
	return &vs
}

// Run another writer after the handler's GET and just before the first Update reaches the API.
func (f *updateFixture) updateAfter(change func(*api.VirtualService)) error {
	f.t.Helper()
	first := true
	writer := interceptor.NewClient(f.kube, interceptor.Funcs{
		Update: func(ctx context.Context, kube client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if !first {
				return kube.Update(ctx, obj, opts...)
			}
			first = false
			change(f.current())
			err := kube.Update(ctx, obj, opts...)
			require.True(f.t, apierrors.IsConflict(err), "expected resourceVersion conflict, got %v", err)
			return err
		},
	})
	err := f.update(writer)
	require.False(f.t, first, "handler did not reach the simulated concurrent write")
	return err
}

func TestUpdateVirtualServiceAfterStatusWrite(t *testing.T) {
	for _, tt := range []struct {
		name         string
		editLiveSpec bool
	}{
		{name: "only status advanced"},
		{name: "retain latest fields omitted from the request", editLiveSpec: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newUpdateFixture(t)
			vs := f.current()
			if tt.editLiveSpec {
				enabled := true
				vs.Spec.UseRemoteAddress = &enabled
				vs.Annotations["example.test/owner"] = "routing"
				require.NoError(t, f.kube.Update(t.Context(), vs))
			}
			vs.Status = api.VirtualServiceStatus{ObservedGeneration: 9, Invalid: true, Message: "publication held"}
			require.NoError(t, f.kube.Status().Update(t.Context(), vs))
			require.NotEqual(t, vs.ResourceVersion, f.vs.ResourceVersion)
			require.NoError(t, f.update(f.kube))

			got := f.current()
			require.Equal(t, []string{"next-node"}, got.GetNodeIDs())
			require.Equal(t, "updated", got.GetDescription())
			require.Equal(t, vs.Status, got.Status)
			require.Equal(t, vs.Spec.UseRemoteAddress, got.Spec.UseRemoteAddress)
			require.Equal(t, vs.Annotations["example.test/owner"], got.Annotations["example.test/owner"])
		})
	}
}

func TestUpdateVirtualServiceRetriesConcurrentStatus(t *testing.T) {
	f := newUpdateFixture(t)
	status := api.VirtualServiceStatus{Invalid: true, Message: "publication held"}
	err := f.updateAfter(func(vs *api.VirtualService) {
		vs.Status = status
		require.NoError(t, f.kube.Status().Update(t.Context(), vs))
	})
	require.NoError(t, err)
	got := f.current()
	require.Equal(t, status, got.Status)
	require.Equal(t, []string{"next-node"}, got.GetNodeIDs())
	require.Equal(t, "updated", got.GetDescription())
}

func TestUpdateVirtualServicePreservesConcurrentEdit(t *testing.T) {
	for _, field := range []string{"spec", "annotations"} {
		t.Run(field, func(t *testing.T) {
			f := newUpdateFixture(t)
			var winner *api.VirtualService
			err := f.updateAfter(func(vs *api.VirtualService) {
				if field == "spec" {
					enabled := true
					vs.Spec.UseRemoteAddress = &enabled
					vs.Generation++ // The fake client does not advance the generation on spec writes.
				} else {
					vs.SetDescription("the other user's edit")
				}
				require.NoError(t, f.kube.Update(t.Context(), vs))
				winner = f.current()
			})
			require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
			require.Equal(t, winner, f.current())
		})
	}
}

func TestUpdateVirtualServicePreservesReplacement(t *testing.T) {
	f := newUpdateFixture(t)
	var replacement *api.VirtualService
	err := f.updateAfter(func(vs *api.VirtualService) {
		require.NoError(t, f.kube.Delete(t.Context(), vs))
		vs.ResourceVersion = ""
		vs.UID = "replacement-uid"
		vs.SetDescription("replacement service")
		require.NoError(t, f.kube.Create(t.Context(), vs))
		// The fake starts recreated objects at version "1". Give the replacement a distinct version.
		vs.Status.Message = "replacement published"
		require.NoError(t, f.kube.Status().Update(t.Context(), vs))
		replacement = f.current()
	})
	require.Error(t, err)
	require.Equal(t, replacement, f.current())
}
