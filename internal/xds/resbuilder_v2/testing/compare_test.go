package testing

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CompareImplementationsTest compares results from original and MainBuilder implementations
func CompareImplementationsTest(t *testing.T, store *store.Store, vs *v1alpha1.VirtualService) {
	t.Helper()

	// Create two ResourceBuilders - one with original implementation, one with MainBuilder
	originalBuilder := resbuilder_v2.NewResourceBuilder(store)

	newBuilder := resbuilder_v2.NewResourceBuilder(store)
	newBuilder.EnableMainBuilder(true)

	// Build resources with both implementations
	originalResources, err1 := originalBuilder.BuildResources(vs)
	if err1 != nil {
		t.Fatalf("Error with original implementation: %v", err1)
	}

	newResources, err2 := newBuilder.BuildResources(vs)
	if err2 != nil {
		t.Fatalf("Error with MainBuilder implementation: %v", err2)
	}

	// Compare results
	differences := compareResources(originalResources, newResources)
	if len(differences) > 0 {
		t.Errorf("Found %d differences between implementations:", len(differences))
		for _, diff := range differences {
			t.Errorf("  - %s", diff)
		}
		t.Fail()
	}
}

// compareResources deeply compares two Resources objects and returns a list of differences
func compareResources(original, new *resbuilder_v2.Resources) []string {
	var differences []string

	// Compare Listener
	if !reflect.DeepEqual(original.Listener, new.Listener) {
		differences = append(differences, fmt.Sprintf("Listener mismatch: %v vs %v",
			original.Listener, new.Listener))
	}

	// Compare FilterChain length
	if len(original.FilterChain) != len(new.FilterChain) {
		differences = append(differences, fmt.Sprintf("FilterChain length mismatch: %d vs %d",
			len(original.FilterChain), len(new.FilterChain)))
	} else {
		// Compare each FilterChain
		for i := range original.FilterChain {
			if !proto.Equal(original.FilterChain[i], new.FilterChain[i]) {
				differences = append(differences, fmt.Sprintf("FilterChain[%d] mismatch", i))
			}
		}
	}

	// Compare RouteConfig
	if (original.RouteConfig == nil) != (new.RouteConfig == nil) {
		differences = append(differences, "RouteConfig presence mismatch")
	} else if original.RouteConfig != nil && !proto.Equal(original.RouteConfig, new.RouteConfig) {
		differences = append(differences, "RouteConfig content mismatch")
	}

	// Compare Clusters length
	if len(original.Clusters) != len(new.Clusters) {
		differences = append(differences, fmt.Sprintf("Clusters length mismatch: %d vs %d",
			len(original.Clusters), len(new.Clusters)))
	} else {
		// Compare each Cluster
		for i := range original.Clusters {
			if !proto.Equal(original.Clusters[i], new.Clusters[i]) {
				differences = append(differences, fmt.Sprintf("Cluster[%d] mismatch", i))
			}
		}
	}

	// Compare Secrets length
	if len(original.Secrets) != len(new.Secrets) {
		differences = append(differences, fmt.Sprintf("Secrets length mismatch: %d vs %d",
			len(original.Secrets), len(new.Secrets)))
	} else {
		// Compare each Secret
		for i := range original.Secrets {
			if !proto.Equal(original.Secrets[i], new.Secrets[i]) {
				differences = append(differences, fmt.Sprintf("Secret[%d] mismatch", i))
			}
		}
	}

	// Compare UsedSecrets length
	if len(original.UsedSecrets) != len(new.UsedSecrets) {
		differences = append(differences, fmt.Sprintf("UsedSecrets length mismatch: %d vs %d",
			len(original.UsedSecrets), len(new.UsedSecrets)))
	} else {
		// Compare each UsedSecret
		for i := range original.UsedSecrets {
			if !reflect.DeepEqual(original.UsedSecrets[i], new.UsedSecrets[i]) {
				differences = append(differences, fmt.Sprintf("UsedSecret[%d] mismatch: %v vs %v",
					i, original.UsedSecrets[i], new.UsedSecrets[i]))
			}
		}
	}

	// Compare Domains
	if len(original.Domains) != len(new.Domains) {
		differences = append(differences, fmt.Sprintf("Domains length mismatch: %d vs %d",
			len(original.Domains), len(new.Domains)))
	} else {
		// Compare each Domain
		for i := range original.Domains {
			if original.Domains[i] != new.Domains[i] {
				differences = append(differences, fmt.Sprintf("Domain[%d] mismatch: %s vs %s",
					i, original.Domains[i], new.Domains[i]))
			}
		}
	}

	return differences
}

// DetailedCompare provides detailed comparison for protobuf objects
func DetailedCompare(t *testing.T, name string, original, new proto.Message) {
	t.Helper()

	if !proto.Equal(original, new) {
		// Convert to text format for easier comparison
		originalText := prototext.Format(original)
		newText := prototext.Format(new)

		t.Errorf("%s objects are not equal:\nOriginal:\n%s\nNew:\n%s",
			name, originalText, newText)
	}
}

// CreateTestVirtualService creates a simple VirtualService for testing
func CreateTestVirtualService() *v1alpha1.VirtualService {
	// Create a default listener namespace
	defaultNamespace := "default"

	return &v1alpha1.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: "envoy.kaasops.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				// Add a default listener reference
				Listener: &v1alpha1.ResourceRef{
					Name:      "test-listener",
					Namespace: &defaultNamespace,
				},
				// Add a simple VirtualHost with a route
				VirtualHost: &runtime.RawExtension{
					Raw: []byte(`{
						"domains": ["example.com"],
						"routes": [
							{
								"match": {
									"prefix": "/"
								},
								"route": {
									"cluster": "test-cluster"
								}
							}
						]
					}`),
				},
			},
		},
	}
}

// CreateTestStore creates a store populated with test resources
func CreateTestStore() *store.Store {
	// Create a new store instance
	s := store.New()

	// Populate with test resources
	// This will be expanded in future iterations

	return s
}

// AddTestListener adds a test listener to the store
func AddTestListener(s *store.Store, name, namespace string) {
	listener := &v1alpha1.Listener{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Listener",
			APIVersion: "envoy.kaasops.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: &runtime.RawExtension{
			Raw: []byte(`{
				"name": "test_listener",
				"address": {
					"socket_address": {
						"address": "0.0.0.0",
						"port_value": 8080
					}
				},
				"filter_chains": []
			}`),
		},
	}

	s.SetListener(listener)
}

// AddTestVirtualService adds a test virtual service to the store
func AddTestVirtualService(s *store.Store, vs *v1alpha1.VirtualService) {
	s.SetVirtualService(vs)
}

// AddTestListenerWithTLS adds a test listener with TLS inspector to the store
func AddTestListenerWithTLS(s *store.Store, name, namespace string) {
	listener := &v1alpha1.Listener{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Listener",
			APIVersion: "envoy.kaasops.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: &runtime.RawExtension{
			Raw: []byte(`{
				"name": "tls_listener",
				"address": {
					"socket_address": {
						"address": "0.0.0.0",
						"port_value": 8443
					}
				},
				"listener_filters": [
					{
						"name": "envoy.filters.listener.tls_inspector",
						"typed_config": {
							"@type": "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"
						}
					}
				],
				"filter_chains": []
			}`),
		},
	}

	s.SetListener(listener)
}

// AddTestSecret adds a test TLS secret to the store
func AddTestSecret(s *store.Store, name, namespace string) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\ntest certificate data\n-----END CERTIFICATE-----"),
			"tls.key": []byte("-----BEGIN PRIVATE KEY-----\ntest private key data\n-----END PRIVATE KEY-----"),
		},
	}
	s.SetSecret(secret)
}

// AddTestCluster adds a test cluster to the store
func AddTestCluster(s *store.Store, name, namespace string) {
	cluster := &v1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "envoy.kaasops.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: &runtime.RawExtension{
			Raw: []byte(`{
				"name": "` + name + `",
				"type": "STRICT_DNS",
				"connect_timeout": "5s",
				"load_assignment": {
					"cluster_name": "` + name + `",
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
	s.SetCluster(cluster)
}

// TestBasicHTTPRouting tests a basic HTTP routing configuration
func TestBasicHTTPRouting(t *testing.T) {
	// Create a test store
	s := CreateTestStore()

	// Add a test listener
	listenerName := "test-listener"
	listenerNamespace := "default"
	AddTestListener(s, listenerName, listenerNamespace)

	// Add a test cluster
	clusterName := "test-cluster"
	clusterNamespace := "default"
	AddTestCluster(s, clusterName, clusterNamespace)

	// Create a virtual service that references the listener
	vs := CreateTestVirtualService()

	// Update the VirtualService to reference the listener
	vs.Spec.Listener = &v1alpha1.ResourceRef{
		Name:      listenerName,
		Namespace: &listenerNamespace,
	}

	// Add a simple VirtualHost with a route
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["example.com"],
			"routes": [
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "test-cluster"
					}
				}
			]
		}`),
	}

	// Add the virtual service to the store
	AddTestVirtualService(s, vs)

	// Run the comparison test
	CompareImplementationsTest(t, s, vs)
}

// TestTLSConfiguration tests TLS configuration handling
// TODO: Fix this test - requires proto registration for TlsInspector
/*
func TestTLSConfiguration(t *testing.T) {
	// Create a test store
	s := CreateTestStore()

	// Add a test listener with TLS inspector
	listenerName := "tls-listener"
	listenerNamespace := "default"
	AddTestListenerWithTLS(s, listenerName, listenerNamespace)

	// Add a test cluster
	clusterName := "secure-cluster"
	clusterNamespace := "default"
	AddTestCluster(s, clusterName, clusterNamespace)

	// Add a secret to the store
	secretName := "test-tls-secret"
	secretNamespace := "default"
	AddTestSecret(s, secretName, secretNamespace)

	// Create a virtual service with TLS configuration
	vs := CreateTestVirtualService()
	vs.ObjectMeta.Name = "tls-vs"

	// Update the VirtualService to reference the listener
	vs.Spec.Listener = &v1alpha1.ResourceRef{
		Name:      listenerName,
		Namespace: &listenerNamespace,
	}

	// Configure TLS with secretRef
	vs.Spec.TlsConfig = &v1alpha1.TlsConfig{
		SecretRef: &v1alpha1.ResourceRef{
			Name:      secretName,
			Namespace: &secretNamespace,
		},
	}

	// Add a simple VirtualHost with a route
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["secure.example.com"],
			"routes": [
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "secure-cluster"
					}
				}
			]
		}`),
	}

	// Add the virtual service to the store
	AddTestVirtualService(s, vs)

	// Run the comparison test
	CompareImplementationsTest(t, s, vs)
}
*/

// TestCompareImplementations runs all comparison tests
func TestCompareImplementations(t *testing.T) {
	t.Run("BasicHTTPRouting", TestBasicHTTPRouting)
	// TODO: Fix TLS test - requires proto registration for TlsInspector
	// t.Run("TLSConfiguration", TestTLSConfiguration)
	t.Run("RBACConfiguration", TestRBACConfiguration)
}

// TestRBACConfiguration tests RBAC filter configuration
func TestRBACConfiguration(t *testing.T) {
	// Create a test store
	s := CreateTestStore()

	// Add a test listener
	listenerName := "rbac-listener"
	listenerNamespace := "default"
	AddTestListener(s, listenerName, listenerNamespace)

	// Add a test cluster
	clusterName := "rbac-cluster"
	clusterNamespace := "default"
	AddTestCluster(s, clusterName, clusterNamespace)

	// Create a virtual service with RBAC configuration
	vs := CreateTestVirtualService()
	vs.ObjectMeta.Name = "rbac-vs"

	// Update the VirtualService to reference the listener
	vs.Spec.Listener = &v1alpha1.ResourceRef{
		Name:      listenerName,
		Namespace: &listenerNamespace,
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

	// Add a simple VirtualHost with a route
	vs.Spec.VirtualHost = &runtime.RawExtension{
		Raw: []byte(`{
			"domains": ["rbac.example.com"],
			"routes": [
				{
					"match": {
						"prefix": "/"
					},
					"route": {
						"cluster": "rbac-cluster"
					}
				}
			]
		}`),
	}

	// Add the virtual service to the store
	AddTestVirtualService(s, vs)

	// Run the comparison test
	CompareImplementationsTest(t, s, vs)
}
