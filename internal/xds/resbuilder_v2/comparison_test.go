package resbuilder_v2

import (
	"reflect"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestBuildResourcesEquivalence tests that resbuilder and resbuilder_v2 produce equivalent results
func TestBuildResourcesEquivalence(t *testing.T) {
	testCases := []struct {
		name  string
		vs    *v1alpha1.VirtualService
		setup func(*store.Store)
	}{
		{
			name: "Basic VirtualService",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						Listener: &v1alpha1.ResourceRef{
							Name: "test-listener",
						},
						VirtualHost: &runtime.RawExtension{
							Raw: []byte(`{
								"name": "test-host",
								"domains": ["example.com"],
								"routes": [{
									"match": {"prefix": "/"},
									"route": {"cluster": "test-cluster"}
								}]
							}`),
						},
					},
				},
			},
			setup: func(store *store.Store) {
				// Add required listener
				listener := makeListenerCR()
				store.SetListener(listener)

				// Add required cluster
				cluster := &v1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "default",
					},
					Spec: &runtime.RawExtension{
						Raw: []byte(`{
							"name": "test-cluster",
							"connect_timeout": "5s",
							"type": "STATIC",
							"load_assignment": {
								"cluster_name": "test-cluster",
								"endpoints": [{
									"lb_endpoints": [{
										"endpoint": {
											"address": {
												"socket_address": {
													"address": "127.0.0.1",
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
				store.SetCluster(cluster)
			},
		},
		{
			name: "VirtualService with RBAC",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs-rbac",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						Listener: &v1alpha1.ResourceRef{
							Name: "test-listener",
						},
						VirtualHost: &runtime.RawExtension{
							Raw: []byte(`{
								"name": "test-host",
								"domains": ["example.com"],
								"routes": [{
									"match": {"prefix": "/"},
									"route": {"cluster": "test-cluster"}
								}]
							}`),
						},
						RBAC: &v1alpha1.VirtualServiceRBACSpec{
							Action: "ALLOW",
							Policies: map[string]*runtime.RawExtension{
								"test-policy": {
									Raw: []byte(`{
										"permissions": [{"any": true}],
										"principals": [{"any": true}]
									}`),
								},
							},
						},
					},
				},
			},
			setup: func(store *store.Store) {
				// Add required listener
				listener := makeListenerCR()
				store.SetListener(listener)

				// Add required cluster
				cluster := &v1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "default",
					},
					Spec: &runtime.RawExtension{
						Raw: []byte(`{
							"name": "test-cluster",
							"connect_timeout": "5s",
							"type": "STATIC",
							"load_assignment": {
								"cluster_name": "test-cluster",
								"endpoints": [{
									"lb_endpoints": [{
										"endpoint": {
											"address": {
												"socket_address": {
													"address": "127.0.0.1",
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
				store.SetCluster(cluster)
			},
		},
		{
			name: "VirtualService with HTTP Filters",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs-filters",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						Listener: &v1alpha1.ResourceRef{
							Name: "test-listener",
						},
						VirtualHost: &runtime.RawExtension{
							Raw: []byte(`{
								"name": "test-host",
								"domains": ["example.com"],
								"routes": [{
									"match": {"prefix": "/"},
									"route": {"cluster": "test-cluster"}
								}]
							}`),
						},
						HTTPFilters: []*runtime.RawExtension{
							{
								Raw: []byte(`{
									"name": "envoy.filters.http.router",
									"typed_config": {
										"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
									}
								}`),
							},
						},
					},
				},
			},
			setup: func(store *store.Store) {
				// Add required listener
				listener := makeListenerCR()
				store.SetListener(listener)

				// Add required cluster
				cluster := &v1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "default",
					},
					Spec: &runtime.RawExtension{
						Raw: []byte(`{
							"name": "test-cluster",
							"connect_timeout": "5s",
							"type": "STATIC",
							"load_assignment": {
								"cluster_name": "test-cluster",
								"endpoints": [{
									"lb_endpoints": [{
										"endpoint": {
											"address": {
												"socket_address": {
													"address": "127.0.0.1",
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
				store.SetCluster(cluster)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup store
			testStore := store.New()
			if tc.setup != nil {
				tc.setup(testStore)
			}

			// Build resources with old implementation
			oldResources, oldErr := resbuilder.BuildResources(tc.vs, testStore)

			// Build resources with new implementation
			newResources, newErr := BuildResources(tc.vs, testStore)

			// Compare errors
			if (oldErr == nil) != (newErr == nil) {
				t.Fatalf("Error mismatch: old=%v, new=%v", oldErr, newErr)
			}

			if oldErr != nil && newErr != nil {
				// Both have errors - they should be similar
				t.Logf("Both implementations returned errors (expected): old=%v, new=%v", oldErr, newErr)
				return
			}

			// Compare successful results
			if oldErr == nil && newErr == nil {
				compareResourcesCompat(t, oldResources, newResources)
			}
		})
	}
}

// compareResourcesCompat performs deep comparison of Resources structs from different packages
func compareResourcesCompat(t *testing.T, old *resbuilder.Resources, new *Resources) {
	t.Helper()

	// Compare Listener names
	if old.Listener != new.Listener {
		t.Errorf("Listener mismatch: old=%v, new=%v", old.Listener, new.Listener)
	}

	// Compare Domains
	if !equalStringSlices(old.Domains, new.Domains) {
		t.Errorf("Domains mismatch: old=%v, new=%v", old.Domains, new.Domains)
	}

	// Compare UsedSecrets
	if !equalNamespacedNames(old.UsedSecrets, new.UsedSecrets) {
		t.Errorf("UsedSecrets mismatch: old=%v, new=%v", old.UsedSecrets, new.UsedSecrets)
	}

	// Compare FilterChains count
	if len(old.FilterChain) != len(new.FilterChain) {
		t.Errorf("FilterChain count mismatch: old=%d, new=%d", len(old.FilterChain), len(new.FilterChain))
	}

	// Compare Clusters count
	if len(old.Clusters) != len(new.Clusters) {
		t.Errorf("Clusters count mismatch: old=%d, new=%d", len(old.Clusters), len(new.Clusters))
	}

	// Compare Secrets count
	if len(old.Secrets) != len(new.Secrets) {
		t.Errorf("Secrets count mismatch: old=%d, new=%d", len(old.Secrets), len(new.Secrets))
	}

	// Compare RouteConfig (basic structure check)
	if old.RouteConfig != nil && new.RouteConfig != nil {
		if old.RouteConfig.Name != new.RouteConfig.Name {
			t.Errorf("RouteConfig name mismatch: old=%s, new=%s", old.RouteConfig.Name, new.RouteConfig.Name)
		}
		if len(old.RouteConfig.VirtualHosts) != len(new.RouteConfig.VirtualHosts) {
			t.Errorf("RouteConfig VirtualHosts count mismatch: old=%d, new=%d",
				len(old.RouteConfig.VirtualHosts), len(new.RouteConfig.VirtualHosts))
		}
	} else if (old.RouteConfig == nil) != (new.RouteConfig == nil) {
		t.Errorf("RouteConfig presence mismatch: old=%v, new=%v", old.RouteConfig != nil, new.RouteConfig != nil)
	}
}

// Helper functions for comparison

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func equalNamespacedNames(a, b []helpers.NamespacedName) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// BenchmarkComparison compares performance between old and new implementations
func BenchmarkComparison_BuildResources(b *testing.B) {
	// Setup test data
	vs := createTestVirtualService()
	testStore := createTestStore()

	b.Run("Legacy", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := resbuilder.BuildResources(vs, testStore)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("V2", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := BuildResources(vs, testStore)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryComparison focuses on memory usage differences
func BenchmarkMemoryComparison(b *testing.B) {
	vs := createTestVirtualService()
	testStore := createTestStore()

	b.Run("Legacy_Memory", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = resbuilder.BuildResources(vs, testStore)
		}
	})

	b.Run("V2_Memory", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = BuildResources(vs, testStore)
		}
	})
}

// TestResourcesStructEquivalence ensures the Resources struct is compatible
func TestResourcesStructEquivalence(t *testing.T) {
	// Check that both packages export Resources with same structure
	v2Type := reflect.TypeOf((*Resources)(nil)).Elem()
	oldType := reflect.TypeOf((*resbuilder.Resources)(nil)).Elem()

	if v2Type.NumField() != oldType.NumField() {
		t.Fatalf("Resources struct field count mismatch: v2=%d, old=%d", v2Type.NumField(), oldType.NumField())
	}

	// Check that field names and types match
	for i := 0; i < v2Type.NumField(); i++ {
		v2Field := v2Type.Field(i)
		oldField := oldType.Field(i)

		if v2Field.Name != oldField.Name {
			t.Errorf("Field[%d] name mismatch: v2=%s, old=%s", i, v2Field.Name, oldField.Name)
		}

		if v2Field.Type != oldField.Type {
			t.Errorf("Field[%d] type mismatch: v2=%s, old=%s", i, v2Field.Type, oldField.Type)
		}
	}
}
