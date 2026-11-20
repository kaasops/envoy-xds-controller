package grpcapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/authn"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/cluster/v1/clusterv1connect"
)

func TestListCluster(t *testing.T) {
	procedure := clusterv1connect.ClusterStoreServiceListClusterProcedure
	action := lookupAction(procedure)
	require.Equal(t, "list-clusters", action)

	m, err := model.NewModelFromString(`[request_definition]
r = sub, dom, obj, act
[policy_definition]
p = sub, dom, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && r.dom == p.dom && r.obj == p.obj && r.act == p.act`)
	require.NoError(t, err)
	enforcer, err := casbin.NewEnforcer(m)
	require.NoError(t, err)
	for _, policy := range [][]string{
		{"reader", "blue", "allowed", action},
		{"reader", GeneralAccessGroup, "common", action},
		{"reader", "blue", "other-action", "list-routes"},
	} {
		_, err = enforcer.AddPolicy(policy)
		require.NoError(t, err)
	}

	s := store.New()
	for _, item := range []struct{ group, name string }{
		{"blue", "allowed"}, {"green", "allowed"}, {"blue", "other-action"}, {"", "common"}, {"", "private"},
	} {
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default", Name: item.name, UID: types.UID(item.group + "/" + item.name),
			},
			Spec: &runtime.RawExtension{Raw: []byte(`{"name":"upstream"}`)},
		}
		if item.group != "" {
			cluster.Namespace = item.group
			cluster.Labels = map[string]string{v1alpha1.LabelAccessGroup: item.group}
		}
		s.SetCluster(cluster)
	}
	_, handler := clusterv1connect.NewClusterStoreServiceHandler(NewClusterStore(s))
	for _, tt := range []struct {
		subject string
		want    []string
	}{
		{"reader", []string{"blue/allowed", "/common"}},
		{"no-grants", []string{}},
	} {
		t.Run(tt.subject, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader("{}"))
			request.Header.Set("Content-Type", "application/json")
			ctx := authn.SetInfo(request.Context(), &Authorizer{name: tt.subject, enforcer: enforcer, action: action})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request.WithContext(ctx))
			require.Equal(t, http.StatusOK, recorder.Code)
			var reply struct{ Items []struct{ UID string } }
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &reply))
			ids := make([]string, 0, len(reply.Items))
			for _, item := range reply.Items {
				ids = append(ids, item.UID)
			}
			require.Equal(t, tt.want, ids)
		})
	}
}
