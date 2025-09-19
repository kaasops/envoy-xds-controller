package resbuilder_v2

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// createTestVirtualService creates a basic VirtualService for benchmarking
func createTestVirtualService() *v1alpha1.VirtualService {
	// Create a basic virtual host configuration
	virtualHost := &routev3.VirtualHost{
		Name:    "test-host",
		Domains: []string{"example.com"},
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

	virtualHostRaw, _ := protoutil.Marshaler.Marshal(virtualHost)

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
				VirtualHost: &runtime.RawExtension{
					Raw: virtualHostRaw,
				},
			},
		},
	}
}

// makeListenerCR creates a test listener CR with given port
func makeListenerCR() *v1alpha1.Listener {
	l := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "127.0.0.1", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 8080}}}},
	}
	b, _ := protoutil.Marshaler.Marshal(l)
	return &v1alpha1.Listener{
		TypeMeta:   metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "Listener"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-listener"},
		Spec:       &runtime.RawExtension{Raw: b},
	}
}

// createTestStore creates a minimal store for benchmarking with required test data
func createTestStore() *store.Store {
	store := store.New()
	// Add test listener required by the VirtualService
	listener := makeListenerCR()
	store.SetListener(listener)

	// Add test cluster required by the VirtualHost routes
	testCluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: &runtime.RawExtension{
			Raw: []byte(`{"name":"test-cluster","connect_timeout":"5s","type":"STATIC","load_assignment":{"cluster_name":"test-cluster","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"127.0.0.1","port_value":8080}}}}]}]}}`),
		},
	}
	store.SetCluster(testCluster)

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

	filtersBuilder := filters.NewBuilder(store)

	for i := 0; i < b.N; i++ {
		_, err := filtersBuilder.BuildHTTPFilters(vs)
		if err != nil {
			b.Fatalf("BuildHTTPFilters failed: %v", err)
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
		_ = utils.FindClusterNames(testData, "cluster")
	}
}

// BenchmarkExtractClusterNamesFromRoute benchmarks the optimized direct route traversal
func BenchmarkExtractClusterNamesFromRoute(b *testing.B) {
	// Create test route with cluster action
	route := &routev3.Route{
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_Cluster{
					Cluster: "test-cluster",
				},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = utils.ExtractClusterNamesFromRoute(route)
	}
}

// BenchmarkExtractClusterNamesFromRouteWeighted benchmarks weighted cluster extraction
func BenchmarkExtractClusterNamesFromRouteWeighted(b *testing.B) {
	// Create test route with weighted clusters
	route := &routev3.Route{
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_WeightedClusters{
					WeightedClusters: &routev3.WeightedCluster{
						Clusters: []*routev3.WeightedCluster_ClusterWeight{
							{Name: "cluster1", Weight: &wrapperspb.UInt32Value{Value: 50}},
							{Name: "cluster2", Weight: &wrapperspb.UInt32Value{Value: 50}},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = utils.ExtractClusterNamesFromRoute(route)
	}
}
