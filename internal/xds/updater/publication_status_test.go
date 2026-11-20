package updater

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func TestPublicationStatus(t *testing.T) {
	tests := []struct {
		name      string
		nodes     map[string][]string
		domains   map[string]string
		listeners map[string]string
		brokenVS  string
		wantError string
		invalid   map[string][]string
		affected  map[string][]string
	}{
		{
			name:      "only nodes with a rejected snapshot and their common service are held",
			nodes:     map[string][]string{"frontend": {"blue"}, "admin": {"blue"}, "separate": {"green"}, "shared": {"*"}},
			listeners: map[string]string{"admin": "other"},
			wantError: "duplicate address",
			invalid:   map[string][]string{"frontend": {"blue", "duplicate address"}, "admin": {"blue"}, "shared": {"blue"}},
			affected:  map[string][]string{"frontend": {"blue"}, "admin": {"blue"}, "shared": {"blue"}},
		},
		{
			name:      "broken common service stops publication for its peers",
			nodes:     map[string][]string{"broken": {"*"}, "separate": {"green"}},
			brokenVS:  "broken",
			wantError: "missing dependency",
			invalid:   map[string][]string{"broken": {"missing dependency"}, "separate": {"publication held"}},
			affected:  map[string][]string{"broken": {"green"}, "separate": {"green"}},
		},
		{
			name:      "broken service stops publication on an unrelated node",
			nodes:     map[string][]string{"broken": {"blue"}, "separate": {"green"}},
			brokenVS:  "broken",
			wantError: "missing dependency",
			invalid:   map[string][]string{"broken": {"missing dependency"}, "separate": {"publication held"}},
			affected:  map[string][]string{"broken": {"blue"}, "separate": {"green"}},
		},
		{
			name:      "duplicate domain identifies both owners and its scope",
			nodes:     map[string][]string{"app": {"blue"}, "shared": {"*"}, "separate": {"green"}},
			domains:   map[string]string{"app": "clash.example", "shared": "clash.example"},
			wantError: "duplicate domain",
			invalid: map[string][]string{
				"app": {"blue", "app", "shared"}, "shared": {"blue", "app", "shared"}, "separate": {"publication held"},
			},
			affected: map[string][]string{"app": {"blue"}, "shared": {"blue", "green"}, "separate": {"green"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			for _, listener := range []string{"web", "other"} {
				s.SetListener(makeListenerCR(listener, "127.0.0.1", 443))
			}
			for name, nodes := range tt.nodes {
				listener := tt.listeners[name]
				if listener == "" {
					listener = "web"
				}
				s.SetVirtualService(makeVSWithListener(name, nodes, listener))
			}
			t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
				if vs.Name == tt.brokenVS {
					return nil, errors.New("missing dependency")
				}
				domain := tt.domains[vs.Name]
				if domain == "" {
					domain = vs.Name + ".example"
				}
				listener, err := vs.GetListenerNamespacedName()
				require.NoError(t, err)
				return &resbuilder.Resources{Listener: listener, Domains: []string{domain}}, nil
			}))
			cache := wrapped.NewSnapshotCache()
			err, _, statuses, _ := buildSnapshots(t.Context(), cache, s, nil)
			require.ErrorContains(t, err, tt.wantError)
			for name := range tt.nodes {
				nn := helpers.NamespacedName{Namespace: testNamespace, Name: name}
				status, found := statuses[nn]
				require.True(t, found, "%s has no publication result", name)
				fragments, invalid := tt.invalid[name]
				require.Equal(t, invalid, status.Invalid, "%s: %s", name, status.Message)
				require.Equal(t, tt.affected[name], status.AffectedNodes, "%s", name)
				for _, fragment := range fragments {
					require.Contains(t, status.Message, fragment, "%s", name)
				}
			}
			_, greenErr := cache.GetSnapshot("green")
			if tt.wantError == "duplicate address" {
				require.NoError(t, greenErr)
			} else {
				require.Error(t, greenErr)
			}
		})
	}
}

func TestPublicationStatusIncompleteSDS(t *testing.T) {
	s := store.New()
	s.SetListener(makeListenerCR("web", "127.0.0.1", 443))
	s.SetVirtualService(makeVSWithListener("app", []string{"blue"}, "web"))
	s.SetVirtualService(makeVSWithListener("shared", []string{"*"}, "web"))
	incomplete := true
	t.Cleanup(withStubbedBuilder(t, func(_ *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		res := &resbuilder.Resources{Listener: helpers.NamespacedName{Namespace: testNamespace, Name: "web"}}
		if incomplete {
			res.Warnings = []string{"missing sds-b", "missing sds-a"}
		}
		return res, nil
	}))
	cache := wrapped.NewSnapshotCache()
	err, _, statuses, _ := buildSnapshots(t.Context(), cache, s, nil)
	require.NoError(t, err)
	_, err = cache.GetSnapshot("blue")
	require.NoError(t, err)
	for _, name := range []string{"app", "shared"} {
		status := statuses[helpers.NamespacedName{Namespace: testNamespace, Name: name}]
		require.False(t, status.Invalid)
		require.Regexp(t, "sds-a.*sds-b", status.Message)
		require.Empty(t, status.AffectedNodes)
	}
	// A failed cache write holds only the VS targeting that node.
	key := helpers.NamespacedName{Namespace: testNamespace, Name: "app"}
	failure := errors.New("failed")
	statuses = publicationResults(s.MapVirtualServices(), statuses, nil, map[string]error{"other": failure}, nil)
	require.False(t, statuses[key].Invalid)
	require.Contains(t, statuses[key].Message, "sds-a")
	statuses = publicationResults(s.MapVirtualServices(), statuses, nil, map[string]error{"blue": failure}, nil)
	require.True(t, statuses[key].Invalid)
	require.Contains(t, statuses[key].Message, "sds-a")
	require.Equal(t, []string{"blue"}, statuses[key].AffectedNodes)

	incomplete = false
	err, _, statuses, _ = buildSnapshots(t.Context(), cache, s, nil)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, VSStatus{}, status)
	}
}

func TestPublicationOfSkippedSDSSecret(t *testing.T) {
	s := store.New()
	load := func(name string, obj any) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("../../../test/testdata/e2e", name+".yaml"))
		require.NoError(t, err)
		require.NoError(t, yaml.Unmarshal(data, obj))
	}
	var listener v1alpha1.Listener
	var cluster v1alpha1.Cluster
	var app v1alpha1.VirtualService
	load("http_filter_sds_secrets/listener", &listener)
	load("http_filter_sds_secrets/cluster", &cluster)
	load("http_filter_sds_secrets/virtual-service", &app)
	listener.Namespace, cluster.Namespace, app.Namespace = "default", "default", "default"
	s.SetListener(&listener)
	s.SetCluster(&cluster)
	s.SetVirtualService(&app)

	cache := wrapped.NewSnapshotCache()
	u := NewCacheUpdater(cache, s)
	require.NoError(t, u.RebuildSnapshots(t.Context()))
	routes, listeners, err := cache.GetRouteConfigurationsAndListeners("test")
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Len(t, listeners, 1)
	secrets, err := cache.GetSecrets("test")
	require.NoError(t, err)
	require.Empty(t, secrets)
	warning, observed := u.PublicationStatusFor(&app)
	require.True(t, observed)
	require.False(t, warning.Invalid, "the snapshot with the OAuth2 filter was published")
	require.Contains(t, warning.Message, "default/oauth2-credentials/token")
	require.NotContains(t, warning.Message, "last snapshot publication")

	// A missing certificate for a TLS listener aborts the build, holding even its healthy peers.
	var tlsListener v1alpha1.Listener
	var broken v1alpha1.VirtualService
	load("basic_https_service/listener", &tlsListener)
	load("basic_https_service/virtual-service", &broken)
	tlsListener.Namespace = app.Namespace
	broken.Namespace, broken.Name = app.Namespace, "broken"
	broken.SetNodeIDs([]string{"other"})
	s.SetListener(&tlsListener)
	s.SetVirtualService(&broken)
	require.Error(t, u.RebuildSnapshots(t.Context()))
	held, _ := u.PublicationStatusFor(&app)
	require.True(t, held.Invalid)
	require.Contains(t, held.Message, "default/oauth2-credentials/token")
	require.Contains(t, held.Message, "publication held by VirtualService broken")
	root, _ := u.PublicationStatusFor(&broken)
	require.True(t, root.Invalid)
	require.Contains(t, root.Message, "can't find secret for domain")
	_, err = cache.GetSnapshot("other")
	require.Error(t, err)
	s.DeleteVirtualService(helpers.NamespacedName{Namespace: broken.Namespace, Name: broken.Name})
	require.NoError(t, u.RebuildSnapshots(t.Context()))
	recovered, _ := u.PublicationStatusFor(&app)
	require.Equal(t, warning, recovered)

	s.SetSecret(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oauth2-credentials", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"token": []byte("test-token"), "hmac": []byte("test-hmac")},
	})
	require.NoError(t, u.RebuildSnapshots(t.Context()))
	secrets, err = cache.GetSecrets("test")
	require.NoError(t, err)
	require.Len(t, secrets, 2)
	recovered, _ = u.PublicationStatusFor(&app)
	require.Equal(t, VSStatus{}, recovered)
}

func TestPublicationWaitsForRemovalFromFormerTarget(t *testing.T) {
	for _, previous := range []string{"blue", "*"} {
		t.Run(previous, func(t *testing.T) { checkPublicationAfterMove(t, previous) })
	}
}

func checkPublicationAfterMove(t *testing.T, previous string) {
	t.Helper()
	s := store.New()
	s.SetListener(makeListenerCR("move", "127.0.0.1", 8000))
	s.SetListener(makeListenerCR("one", "127.0.0.1", 8001))
	s.SetListener(makeListenerCR("two", "127.0.0.1", 8002))
	moved := makeVSWithListener("move", []string{previous}, "move")
	moved.UID = "moving-service"
	s.SetVirtualService(moved)
	for _, name := range []string{"one", "two"} {
		s.SetVirtualService(makeVSWithListener(name, []string{"blue"}, name))
	}
	t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		listener, err := vs.GetListenerNamespacedName()
		return &resbuilder.Resources{Listener: listener}, err
	}))

	cache := wrapped.NewSnapshotCache()
	u := NewCacheUpdater(cache, s)
	require.NoError(t, u.RebuildSnapshots(t.Context()))
	hasMovedListener := func(node string) bool {
		listeners, err := cache.GetListeners(node)
		require.NoError(t, err)
		for _, listener := range listeners {
			if listener.GetName() == testNamespace+"/move" {
				return true
			}
		}
		return false
	}
	require.True(t, hasMovedListener("blue"))

	u.ApplyListener(t.Context(), makeListenerCR("two", "127.0.0.1", 8001))
	moved = moved.DeepCopy()
	moved.SetNodeIDs([]string{"green"})
	u.ApplyVirtualService(t.Context(), moved)
	for range 2 {
		require.True(t, hasMovedListener("blue"))
		require.True(t, hasMovedListener("green"))
		result, observed := u.PublicationStatusFor(moved)
		require.True(t, observed)
		require.True(t, result.Invalid, "the old snapshot still contains this service")
		require.Contains(t, result.Message, "blue")
		require.Error(t, u.RebuildSnapshots(t.Context()))
	}

	u.ApplyListener(t.Context(), makeListenerCR("two", "127.0.0.1", 8002))
	require.False(t, hasMovedListener("blue"))
	require.True(t, hasMovedListener("green"))
	result, observed := u.PublicationStatusFor(moved)
	require.True(t, observed)
	require.False(t, result.Invalid)

	u.ApplyListener(t.Context(), makeListenerCR("two", "127.0.0.1", 8001))
	result, observed = u.PublicationStatusFor(moved)
	require.True(t, observed)
	require.False(t, result.Invalid, "a conflict on the former target no longer holds the service")
}

func TestPublicationStatusIsolatesBuildErrors(t *testing.T) {
	s := store.New()
	s.SetListener(makeListenerCR("web", "127.0.0.1", 80))
	good := makeVSWithListener("good", []string{"green"}, "web")
	s.SetVirtualService(good)
	bad := makeVSWithListener("broken", []string{"blue"}, "web")
	bad.Namespace = "other-team"
	s.SetVirtualService(bad)
	for _, name := range []string{"z-broken", "nearby"} {
		vs := makeVSWithListener(name, []string{"blue"}, "web")
		vs.Namespace = bad.Namespace
		s.SetVirtualService(vs)
	}
	missing := "missing secret other-team/private-cert"
	t.Cleanup(withStubbedBuilder(t, func(vs *v1alpha1.VirtualService, _ store.Store) (*resbuilder.Resources, error) {
		if strings.HasSuffix(vs.Name, "broken") {
			return nil, errors.New(missing)
		}
		return &resbuilder.Resources{Listener: helpers.NamespacedName{Namespace: testNamespace, Name: "web"}}, nil
	}))
	err, _, statuses, _ := buildSnapshots(t.Context(), wrapped.NewSnapshotCache(), s, nil)
	require.ErrorContains(t, err, missing)
	broken := statuses[helpers.NamespacedName{Namespace: bad.Namespace, Name: bad.Name}]
	require.Contains(t, broken.Message, missing)
	held := statuses[helpers.NamespacedName{Namespace: good.Namespace, Name: good.Name}]
	require.True(t, held.Invalid)
	require.Contains(t, held.Message, "publication held")
	require.NotContains(t, held.Message, "other-team")
	require.NotContains(t, held.Message, "private-cert")
	nearby := statuses[helpers.NamespacedName{Namespace: bad.Namespace, Name: "nearby"}]
	require.Contains(t, nearby.Message, "VirtualService broken;")
	require.NotContains(t, nearby.Message, "z-broken")
	require.NotContains(t, nearby.Message, "private-cert")
}

func TestPublicationStatusBoundsFleetDiagnostics(t *testing.T) {
	var nodes []string
	failures := map[string]error{}
	for i := range 200 {
		node := fmt.Sprintf("node-%03d", i)
		nodes = append(nodes, node)
		failures[node] = errors.New("another-namespace/private-dependency")
	}
	nn := helpers.NamespacedName{Namespace: testNamespace, Name: "app"}
	s := store.New()
	s.SetVirtualService(makeVSWithListener(nn.Name, nodes, "web"))
	cache := wrapped.NewSnapshotCache()
	updater := NewCacheUpdater(cache, s)
	result := publicationResults(s.MapVirtualServices(), map[helpers.NamespacedName]VSStatus{
		nn: {Invalid: true, Message: "own warning: " + strings.Repeat("界", 1800)},
	}, nil, failures, nil)
	message := updater.publicationMessage(result[nn])
	require.LessOrEqual(t, len(message), 4096)
	require.True(t, utf8.ValidString(message))
	require.Contains(t, message, "own warning")
	require.Contains(t, message, "node-000")
	require.Contains(t, message, "192 more")
	require.NotContains(t, message, "private-dependency")
}

func TestPublicationPreservesRouteOrderWithinEachVirtualService(t *testing.T) {
	s := store.New()
	s.SetListener(makeListenerCR("web", "127.0.0.1", 8080))
	for _, name := range []string{"zebra", "alpha"} {
		vs := makeVSWithListener(name, []string{"node"}, "web")
		vs.Spec.HTTPFilters = []*runtime.RawExtension{
			{Raw: []byte(`{"name":"envoy.filters.http.router",
				"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}}`)},
		}
		vs.Spec.VirtualHost = &runtime.RawExtension{Raw: fmt.Appendf(nil, `{"domains":[%q],"routes":[
			{"name":"special","match":{"prefix":"/shop/special"},"direct_response":{"status":200}},
			{"name":"catalog","match":{"prefix":"/shop"},"direct_response":{"status":202}},
			{"name":"default","match":{"prefix":"/"},"direct_response":{"status":404}}]}`, name+".example")}
		s.SetVirtualService(vs)
	}
	cache := wrapped.NewSnapshotCache()
	err, _, _, _ := buildSnapshots(t.Context(), cache, s, nil)
	require.NoError(t, err)
	configs, err := cache.GetRouteConfigurations("node")
	require.NoError(t, err)
	require.Len(t, configs, 2)
	for _, cfg := range configs {
		var names []string
		for _, route := range cfg.GetVirtualHosts()[0].GetRoutes() {
			names = append(names, route.GetName())
		}
		require.Equal(t, []string{"special", "catalog", "default"}, names, cfg.GetName())
	}
}
