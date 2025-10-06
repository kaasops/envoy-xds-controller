package store

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarkMapDomainSecrets_OptimizedStore benchmarks the optimized implementation
func BenchmarkMapDomainSecrets_OptimizedStore(b *testing.B) {
	store := NewOptimizedStore()

	// Populate store with test secrets
	for i := 0; i < 100; i++ {
		domain := fmt.Sprintf("domain-%d.example.com", i)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("secret-%d", i),
				Namespace: "default",
				Annotations: map[string]string{
					"envoy.kaasops.io/domains": domain,
				},
			},
			Data: map[string][]byte{
				"tls.crt": make([]byte, 1024),
				"tls.key": make([]byte, 1024),
			},
		}
		store.SetSecret(secret)
	}

	b.ResetTimer()
	b.ReportAllocs()

	var result map[string]*corev1.Secret
	for i := 0; i < b.N; i++ {
		result = store.MapDomainSecrets()
	}
	_ = result
}
