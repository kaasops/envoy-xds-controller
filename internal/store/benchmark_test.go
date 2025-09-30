package store

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	nsDefault    = "default"
	nsProduction = "production"
	nsStaging    = "staging"
)

// Benchmark baseline LegacyStore performance

func BenchmarkLegacyStore_Copy(b *testing.B) {
	store := New()

	// Populate with production-scale data: 10k secrets, 200 VS, 50 other resources
	populateStoreProductionSimple(store)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = store.Copy()
	}
}

func BenchmarkLegacyStore_SetVirtualService(b *testing.B) {
	store := New()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vs",
				Namespace: nsDefault,
				UID:       types.UID("uid-test"),
			},
		}
		store.SetVirtualService(vs)
	}
}

func BenchmarkLegacyStore_GetVirtualServiceByUID(b *testing.B) {
	store := New()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vs-" + string(rune(i+65)),
				Namespace: nsDefault,
				UID:       types.UID("uid-" + string(rune(i+65))),
			},
		}
		store.SetVirtualService(vs)
	}

	targetUID := "uid-" + string(rune(500+65))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = store.GetVirtualServiceByUID(targetUID)
	}
}

func BenchmarkLegacyStore_FillFromKubernetes_Simulation(b *testing.B) {
	store := New()

	// Simulate what FillFromKubernetes would do
	createResources := func() {
		// Create 100 VirtualServices (simplified to avoid UnmarshalV3 issues)
		for i := 0; i < 100; i++ {
			vs := &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vs-" + string(rune(i+65)),
					Namespace: nsDefault,
					UID:       types.UID("vs-uid-" + string(rune(i+65))),
				},
			}
			store.SetVirtualService(vs)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store = New() // Reset store
		createResources()
	}
}

func BenchmarkLegacyStore_ConcurrentReads(b *testing.B) {
	store := New()

	// Pre-populate
	for i := 0; i < 100; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vs-" + string(rune(i+65)),
				Namespace: nsDefault,
				UID:       types.UID("uid-" + string(rune(i+65))),
			},
		}
		store.SetVirtualService(vs)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.MapVirtualServices()
		}
	})
}

func BenchmarkLegacyStore_MemoryFootprint(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store := New()

		// Add production scale resources
		populateStoreProductionSimple(store)

		// Force allocation tracking
		_ = store.Copy()
	}
}

// Benchmark RealOptimizedStore vs LegacyStore

func BenchmarkOptimizedStore_Copy(b *testing.B) {
	store := NewOptimizedStore().(*OptimizedStore)

	// Populate with production-scale data: 10k secrets, 200 VS, 50 other resources
	populateOptimizedStoreProduction(store)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = store.Copy()
	}
}

func BenchmarkOptimizedStore_SetVirtualService(b *testing.B) {
	store := NewOptimizedStore()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vs",
				Namespace: nsDefault,
				UID:       types.UID("uid-test"),
			},
		}
		store.SetVirtualService(vs)
	}
}

func BenchmarkOptimizedStore_GetVirtualServiceByUID(b *testing.B) {
	store := NewOptimizedStore()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vs-" + string(rune(i+65)),
				Namespace: nsDefault,
				UID:       types.UID("uid-" + string(rune(i+65))),
			},
		}
		store.SetVirtualService(vs)
	}

	targetUID := "uid-" + string(rune(500+65))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = store.GetVirtualServiceByUID(targetUID)
	}
}

func BenchmarkOptimizedStore_ConcurrentReads(b *testing.B) {
	store := NewOptimizedStore()

	// Pre-populate
	for i := 0; i < 100; i++ {
		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vs-" + string(rune(i+65)),
				Namespace: nsDefault,
				UID:       types.UID("uid-" + string(rune(i+65))),
			},
		}
		store.SetVirtualService(vs)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.MapVirtualServices()
		}
	})
}

func BenchmarkOptimizedStore_MemoryFootprint(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store := NewOptimizedStore().(*OptimizedStore)

		// Add production scale resources
		populateOptimizedStoreProduction(store)

		// Force allocation tracking
		_ = store.Copy()
	}
}

// Helper functions for production-scale data population

// populateStoreProductionSimple fills LegacyStore with production-scale data without problematic clusters:
// 10,000 secrets, 200 VirtualServices, 50 of other resources (no clusters due to UnmarshalV3 issues)
func populateStoreProductionSimple(store *LegacyStore) {
	// Add 10,000 secrets with realistic names
	for i := 0; i < 10000; i++ {
		namespace := nsDefault
		if i%10 == 0 {
			namespace = "kube-system"
		} else if i%15 == 0 {
			namespace = "istio-system"
		} else if i%7 == 0 {
			namespace = nsProduction
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("secret-%05d", i),
				Namespace: namespace,
				UID:       types.UID(fmt.Sprintf("secret-uid-%05d", i)),
			},
			Data: map[string][]byte{
				"tls.crt": []byte("cert-data-" + strconv.Itoa(i)),
				"tls.key": []byte("key-data-" + strconv.Itoa(i)),
			},
		}
		store.SetSecret(secret)
	}

	// Add 200 VirtualServices
	for i := 0; i < 200; i++ {
		namespace := nsDefault
		if i%5 == 0 {
			namespace = nsProduction
		} else if i%8 == 0 {
			namespace = nsStaging
		}

		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("vs-%03d", i),
				Namespace: namespace,
				UID:       types.UID(fmt.Sprintf("vs-uid-%03d", i)),
			},
		}
		store.SetVirtualService(vs)
	}

	// Add 50 Listeners
	for i := 0; i < 50; i++ {
		listener := &v1alpha1.Listener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("listener-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("listener-uid-%02d", i)),
			},
		}
		store.SetListener(listener)
	}

	// Skip clusters for LegacyStore due to UnmarshalV3 issues in benchmark

	// Add 50 Routes
	for i := 0; i < 50; i++ {
		route := &v1alpha1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("route-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("route-uid-%02d", i)),
			},
		}
		store.SetRoute(route)
	}

	// Add 50 HttpFilters
	for i := 0; i < 50; i++ {
		httpFilter := &v1alpha1.HttpFilter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("httpfilter-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("httpfilter-uid-%02d", i)),
			},
		}
		store.SetHTTPFilter(httpFilter)
	}

	// Add 50 Policies
	for i := 0; i < 50; i++ {
		policy := &v1alpha1.Policy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("policy-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("policy-uid-%02d", i)),
			},
		}
		store.SetPolicy(policy)
	}

	// Add 50 AccessLogConfigs
	for i := 0; i < 50; i++ {
		accessLog := &v1alpha1.AccessLogConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("accesslog-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("accesslog-uid-%02d", i)),
			},
		}
		store.SetAccessLog(accessLog)
	}

	// Add 50 Tracings
	for i := 0; i < 50; i++ {
		tracing := &v1alpha1.Tracing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("tracing-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("tracing-uid-%02d", i)),
			},
		}
		store.SetTracing(tracing)
	}
}

// populateOptimizedStoreProduction fills OptimizedStore with the same production-scale data
func populateOptimizedStoreProduction(store *OptimizedStore) {
	// Add 10,000 secrets with realistic names
	for i := 0; i < 10000; i++ {
		namespace := nsDefault
		if i%10 == 0 {
			namespace = "kube-system"
		} else if i%15 == 0 {
			namespace = "istio-system"
		} else if i%7 == 0 {
			namespace = nsProduction
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("secret-%05d", i),
				Namespace: namespace,
				UID:       types.UID(fmt.Sprintf("secret-uid-%05d", i)),
			},
			Data: map[string][]byte{
				"tls.crt": []byte("cert-data-" + strconv.Itoa(i)),
				"tls.key": []byte("key-data-" + strconv.Itoa(i)),
			},
		}
		store.SetSecret(secret)
	}

	// Add 200 VirtualServices
	for i := 0; i < 200; i++ {
		namespace := nsDefault
		if i%5 == 0 {
			namespace = nsProduction
		} else if i%8 == 0 {
			namespace = nsStaging
		}

		vs := &v1alpha1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("vs-%03d", i),
				Namespace: namespace,
				UID:       types.UID(fmt.Sprintf("vs-uid-%03d", i)),
			},
		}
		store.SetVirtualService(vs)
	}

	// Add 50 Listeners
	for i := 0; i < 50; i++ {
		listener := &v1alpha1.Listener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("listener-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("listener-uid-%02d", i)),
			},
		}
		store.SetListener(listener)
	}

	// Add 50 Clusters
	for i := 0; i < 50; i++ {
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("cluster-uid-%02d", i)),
			},
		}
		store.SetCluster(cluster)
	}

	// Add 50 Routes
	for i := 0; i < 50; i++ {
		route := &v1alpha1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("route-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("route-uid-%02d", i)),
			},
		}
		store.SetRoute(route)
	}

	// Add 50 HttpFilters
	for i := 0; i < 50; i++ {
		httpFilter := &v1alpha1.HttpFilter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("httpfilter-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("httpfilter-uid-%02d", i)),
			},
		}
		store.SetHTTPFilter(httpFilter)
	}

	// Add 50 Policies
	for i := 0; i < 50; i++ {
		policy := &v1alpha1.Policy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("policy-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("policy-uid-%02d", i)),
			},
		}
		store.SetPolicy(policy)
	}

	// Add 50 AccessLogConfigs
	for i := 0; i < 50; i++ {
		accessLog := &v1alpha1.AccessLogConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("accesslog-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("accesslog-uid-%02d", i)),
			},
		}
		store.SetAccessLog(accessLog)
	}

	// Add 50 Tracings
	for i := 0; i < 50; i++ {
		tracing := &v1alpha1.Tracing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("tracing-%02d", i),
				Namespace: nsDefault,
				UID:       types.UID(fmt.Sprintf("tracing-uid-%02d", i)),
			},
		}
		store.SetTracing(tracing)
	}
}

// String Pool Benchmarks
func BenchmarkStringPool_Intern(b *testing.B) {
	pool := NewStringPool()

	strings := []string{
		nsDefault, "kube-system", "istio-system",
		"my-service", "my-app", "frontend", "backend",
		"uid-12345", "uid-67890", "uid-abcdef",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, s := range strings {
			pool.Intern(s)
		}
	}
}

func BenchmarkStringPool_InternUID(b *testing.B) {
	pool := NewStringPool()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		uid := "uid-" + string(rune(i%1000+65))
		pool.InternUID(uid)
	}
}
