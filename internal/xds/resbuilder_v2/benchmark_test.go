package resbuilder_v2

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// createTestVirtualService creates a basic VirtualService for benchmarking
func createTestVirtualService() *v1alpha1.VirtualService {
	return &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				Listener: &v1alpha1.ResourceRef{
					Name: "test-listener",
				},
			},
		},
	}
}

// makeListenerCR creates a test listener CR with given host:port
func makeListenerCR(ns, name, host string, port uint32) *v1alpha1.Listener {
	l := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: host, PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: port}}}},
	}
	b, _ := protoutil.Marshaler.Marshal(l)
	return &v1alpha1.Listener{
		TypeMeta:   metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "Listener"},
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       &runtime.RawExtension{Raw: b},
	}
}

// createTestStore creates a minimal store for benchmarking with required test data
func createTestStore() *store.Store {
	store := store.New()
	// Add test listener required by the VirtualService
	listener := makeListenerCR("default", "test-listener", "127.0.0.1", 8080)
	store.SetListener(listener)
	return store
}

// BenchmarkBuildResources benchmarks the main BuildResources function
func BenchmarkBuildResources(b *testing.B) {
	vs := createTestVirtualService()
	store := createTestStore()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := BuildResources(vs, store)
		if err != nil {
			b.Fatalf("BuildResources failed: %v", err)
		}
	}
}

// BenchmarkBuildResourcesMemory specifically focuses on memory allocations
func BenchmarkBuildResourcesMemory(b *testing.B) {
	vs := createTestVirtualService()
	store := createTestStore()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := BuildResources(vs, store)
		if err != nil {
			b.Fatalf("BuildResources failed: %v", err)
		}
	}
}

// BenchmarkBuildHTTPFilters benchmarks the buildHTTPFilters function
func BenchmarkBuildHTTPFilters(b *testing.B) {
	vs := createTestVirtualService()
	store := createTestStore()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := buildHTTPFilters(vs, store)
		if err != nil {
			b.Fatalf("buildHTTPFilters failed: %v", err)
		}
	}
}

// BenchmarkFindClusterNames benchmarks the findClusterNames function that uses JSON operations
func BenchmarkFindClusterNames(b *testing.B) {
	// Create test data structure
	testData := map[string]interface{}{
		"route": map[string]interface{}{
			"cluster": "test-cluster-1",
		},
		"routes": []map[string]interface{}{
			{"cluster": "test-cluster-2"},
			{"cluster": "test-cluster-3"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = findClusterNames(testData, "cluster")
	}
}

// BenchmarkFindSDSNames benchmarks the findSDSNames function
func BenchmarkFindSDSNames(b *testing.B) {
	// Create test data structure  
	testData := map[string]interface{}{
		"transport_socket": map[string]interface{}{
			"typed_config": map[string]interface{}{
				"common_tls_context": map[string]interface{}{
					"tls_certificate_sds_secret_configs": []map[string]interface{}{
						{"name": "test-sds-1"},
						{"name": "test-sds-2"},
					},
				},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = findSDSNames(testData, "name")
	}
}