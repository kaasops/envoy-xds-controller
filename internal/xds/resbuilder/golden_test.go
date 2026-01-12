package resbuilder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsInspectorv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// goldenSnapshot represents the serializable form of Resources for golden tests
type goldenSnapshot struct {
	ListenerName     string   `json:"listener_name"`
	FilterChainCount int      `json:"filter_chain_count"`
	FilterChainNames []string `json:"filter_chain_names"`
	HasRouteConfig   bool     `json:"has_route_config"`
	RouteConfigName  string   `json:"route_config_name,omitempty"`
	VirtualHostCount int      `json:"virtual_host_count,omitempty"`
	ClusterCount     int      `json:"cluster_count"`
	ClusterNames     []string `json:"cluster_names"`
	SecretCount      int      `json:"secret_count"`
	SecretNames      []string `json:"secret_names"`
	Domains          []string `json:"domains"`
}

// resourceToSnapshot converts Resources to a comparable snapshot
func resourceToSnapshot(r *Resources) *goldenSnapshot {
	if r == nil {
		return nil
	}

	snapshot := &goldenSnapshot{
		ListenerName:     r.Listener.String(),
		FilterChainCount: len(r.FilterChain),
		HasRouteConfig:   r.RouteConfig != nil,
		ClusterCount:     len(r.Clusters),
		SecretCount:      len(r.Secrets),
		Domains:          r.Domains,
	}

	// Extract filter chain names
	for _, fc := range r.FilterChain {
		snapshot.FilterChainNames = append(snapshot.FilterChainNames, fc.Name)
	}

	// Extract route config info
	if r.RouteConfig != nil {
		snapshot.RouteConfigName = r.RouteConfig.Name
		snapshot.VirtualHostCount = len(r.RouteConfig.VirtualHosts)
	}

	// Extract cluster names
	for _, c := range r.Clusters {
		snapshot.ClusterNames = append(snapshot.ClusterNames, c.Name)
	}

	// Extract secret names
	for _, s := range r.Secrets {
		snapshot.SecretNames = append(snapshot.SecretNames, s.Name)
	}

	return snapshot
}

const goldenDir = "testdata/golden"

// updateGolden controls whether to update golden files (set via -update flag or env)
var updateGolden = os.Getenv("UPDATE_GOLDEN") == "1"

func loadOrUpdateGolden(t *testing.T, name string, actual *goldenSnapshot) *goldenSnapshot {
	t.Helper()

	goldenPath := filepath.Join(goldenDir, name+".json")

	if updateGolden {
		// Create directory if needed
		err := os.MkdirAll(goldenDir, 0755)
		require.NoError(t, err)

		// Write actual as new golden
		data, err := json.MarshalIndent(actual, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(goldenPath, data, 0644)
		require.NoError(t, err)
		t.Logf("Updated golden file: %s", goldenPath)
		return actual
	}

	// Load existing golden
	data, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file not found: %s. Run with UPDATE_GOLDEN=1 to create it.", goldenPath)
	}
	require.NoError(t, err)

	var expected goldenSnapshot
	err = json.Unmarshal(data, &expected)
	require.NoError(t, err)

	return &expected
}

// ============================================================================
// Test Fixtures
// ============================================================================

// createBaseStore creates a store with common test data
func createBaseStore() store.Store {
	s := store.New()

	// Add HTTP listener (non-TLS)
	httpListener := createListenerCR("http-listener", 8080, false)
	s.SetListener(httpListener)

	// Add HTTPS listener (TLS)
	httpsListener := createListenerCR("https-listener", 443, true)
	s.SetListener(httpsListener)

	// Add test cluster
	testCluster := createClusterCR("test-cluster", "127.0.0.1", 8080)
	s.SetCluster(testCluster)

	// Add backend cluster
	backendCluster := createClusterCR("backend-cluster", "backend.local", 9090)
	s.SetCluster(backendCluster)

	return s
}

func createListenerCR(name string, port uint32, withTLS bool) *v1alpha1.Listener {
	l := &listenerv3.Listener{
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address:       "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: port},
				},
			},
		},
	}

	if withTLS {
		// Add TLS inspector filter to mark this as TLS listener
		// Use proper protobuf message for TLS inspector
		tlsInspectorMsg := &tlsInspectorv3.TlsInspector{}
		tlsInspectorAny, _ := anypb.New(tlsInspectorMsg)
		l.ListenerFilters = []*listenerv3.ListenerFilter{
			{
				Name: "envoy.filters.listener.tls_inspector",
				ConfigType: &listenerv3.ListenerFilter_TypedConfig{
					TypedConfig: tlsInspectorAny,
				},
			},
		}
	}

	b, _ := protoutil.Marshaler.Marshal(l)
	return &v1alpha1.Listener{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name},
		Spec:       &runtime.RawExtension{Raw: b},
	}
}

func createClusterCR(name, address string, _ uint32) *v1alpha1.Cluster {
	clusterJSON := `{
		"name":"` + name + `",
		"connect_timeout":"5s",
		"type":"LOGICAL_DNS",
		"load_assignment":{
			"cluster_name":"` + name + `",
			"endpoints":[{
				"lb_endpoints":[{
					"endpoint":{
						"address":{
							"socket_address":{"address":"` + address + `","port_value":8080}
						}
					}
				}]
			}]
		}
	}`

	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: &runtime.RawExtension{Raw: []byte(clusterJSON)},
	}
}

func createVirtualHostRaw(domains []string) []byte {
	vh := &routev3.VirtualHost{
		Name:    "test-vh",
		Domains: domains,
		Routes: []*routev3.Route{
			{
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Prefix{Prefix: "/"},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "test-cluster",
						},
					},
				},
			},
		},
	}
	b, _ := protoutil.Marshaler.Marshal(vh)
	return b
}

// ============================================================================
// Golden Tests
// ============================================================================

// TestGolden_SimpleVS tests a basic VirtualService without TLS
func TestGolden_SimpleVS(t *testing.T) {
	s := createBaseStore()

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener: &v1alpha1.ResourceRef{Name: "http-listener"},
				VirtualHost: &runtime.RawExtension{
					Raw: createVirtualHostRaw([]string{"example.com"}),
				},
			},
		},
	}

	result, err := BuildResources(vs, s)
	require.NoError(t, err)
	require.NotNil(t, result)

	actual := resourceToSnapshot(result)
	expected := loadOrUpdateGolden(t, "simple_vs", actual)

	assert.Equal(t, expected.ListenerName, actual.ListenerName)
	assert.Equal(t, expected.FilterChainCount, actual.FilterChainCount)
	assert.Equal(t, expected.HasRouteConfig, actual.HasRouteConfig)
	assert.Equal(t, expected.ClusterCount, actual.ClusterCount)
	assert.ElementsMatch(t, expected.ClusterNames, actual.ClusterNames)
	assert.ElementsMatch(t, expected.Domains, actual.Domains)
}

// TestGolden_VSWithTLSSecretRef tests VirtualService with TLS using secretRef
func TestGolden_VSWithTLSSecretRef(t *testing.T) {
	s := createBaseStore()

	// Add TLS secret to store
	tlsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
			"tls.key": []byte("-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"),
		},
	}
	s.SetSecret(tlsSecret)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener: &v1alpha1.ResourceRef{Name: "https-listener"},
				VirtualHost: &runtime.RawExtension{
					Raw: createVirtualHostRaw([]string{"secure.example.com"}),
				},
				TlsConfig: &v1alpha1.TlsConfig{
					SecretRef: &v1alpha1.ResourceRef{Name: "tls-secret"},
				},
			},
		},
	}

	result, err := BuildResources(vs, s)
	require.NoError(t, err)
	require.NotNil(t, result)

	actual := resourceToSnapshot(result)
	expected := loadOrUpdateGolden(t, "tls_secret_ref_vs", actual)

	assert.Equal(t, expected.ListenerName, actual.ListenerName)
	assert.Equal(t, expected.FilterChainCount, actual.FilterChainCount)
	assert.Equal(t, expected.SecretCount, actual.SecretCount)
	assert.ElementsMatch(t, expected.SecretNames, actual.SecretNames)
	assert.ElementsMatch(t, expected.Domains, actual.Domains)
}

// TestGolden_VSWithRBAC tests VirtualService with RBAC configuration
func TestGolden_VSWithRBAC(t *testing.T) {
	s := createBaseStore()

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbac-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener: &v1alpha1.ResourceRef{Name: "http-listener"},
				VirtualHost: &runtime.RawExtension{
					Raw: createVirtualHostRaw([]string{"api.example.com"}),
				},
				RBAC: &v1alpha1.VirtualServiceRBACSpec{
					Action: "ALLOW",
					Policies: map[string]*runtime.RawExtension{
						"allow-all": {
							Raw: []byte(`{
								"permissions": [{"any": true}],
								"principals": [{"any": true}]
							}`),
						},
					},
				},
			},
		},
	}

	result, err := BuildResources(vs, s)
	require.NoError(t, err)
	require.NotNil(t, result)

	actual := resourceToSnapshot(result)
	expected := loadOrUpdateGolden(t, "rbac_vs", actual)

	assert.Equal(t, expected.ListenerName, actual.ListenerName)
	assert.Equal(t, expected.FilterChainCount, actual.FilterChainCount)
	assert.Equal(t, expected.HasRouteConfig, actual.HasRouteConfig)
	assert.ElementsMatch(t, expected.Domains, actual.Domains)
}

// TestGolden_VSWithMultipleDomains tests VirtualService with multiple domains
func TestGolden_VSWithMultipleDomains(t *testing.T) {
	s := createBaseStore()

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-domain-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener: &v1alpha1.ResourceRef{Name: "http-listener"},
				VirtualHost: &runtime.RawExtension{
					Raw: createVirtualHostRaw(
						[]string{"example.com", "www.example.com", "api.example.com"},
					),
				},
			},
		},
	}

	result, err := BuildResources(vs, s)
	require.NoError(t, err)
	require.NotNil(t, result)

	actual := resourceToSnapshot(result)
	expected := loadOrUpdateGolden(t, "multi_domain_vs", actual)

	assert.Equal(t, expected.ListenerName, actual.ListenerName)
	assert.Equal(t, expected.FilterChainCount, actual.FilterChainCount)
	assert.ElementsMatch(t, expected.Domains, actual.Domains)
}

// TestGolden_VSWithWeightedClusters tests VirtualService with weighted clusters
func TestGolden_VSWithWeightedClusters(t *testing.T) {
	s := createBaseStore()

	// Create VirtualHost with weighted clusters
	vh := &routev3.VirtualHost{
		Name:    "weighted-vh",
		Domains: []string{"canary.example.com"},
		Routes: []*routev3.Route{
			{
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Prefix{Prefix: "/"},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_WeightedClusters{
							WeightedClusters: &routev3.WeightedCluster{
								Clusters: []*routev3.WeightedCluster_ClusterWeight{
									{Name: "test-cluster", Weight: wrappedUint32(90)},
									{Name: "backend-cluster", Weight: wrappedUint32(10)},
								},
							},
						},
					},
				},
			},
		},
	}
	vhRaw, _ := protoutil.Marshaler.Marshal(vh)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "weighted-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener:    &v1alpha1.ResourceRef{Name: "http-listener"},
				VirtualHost: &runtime.RawExtension{Raw: vhRaw},
			},
		},
	}

	result, err := BuildResources(vs, s)
	require.NoError(t, err)
	require.NotNil(t, result)

	actual := resourceToSnapshot(result)
	expected := loadOrUpdateGolden(t, "weighted_clusters_vs", actual)

	assert.Equal(t, expected.ListenerName, actual.ListenerName)
	assert.Equal(t, expected.ClusterCount, actual.ClusterCount)
	assert.ElementsMatch(t, expected.ClusterNames, actual.ClusterNames)
}

func wrappedUint32(v uint32) *wrapperspb.UInt32Value {
	return &wrapperspb.UInt32Value{Value: v}
}
