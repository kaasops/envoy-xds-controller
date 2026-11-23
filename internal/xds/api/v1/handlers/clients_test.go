package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
)

func TestFilterClients(t *testing.T) {
	// In the order the registry lists them: by protocol, then by stream.
	all := []clients.Info{
		{Protocol: clients.Delta, ID: 1, NodeID: "edge"},
		{Protocol: clients.Delta, ID: 2, NodeID: "core"},
		{Protocol: clients.Delta, ID: 3, NodeID: "core"},
		{Protocol: clients.Delta, ID: 4}, // has not named its node yet
		{Protocol: clients.SotW, ID: 1, NodeID: "edge"},
	}
	tests := []struct {
		name       string
		nodeID     string
		allowed    map[string]struct{}
		restricted bool
		want       []string
	}{
		{
			name: "without a restriction every stream is listed, grouped by node",
			want: []string{"/delta#4", "core/delta#2", "core/delta#3", "edge/delta#1", "edge/sotw#1"},
		},
		{
			name:       "a restricted user sees the streams of allowed nodes only",
			allowed:    map[string]struct{}{"core": {}},
			restricted: true,
			want:       []string{"core/delta#2", "core/delta#3"},
		},
		{
			name:       "a user allowed no node sees nothing, nameless streams included",
			allowed:    map[string]struct{}{},
			restricted: true,
			want:       []string{},
		},
		{
			name:   "node_id narrows the list",
			nodeID: "edge",
			want:   []string{"edge/delta#1", "edge/sotw#1"},
		},
		{
			name:       "node_id cannot reach past the restriction",
			nodeID:     "edge",
			allowed:    map[string]struct{}{"core": {}},
			restricted: true,
			want:       []string{},
		},
		{
			name:   "a node nobody connected with gives an empty list",
			nodeID: "typo",
			want:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := []string{}
			filtered := filterClients(all, tt.nodeID, tt.allowed, tt.restricted)
			if filtered == nil {
				t.Fatal("got nil, want an empty list: gin encodes nil as null")
			}
			for _, c := range filtered {
				got = append(got, fmt.Sprintf("%s/%s#%d", c.NodeID, c.Protocol, c.ID))
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

type clientsBody struct {
	ControllerPod    string                       `json:"controllerPod"`
	SnapshotVersions map[string]map[string]string `json:"snapshotVersions"`
	Clients          []struct {
		NodeID       string            `json:"nodeId"`
		Address      string            `json:"address"`
		EnvoyVersion string            `json:"envoyVersion"`
		Types        []json.RawMessage `json:"types"`
	} `json:"clients"`
}

func registryWithNodes(nodeIDs ...string) *clients.Registry {
	registry := clients.NewRegistry()
	for i, nodeID := range nodeIDs {
		key := clients.Key{Protocol: clients.Delta, ID: int64(i + 1)}
		registry.Set(key, &clients.Info{Address: fmt.Sprintf("10.0.0.%d:41000", i+1)})
		registry.Update(key, &clients.Info{NodeID: nodeID, Version: "v1.40.2"})
	}
	return registry
}

// cacheWithSnapshots returns a cache that holds a snapshot of the given
// version for every node.
func cacheWithSnapshots(t *testing.T, version string, nodeIDs ...string) *xdscache.SnapshotCache {
	t.Helper()
	snapshots := xdscache.NewSnapshotCache()
	for _, nodeID := range nodeIDs {
		snapshot, err := cache.NewSnapshot(version, map[resourcev3.Type][]types.Resource{
			resourcev3.ClusterType:  {&clusterv3.Cluster{Name: "backend"}},
			resourcev3.ListenerType: {&listenerv3.Listener{Name: "https"}},
			resourcev3.RouteType:    {&routev3.RouteConfiguration{Name: "https"}},
			resourcev3.SecretType:   {&tlsv3.Secret{Name: "cert"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := snapshots.SetSnapshot(t.Context(), nodeID, snapshot); err != nil {
			t.Fatal(err)
		}
	}
	return snapshots
}

// serveClients calls the handler the way gin does after the auth middleware:
// restriction, when not nil, is what the middleware left in the context.
func serveClients(t *testing.T, h *handler, query string, restriction any) (int, clientsBody) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/clients"+query, nil)
	if restriction != nil {
		ctx.Set(middlewares.AvailableNodeIDs, restriction)
	}

	h.getClients(ctx)

	var body clientsBody
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%v in %s", err, rec.Body)
		}
	}
	return rec.Code, body
}

// The auth middleware only authenticates; what a user may see is decided by
// each handler from the node ids it left in the context.
func TestGetClientsServesOnlyAllowedNodes(t *testing.T) {
	h := &handler{
		cache:   cacheWithSnapshots(t, "7", "core", "edge"),
		clients: registryWithNodes("core", "edge"),
		podName: "controller-0",
	}

	code, body := serveClients(t, h, "", map[string]struct{}{"edge": {}})

	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if body.ControllerPod != "controller-0" {
		t.Errorf("controllerPod %q, want controller-0", body.ControllerPod)
	}
	if len(body.Clients) != 1 {
		t.Fatalf("got %d clients, want the one of node edge: %+v", len(body.Clients), body)
	}
	got := body.Clients[0]
	if got.NodeID != "edge" || got.Address != "10.0.0.2:41000" || got.EnvoyVersion != "v1.40.2" || got.Types == nil {
		t.Errorf("unexpected client %+v", got)
	}
	if _, leaked := body.SnapshotVersions["core"]; leaked || len(body.SnapshotVersions) != 1 {
		t.Errorf("snapshot versions of nodes the caller may not see: %v", body.SnapshotVersions)
	}
}

// A context value that cannot be read must not lift the restriction.
func TestGetClientsHidesEverythingFromACallerAllowedNothing(t *testing.T) {
	for name, restriction := range map[string]any{
		"no node is allowed":      map[string]struct{}{},
		"the value is unreadable": []string{"core", "edge"},
	} {
		t.Run(name, func(t *testing.T) {
			h := &handler{
				cache:   cacheWithSnapshots(t, "7", "core", "edge"),
				clients: registryWithNodes("core", "edge", ""),
			}

			code, body := serveClients(t, h, "", restriction)

			if code != http.StatusOK || len(body.Clients) != 0 || len(body.SnapshotVersions) != 0 {
				t.Errorf("status %d, body %+v, want an empty list", code, body)
			}
		})
	}
}

// A snapshot version is a counter of one controller pod, so it can only be
// compared with what the streams of the same pod acknowledged.
func TestGetClientsReportsSnapshotVersionsOfListedNodes(t *testing.T) {
	snapshots := cacheWithSnapshots(t, "7", "core")
	// Every type at a version of its own, and a type the node has no version of.
	uneven := &cache.Snapshot{}
	uneven.Resources[types.Cluster] = cache.NewResources("1", []types.Resource{&clusterv3.Cluster{Name: "backend"}})
	uneven.Resources[types.Listener] = cache.NewResources("2", []types.Resource{&listenerv3.Listener{Name: "https"}})
	uneven.Resources[types.Route] = cache.NewResources("3", []types.Resource{&routev3.RouteConfiguration{Name: "https"}})
	if err := snapshots.SetSnapshot(t.Context(), "edge", uneven); err != nil {
		t.Fatal(err)
	}
	h := &handler{cache: snapshots, clients: registryWithNodes("core", "edge", "typo")}

	code, body := serveClients(t, h, "", nil)

	if code != http.StatusOK || len(body.Clients) != 3 {
		t.Fatalf("status %d, body %+v", code, body)
	}
	// Node typo has no snapshot.
	want := map[string]map[string]string{
		"core": {
			resourcev3.ClusterType:  "7",
			resourcev3.ListenerType: "7",
			resourcev3.RouteType:    "7",
			resourcev3.SecretType:   "7",
		},
		"edge": {
			resourcev3.ClusterType:  "1",
			resourcev3.ListenerType: "2",
			resourcev3.RouteType:    "3",
		},
	}
	if !reflect.DeepEqual(body.SnapshotVersions, want) {
		t.Errorf("got %v, want %v", body.SnapshotVersions, want)
	}
}

// Every other handler refuses a repeated node_id instead of picking one.
func TestGetClientsRefusesRepeatedNodeID(t *testing.T) {
	h := &handler{cache: xdscache.NewSnapshotCache(), clients: registryWithNodes("core", "edge")}

	code, _ := serveClients(t, h, "?node_id=core&node_id=edge", nil)

	if code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", code)
	}
}

// The list used to be served in development mode only.
func TestClientsRouteIsRegisteredWithTheAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine, xdscache.NewSnapshotCache(), registryWithNodes("core", "edge"))
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/clients?node_id=core", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	var body clientsBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%v in %s", err, rec.Body)
	}
	if len(body.Clients) != 1 || body.Clients[0].NodeID != "core" {
		t.Errorf("want the client of node core, got %s", rec.Body)
	}
	// In Kubernetes the host name is the pod name: it tells replicas apart.
	if host, _ := os.Hostname(); host == "" || body.ControllerPod != host {
		t.Errorf("controllerPod %q, want the host name %q", body.ControllerPod, host)
	}
}
