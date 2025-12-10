package resbuilder

import (
	"testing"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestApplyVirtualServiceTemplate(t *testing.T) {
	testCases := []struct {
		name          string
		vs            *v1alpha1.VirtualService
		templates     []*v1alpha1.VirtualServiceTemplate
		expectError   bool
		errorContains string
	}{
		{
			name: "No template",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{},
			},
			templates:   nil,
			expectError: false,
		},
		{
			name: "Template not found",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					Template: &v1alpha1.ResourceRef{
						Name: "non-existent",
					},
				},
			},
			templates:     nil,
			expectError:   true,
			errorContains: "not found",
		},
		{
			name: "Valid template",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					Template: &v1alpha1.ResourceRef{
						Name: "test-template",
					},
				},
			},
			templates: []*v1alpha1.VirtualServiceTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "default",
					},
					Spec: v1alpha1.VirtualServiceTemplateSpec{
						VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
							Listener: &v1alpha1.ResourceRef{
								Name: "test-listener",
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock store
			mockStore := store.New()
			for _, template := range tc.templates {
				mockStore.SetVirtualServiceTemplate(template)
			}

			// Call the function
			result, err := applyVirtualServiceTemplate(tc.vs, mockStore)

			// Check the result
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.vs.Spec.Template == nil {
					assert.Equal(t, tc.vs, result)
				} else {
					assert.NotEqual(t, tc.vs, result)
				}
			}
		})
	}
}

func TestCheckFilterChainsConflicts(t *testing.T) {
	testCases := []struct {
		name          string
		vs            *v1alpha1.VirtualService
		expectError   bool
		errorContains string
	}{
		{
			name: "No conflicts",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{},
			},
			expectError: false,
		},
		{
			name: "Conflict with VirtualHost",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						VirtualHost: &runtime.RawExtension{},
					},
				},
			},
			expectError:   true,
			errorContains: "virtual host is set",
		},
		{
			name: "Conflict with AdditionalRoutes",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						AdditionalRoutes: []*v1alpha1.ResourceRef{
							{Name: "test-route"},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "additional routes are set",
		},
		{
			name: "Conflict with HTTPFilters",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						HTTPFilters: []*runtime.RawExtension{
							{},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "http filters are set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFilterChainsConflicts(tc.vs)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractClustersFromFilterChains(t *testing.T) {
	// Helper function to create a TCP proxy filter chain
	createTCPProxyFilterChain := func(clusterName string) *listenerv3.FilterChain {
		tcpProxy := &tcpProxyv3.TcpProxy{
			StatPrefix: "test",
			ClusterSpecifier: &tcpProxyv3.TcpProxy_Cluster{
				Cluster: clusterName,
			},
		}
		tcpProxyAny, err := anypb.New(tcpProxy)
		require.NoError(t, err)

		return &listenerv3.FilterChain{
			Filters: []*listenerv3.Filter{
				{
					Name: "envoy.filters.network.tcp_proxy",
					ConfigType: &listenerv3.Filter_TypedConfig{
						TypedConfig: tcpProxyAny,
					},
				},
			},
		}
	}

	// Helper function to create a cluster in store
	createClusterInStore := func(mockStore store.Store, name string) {
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: &runtime.RawExtension{
				Raw: []byte(`{
					"name": "` + name + `",
					"connect_timeout": "30s",
					"type": "LOGICAL_DNS",
					"load_assignment": {
						"cluster_name": "` + name + `",
						"endpoints": [{
							"lb_endpoints": [{
								"endpoint": {
									"address": {
										"socket_address": {
											"address": "localhost",
											"port_value": 8080
										}
									}
								}
							}]
						}]
					}
				}`),
			},
		}
		mockStore.SetCluster(cluster)
	}

	t.Run("Empty filter chains", func(t *testing.T) {
		mockStore := store.New()
		clusters, err := extractClustersFromFilterChains(nil, mockStore)
		assert.NoError(t, err)
		assert.Empty(t, clusters)
	})

	t.Run("Single filter chain with TCP proxy", func(t *testing.T) {
		mockStore := store.New()
		createClusterInStore(mockStore, "cluster-1")

		filterChains := []*listenerv3.FilterChain{
			createTCPProxyFilterChain("cluster-1"),
		}

		clusters, err := extractClustersFromFilterChains(filterChains, mockStore)
		assert.NoError(t, err)
		assert.Len(t, clusters, 1)
		assert.Equal(t, "cluster-1", clusters[0].Name)
	})

	t.Run("Multiple filter chains with TCP proxy", func(t *testing.T) {
		mockStore := store.New()
		createClusterInStore(mockStore, "cluster-1")
		createClusterInStore(mockStore, "cluster-2")
		createClusterInStore(mockStore, "cluster-3")

		filterChains := []*listenerv3.FilterChain{
			createTCPProxyFilterChain("cluster-1"),
			createTCPProxyFilterChain("cluster-2"),
			createTCPProxyFilterChain("cluster-3"),
		}

		clusters, err := extractClustersFromFilterChains(filterChains, mockStore)
		assert.NoError(t, err)
		assert.Len(t, clusters, 3)

		clusterNames := make([]string, len(clusters))
		for i, c := range clusters {
			clusterNames[i] = c.Name
		}
		assert.Contains(t, clusterNames, "cluster-1")
		assert.Contains(t, clusterNames, "cluster-2")
		assert.Contains(t, clusterNames, "cluster-3")
	})

	t.Run("Cluster not found in store", func(t *testing.T) {
		mockStore := store.New()
		// Don't add cluster to store

		filterChains := []*listenerv3.FilterChain{
			createTCPProxyFilterChain("non-existent-cluster"),
		}

		clusters, err := extractClustersFromFilterChains(filterChains, mockStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Nil(t, clusters)
	})

	t.Run("Filter chain without typed config", func(t *testing.T) {
		mockStore := store.New()

		filterChains := []*listenerv3.FilterChain{
			{
				Filters: []*listenerv3.Filter{
					{
						Name: "envoy.filters.network.tcp_proxy",
						// No TypedConfig
					},
				},
			},
		}

		clusters, err := extractClustersFromFilterChains(filterChains, mockStore)
		assert.NoError(t, err)
		assert.Empty(t, clusters)
	})
}

func TestGetWildcardDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{
			name:     "Simple domain",
			domain:   "example.com",
			expected: "*.com",
		},
		{
			name:     "Subdomain",
			domain:   "sub.example.com",
			expected: "*.example.com",
		},
		{
			name:     "Already wildcard",
			domain:   "*.example.com",
			expected: "*.example.com",
		},
		{
			name:     "Empty domain",
			domain:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWildcardDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckAllDomainsUnique(t *testing.T) {
	tests := []struct {
		name        string
		domains     []string
		expectError bool
	}{
		{
			name:        "All unique domains",
			domains:     []string{"example.com", "test.com", "another.com"},
			expectError: false,
		},
		{
			name:        "Duplicate domains",
			domains:     []string{"example.com", "test.com", "example.com"},
			expectError: true,
		},
		{
			name:        "Empty domains",
			domains:     []string{},
			expectError: false,
		},
		{
			name:        "Single domain",
			domains:     []string{"example.com"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAllDomainsUnique(tt.domains)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsTLSListener(t *testing.T) {
	// Test with nil listener
	result := isTLSListener(nil)
	assert.False(t, result, "nil listener should not be considered a TLS listener")
}

func TestGetTLSType(t *testing.T) {
	tests := []struct {
		name        string
		tlsConfig   *v1alpha1.TlsConfig
		expected    string
		expectError bool
	}{
		{
			name:        "Nil config",
			tlsConfig:   nil,
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty config",
			tlsConfig:   &v1alpha1.TlsConfig{},
			expected:    "",
			expectError: true,
		},
		{
			name: "SecretRef type",
			tlsConfig: &v1alpha1.TlsConfig{
				SecretRef: &v1alpha1.ResourceRef{
					Name: "test-secret",
				},
			},
			expected:    SecretRefType,
			expectError: false,
		},
		{
			name: "AutoDiscovery type",
			tlsConfig: &v1alpha1.TlsConfig{
				AutoDiscovery: func() *bool { b := true; return &b }(),
			},
			expected:    AutoDiscoveryType,
			expectError: false,
		},
		{
			name: "Both types specified",
			tlsConfig: &v1alpha1.TlsConfig{
				SecretRef: &v1alpha1.ResourceRef{
					Name: "test-secret",
				},
				AutoDiscovery: func() *bool { b := true; return &b }(),
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getTLSType(tt.tlsConfig)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
