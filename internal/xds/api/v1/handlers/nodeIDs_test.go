package handlers

import (
	"net/http/httptest"
	"testing"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/stretchr/testify/require"
)

func TestNodeInventoryCountsDescribeOnlyVisibleServedResources(t *testing.T) {
	c := cacheWithSnapshots(t, "1", "edge", "hidden")
	empty, err := cachev3.NewSnapshot("2", nil)
	require.NoError(t, err)
	c.Freeze("edge", xdscache.Freeze{})
	c.Freeze("waiting", xdscache.Freeze{})
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
