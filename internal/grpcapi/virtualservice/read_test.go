package virtualservice

import (
	"testing"

	"connectrpc.com/connect"
	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
	"github.com/stretchr/testify/require"
)

// Status subresource writes reach the Kubernetes informer independently of the
// input store: a status-only watch event does not enqueue a snapshot rebuild.
func TestVirtualServiceReadsFollowKubernetesPublicationStatus(t *testing.T) {
	f := newUpdateFixture(t)
	svc := NewVirtualServiceStore(f.s, f.kube, nil, "", nil)
	// A snapshot build may have diagnosed a deprecated field on the input copy.
	indexed := f.s.GetVirtualServiceByUID(string(f.vs.UID)).DeepCopy()
	indexed.Status.Message = "accessLog is deprecated, use accessLogs instead"
	f.s.SetVirtualService(indexed)

	for _, status := range []api.VirtualServiceStatus{
		{Invalid: true, Message: "listener missing"},
		{Invalid: false, Message: "SDS secret skipped"},
		{Invalid: false},
	} {
		obj := f.current()
		obj.Status = status
		require.NoError(t, f.kube.Status().Update(t.Context(), obj))

		list, err := svc.ListVirtualServices(t.Context(), connect.NewRequest(&v1.ListVirtualServicesRequest{
			AccessGroup: f.vs.GetAccessGroup(),
		}))
		require.NoError(t, err)
		require.Len(t, list.Msg.Items, 1)
		item := list.Msg.Items[0]
		require.Equal(t, string(obj.UID), item.Uid)
		require.Equal(t, status.Invalid, item.Status.Invalid)
		require.Equal(t, status.Message, item.Status.Message)

		detail, err := svc.GetVirtualService(t.Context(), connect.NewRequest(&v1.GetVirtualServiceRequest{Uid: item.Uid}))
		require.NoError(t, err)
		require.Equal(t, status.Invalid, detail.Msg.Status.Invalid)
		require.Equal(t, status.Message, detail.Msg.Status.Message)
		require.Equal(t, obj.ResourceVersion, f.current().ResourceVersion)
		require.Equal(t, obj.Spec, f.current().Spec)
	}
}

func TestGetVirtualServiceRejectsReplacedUID(t *testing.T) {
	f := newUpdateFixture(t)
	require.NoError(t, f.kube.Delete(t.Context(), f.current()))
	replacement := f.vs.DeepCopy()
	replacement.ResourceVersion = ""
	replacement.UID = "replacement"
	require.NoError(t, f.kube.Create(t.Context(), replacement))
	svc := NewVirtualServiceStore(f.s, f.kube, nil, "", nil)
	_, err := svc.GetVirtualService(t.Context(), connect.NewRequest(&v1.GetVirtualServiceRequest{Uid: string(f.vs.UID)}))
	require.Error(t, err)
	got, err := svc.GetVirtualService(t.Context(), connect.NewRequest(&v1.GetVirtualServiceRequest{Uid: string(replacement.UID)}))
	require.NoError(t, err)
	require.Equal(t, string(replacement.UID), got.Msg.Uid)
}

func TestReadVirtualServiceWithMissingReferences(t *testing.T) {
	f := newUpdateFixture(t)
	obj := f.current()
	missing := &api.ResourceRef{Name: "missing"}
	obj.Spec.Template = missing
	obj.Spec.Listener = missing
	obj.Spec.AccessLogConfigs = []*api.ResourceRef{missing}
	obj.Spec.AdditionalRoutes = []*api.ResourceRef{missing}
	obj.Spec.AdditionalHttpFilters = []*api.ResourceRef{missing}
	require.NoError(t, f.kube.Update(t.Context(), obj))
	obj.Status = api.VirtualServiceStatus{Invalid: true, Message: "template missing"}
	require.NoError(t, f.kube.Status().Update(t.Context(), obj))
	svc := NewVirtualServiceStore(f.s, f.kube, nil, "", nil)

	other, err := svc.ListVirtualServices(t.Context(), connect.NewRequest(&v1.ListVirtualServicesRequest{
		AccessGroup: "other",
	}))
	require.NoError(t, err)
	require.Empty(t, other.Msg.Items)
	list, err := svc.ListVirtualServices(t.Context(), connect.NewRequest(&v1.ListVirtualServicesRequest{
		AccessGroup: obj.GetAccessGroup(),
	}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Items, 1)
	require.Equal(t, "missing", list.Msg.Items[0].Template.Name)
	require.Empty(t, list.Msg.Items[0].Template.Uid)
	detail, err := svc.GetVirtualService(t.Context(), connect.NewRequest(&v1.GetVirtualServiceRequest{Uid: string(obj.UID)}))
	require.NoError(t, err)
	require.True(t, detail.Msg.Status.Invalid)
	require.Equal(t, "template missing", detail.Msg.Status.Message)
	require.Equal(t, "missing", detail.Msg.Listener.Name)
	require.Equal(t, "missing", detail.Msg.GetAccessLogConfigs().Refs[0].Name)
	require.Equal(t, "missing", detail.Msg.AdditionalRoutes[0].Name)
	require.Equal(t, "missing", detail.Msg.AdditionalHttpFilters[0].Name)
	require.JSONEq(t, string(obj.Raw()), detail.Msg.Raw)
}
