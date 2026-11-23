package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/snapshots"
)

type stubBuilder struct{ built gcp.ResourceSnapshot }

func (b stubBuilder) DryBuildNode(context.Context, string) (gcp.ResourceSnapshot, error) {
	return b.built, nil
}

func (stubBuilder) RebuildSnapshots(context.Context) error { return nil }

func listenerSnapshot(t *testing.T, version string, port uint32) gcp.ResourceSnapshot {
	t.Helper()
	s, err := gcp.NewSnapshot(version, map[resource.Type][]types.Resource{resource.ListenerType: {&listenerv3.Listener{
		Name: "ns/https",
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{
			Address: "0.0.0.0", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: port},
		}}},
	}}})
	require.NoError(t, err)
	return s
}

// snapshotServer serves node-a two snapshots in a row; restrict, when set, limits the caller to its nodes.
func snapshotServer(t *testing.T, writes bool, restrict map[string]struct{}) (*gin.Engine, *xdscache.SnapshotCache) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cache := xdscache.NewSnapshotCache()
	cache.SetHistoryLimit(10)
	require.NoError(t, cache.SetSnapshot(t.Context(), "node-a", listenerSnapshot(t, "1", 443)))
	require.NoError(t, cache.SetSnapshot(t.Context(), "node-a", listenerSnapshot(t, "2", 8443)))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middlewares.UserName, "alice@example.com")
		if restrict != nil {
			c.Set(middlewares.AvailableNodeIDs, restrict)
		}
	})
	control := &snapshots.Control{
		Cache: cache, Builder: stubBuilder{built: listenerSnapshot(t, "1", 9443)}, Now: time.Now,
	}
	RegisterSnapshotRoutes(r, control, writes, zap.NewNop())
	return r, cache
}

func call(t *testing.T, r *gin.Engine, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&payload).Encode(body))
	}
	req := httptest.NewRequest(method, path, &payload)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), w.Body.String())
	return w.Code, out
}

func TestSnapshotStateAndDiff(t *testing.T) {
	r, _ := snapshotServer(t, false, nil)

	code, state := call(t, r, http.MethodGet, "/api/v1/nodes/node-a/snapshots", nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, false, state["canControl"])
	require.NotEmpty(t, state["controllerPod"])
	require.Len(t, state["history"], 1)

	code, diff := call(t, r, http.MethodGet, "/api/v1/nodes/node-a/snapshots/diff?from=served&to=built", nil)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, diff["resources"], 1)

	code, res := call(t, r, http.MethodGet,
		"/api/v1/nodes/node-a/snapshots/resource?snapshot=served&type="+resource.ListenerType+"&name=ns/https", nil)
	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, res["json"])

	code, _ = call(t, r, http.MethodGet, "/api/v1/nodes/node-x/snapshots", nil)
	require.Equal(t, http.StatusNotFound, code)
}

func TestSnapshotWritesNeedTheSetting(t *testing.T) {
	r, _ := snapshotServer(t, false, nil)

	code, _ := call(t, r, http.MethodPost, "/api/v1/nodes/node-a/freeze", map[string]string{"reason": "switch"})

	require.Equal(t, http.StatusForbidden, code)
}

func TestSnapshotWritesFollowTheNodeACL(t *testing.T) {
	r, _ := snapshotServer(t, true, map[string]struct{}{"node-b": {}})

	code, _ := call(t, r, http.MethodPost, "/api/v1/nodes/node-a/freeze", map[string]string{"reason": "switch"})
	require.Equal(t, http.StatusForbidden, code)
	code, _ = call(t, r, http.MethodGet, "/api/v1/nodes/node-a/snapshots", nil)
	require.Equal(t, http.StatusForbidden, code)
}

func TestFreezeRollbackUnfreeze(t *testing.T) {
	r, cache := snapshotServer(t, true, nil)

	code, _ := call(t, r, http.MethodPost, "/api/v1/nodes/node-a/freeze", map[string]string{})
	require.Equal(t, http.StatusBadRequest, code, "a reason is required")

	code, state := call(t, r, http.MethodPost, "/api/v1/nodes/node-a/freeze", map[string]string{"reason": "switch"})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, true, state["frozen"])
	require.Equal(t, "alice@example.com", state["freeze"].(map[string]any)["by"])

	target := cache.History("node-a")[0].ID()
	code, _ = call(t, r, http.MethodPost, "/api/v1/nodes/node-a/rollback",
		map[string]string{"to": target, "expectServed": "000000000000"})
	require.Equal(t, http.StatusConflict, code)

	served, _ := cache.Served("node-a")
	code, state = call(t, r, http.MethodPost, "/api/v1/nodes/node-a/rollback",
		map[string]string{"to": target, "expectServed": served.ID()})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, target, state["served"].(map[string]any)["id"])

	code, _ = call(t, r, http.MethodPost, "/api/v1/nodes/node-a/unfreeze", map[string]string{"expectBuilt": "000000000000"})
	require.Equal(t, http.StatusConflict, code)
	code, state = call(t, r, http.MethodPost, "/api/v1/nodes/node-a/unfreeze", map[string]string{})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, false, state["frozen"])
}

func TestFrozenNodesFollowTheNodeACL(t *testing.T) {
	r, cache := snapshotServer(t, true, map[string]struct{}{"node-a": {}})
	cache.RestoreFrozen(map[string]xdscache.Freeze{"node-a": {By: "alice"}, "node-b": {By: "bob"}})

	code, body := call(t, r, http.MethodGet, "/api/v1/frozenNodes", nil)

	require.Equal(t, http.StatusOK, code)
	require.Equal(t, map[string]any{"node-a": map[string]any{"by": "alice", "at": "0001-01-01T00:00:00Z", "reason": ""}},
		body["frozenNodes"])
}
