package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/stretchr/testify/require"
)

func TestNodeInventoryCountsDescribeOnlyVisibleServedResources(t *testing.T) {
	c := cacheWithSnapshots(t, "1", "edge", "hidden")
	empty, err := cachev3.NewSnapshot("2", nil)
	require.NoError(t, err)
	c.Pause("edge", xdscache.Pause{})
	c.Pause("waiting", xdscache.Pause{})
	for _, node := range []string{"edge", "empty", "waiting"} {
		require.NoError(t, c.SetSnapshot(t.Context(), node, empty))
	}
	read := func(query string) string {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest("GET", "/api/v1/nodeIDs"+query, nil)
		ctx.Set(middlewares.AvailableNodeIDs, map[string]struct{}{"edge": {}, "empty": {}, "waiting": {}})
		h := handler{cache: c}
		h.getNodeIDs(ctx)
		require.Equal(t, 200, recorder.Code)
		return recorder.Body.String()
	}
	require.JSONEq(t, `["edge","empty","waiting"]`, read(""))
	require.JSONEq(t, `[{"node_id":"edge","counts":{"listeners":1,"routes":1,"clusters":1}},
		{"node_id":"empty","counts":{"listeners":0,"routes":0,"clusters":0}},{"node_id":"waiting"}]`, read("?include=counts"))
}

func TestInventoryWithRegisteredUnservedNode(t *testing.T) {
	r, c := snapshotServer(t, false, map[string]struct{}{"node-a": {}, "waiting": {}})
	RegisterRoutes(r, c, nil)
	c.RestorePaused(map[string]xdscache.Pause{"waiting": {Reason: "maintenance"}})
	for _, item := range []struct{ node, secret string }{
		{"node-a", "public-key"}, {"waiting", "pending-only"}, {"hidden", "hidden-key"},
	} {
		snapshot, err := cachev3.NewSnapshot("7", map[resourcev3.Type][]types.Resource{
			resourcev3.SecretType: {&tlsv3.Secret{Name: "apps/" + item.secret, Type: &tlsv3.Secret_GenericSecret{GenericSecret: &tlsv3.GenericSecret{}}}},
		})
		require.NoError(t, err)
		require.NoError(t, c.SetSnapshot(t.Context(), item.node, snapshot))
	}

	get := func(t *testing.T, path string, status int) string {
		t.Helper()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/"+path, nil))
		require.Equal(t, status, w.Code, "%s: %s", path, w.Body.String())
		return w.Body.String()
	}

	require.JSONEq(t, `["node-a","waiting"]`, get(t, "nodeIDs", http.StatusOK))
	require.Contains(t, get(t, "pausedNodes", http.StatusOK), `"waiting"`)
	t.Run("versions include only visible served nodes", func(t *testing.T) {
		require.JSONEq(t, `[{"node_id":"node-a","versions":{"listeners":"","routes":"","clusters":"","secrets":"7"}}]`,
			get(t, "nodeIDs/versions", http.StatusOK))
	})
	t.Run("secret lookup searches only visible served snapshots", func(t *testing.T) {
		require.JSONEq(t, `{"name":"public-key","namespace":"apps","type":"generic_secret"}`,
			get(t, "secrets/apps/public-key", http.StatusOK))
		get(t, "secrets/apps/hidden-key", http.StatusNotFound)
		get(t, "secrets/apps/pending-only", http.StatusNotFound)
	})

	// The published node is still registered as served. Failures reading it must remain visible.
	c.SnapshotCache.ClearSnapshot("node-a")
	get(t, "nodeIDs/versions", http.StatusInternalServerError)
	get(t, "secrets/apps/public-key", http.StatusInternalServerError)

	c.ClearSnapshot("node-a")
	require.JSONEq(t, `[]`, get(t, "nodeIDs/versions", http.StatusOK))
}
