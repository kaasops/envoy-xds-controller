package testing

import (
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BenchmarkResourceBuilder benchmarks both implementations of ResourceBuilder
func BenchmarkResourceBuilder(b *testing.B) {
	// Run benchmarks for different VirtualService configurations
	scenarios := []struct {
		name string
		vs   *v1alpha1.VirtualService
	}{
		{
			name: "BasicHTTPRouting",
			vs:   createBasicHTTPRoutingVS(),
		},
		{
			name: "TLSConfiguration",
			vs:   createTLSConfigurationVS(),
		},
		{
			name: "RBACConfiguration",
			vs:   createRBACConfigurationVS(),
		},
		{
			name: "ComplexConfiguration",
			vs:   createComplexConfigurationVS(),
		},
	}

	// Run benchmarks for each scenario with both implementations
	for _, scenario := range scenarios {
		// Benchmark original implementation
		b.Run("Original-"+scenario.name, func(b *testing.B) {
			// Create a store and add required resources
			s := createBenchmarkStore()
			addRequiredResources(s, scenario.vs)

			// Create a ResourceBuilder with original implementation
			rb := resbuilder_v2.NewResourceBuilder(s)
			rb.EnableMainBuilder(false)

			// Reset timer to exclude setup time
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				_, err := rb.BuildResources(scenario.vs)
				if err != nil {
					b.Fatalf("Error building resources: %v", err)
				}
			}
		})

		// Benchmark MainBuilder implementation
		b.Run("MainBuilder-"+scenario.name, func(b *testing.B) {
			// Create a store and add required resources
			s := createBenchmarkStore()
			addRequiredResources(s, scenario.vs)

			// Create a ResourceBuilder with MainBuilder implementation
			rb := resbuilder_v2.NewResourceBuilder(s)
			rb.EnableMainBuilder(true)

			// Reset timer to exclude setup time
			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				_, err := rb.BuildResources(scenario.vs)
				if err != nil {
					b.Fatalf("Error building resources: %v", err)
				}
			}
		})
	}
}

// createBenchmarkStore creates a store for benchmarking
func createBenchmarkStore() *store.Store {
	return store.New()
}

// addRequiredResources adds required resources to the store for the given VirtualService
func addRequiredResources(s *store.Store, vs *v1alpha1.VirtualService) {
	// Add listener
	if vs.Spec.Listener != nil {
		listenerName := vs.Spec.Listener.Name
		listenerNamespace := vs.Namespace
		if vs.Spec.Listener.Namespace != nil {
			listenerNamespace = *vs.Spec.Listener.Namespace
		}

		listener := &v1alpha1.Listener{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Listener",
				APIVersion: "envoy.kaasops.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      listenerName,
				Namespace: listenerNamespace,
			},
			Spec: &runtime.RawExtension{}, // Empty spec for testing
		}

		s.SetListener(listener)
	}

	// Add VirtualService
	s.SetVirtualService(vs)
}

// createBasicHTTPRoutingVS creates a VirtualService with basic HTTP routing
func createBasicHTTPRoutingVS() *v1alpha1.VirtualService {
	vs := &v1alpha1.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: "envoy.kaasops.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-basic-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{},
	}

	// Configure listener
	listenerNamespace := "default"
	vs.Spec.Listener = &v1alpha1.ResourceRef{
		Name:      "benchmark-listener",
		Namespace: &listenerNamespace,
	}

	// Configure VirtualHost
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["example.com"],
			"routes": [
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "benchmark-cluster"
					}
				}
			]
		}`),
	}

	return vs
}

// createTLSConfigurationVS creates a VirtualService with TLS configuration
func createTLSConfigurationVS() *v1alpha1.VirtualService {
	vs := createBasicHTTPRoutingVS()
	vs.ObjectMeta.Name = "benchmark-tls-vs"

	// Configure TLS
	secretNamespace := "default"
	vs.Spec.TlsConfig = &v1alpha1.TlsConfig{
		SecretRef: &v1alpha1.ResourceRef{
			Name:      "benchmark-tls-secret",
			Namespace: &secretNamespace,
		},
	}

	// Update VirtualHost
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["secure.example.com"],
			"routes": [
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "benchmark-secure-cluster"
					}
				}
			]
		}`),
	}

	return vs
}

// createRBACConfigurationVS creates a VirtualService with RBAC configuration
func createRBACConfigurationVS() *v1alpha1.VirtualService {
	vs := createBasicHTTPRoutingVS()
	vs.ObjectMeta.Name = "benchmark-rbac-vs"

	// Configure RBAC
	vs.Spec.RBAC = &v1alpha1.VirtualServiceRBACSpec{
		Action: "ALLOW",
		Policies: map[string]*runtime.RawExtension{
			"allow-admin": {
				Raw: []byte(`{
					"permissions": [
						{
							"any": true
						}
					],
					"principals": [
						{
							"authenticated": {
								"principalName": {
									"exact": "admin"
								}
							}
						}
					]
				}`),
			},
		},
	}

	// Update VirtualHost
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["rbac.example.com"],
			"routes": [
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "benchmark-rbac-cluster"
					}
				}
			]
		}`),
	}

	return vs
}

// createComplexConfigurationVS creates a VirtualService with complex configuration
func createComplexConfigurationVS() *v1alpha1.VirtualService {
	vs := createBasicHTTPRoutingVS()
	vs.ObjectMeta.Name = "benchmark-complex-vs"

	// Configure TLS
	secretNamespace := "default"
	vs.Spec.TlsConfig = &v1alpha1.TlsConfig{
		SecretRef: &v1alpha1.ResourceRef{
			Name:      "benchmark-tls-secret",
			Namespace: &secretNamespace,
		},
	}

	// Configure RBAC
	vs.Spec.RBAC = &v1alpha1.VirtualServiceRBACSpec{
		Action: "ALLOW",
		Policies: map[string]*runtime.RawExtension{
			"allow-admin": {
				Raw: []byte(`{
					"permissions": [
						{
							"any": true
						}
					],
					"principals": [
						{
							"authenticated": {
								"principalName": {
									"exact": "admin"
								}
							}
						}
					]
				}`),
			},
		},
	}

	// Configure UseRemoteAddress
	useRemoteAddress := true
	vs.Spec.UseRemoteAddress = &useRemoteAddress

	// Configure XFFNumTrustedHops
	xffNumTrustedHops := uint32(2)
	vs.Spec.XFFNumTrustedHops = &xffNumTrustedHops

	// Configure HTTPFilters
	vs.Spec.HTTPFilters = []*runtime.RawExtension{
		{
			Raw: []byte(`{
				"name": "envoy.filters.http.cors",
				"typedConfig": {
					"@type": "type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors"
				}
			}`),
		},
	}

	// Configure UpgradeConfigs
	vs.Spec.UpgradeConfigs = []*runtime.RawExtension{
		{
			Raw: []byte(`{
				"upgradeType": "websocket",
				"enabled": true
			}`),
		},
	}

	// Update VirtualHost with more routes
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["complex.example.com"],
			"routes": [
				{
					"match": {
						"path": "/api"
					},
					"route": {
						"cluster": "benchmark-api-cluster"
					}
				},
				{
					"match": {
						"prefix": "/admin"
					},
					"route": {
						"cluster": "benchmark-admin-cluster"
					}
				},
				{
					"match": {
						"prefix": "/static"
					},
					"route": {
						"cluster": "benchmark-static-cluster"
					}
				},
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "benchmark-default-cluster"
					}
				}
			]
		}`),
	}

	return vs
}
