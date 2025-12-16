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

// BenchmarkParseCertificateNotAfter benchmarks certificate parsing overhead
func BenchmarkParseCertificateNotAfter(b *testing.B) {
	// Use a pre-generated test certificate for realistic benchmarking
	certPEM := benchTestCertPEM

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": []byte("dummy-key"),
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseCertificateNotAfter(secret)
	}
}

// BenchmarkSetSecret_WithCertParsing benchmarks SetSecret with real certificate parsing
func BenchmarkSetSecret_WithCertParsing(b *testing.B) {
	certPEM := benchTestCertPEM

	b.Run("with_domain_annotation", func(b *testing.B) {
		store := NewOptimizedStore()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("secret-%d", i),
					Namespace: "default",
					Annotations: map[string]string{
						"envoy.kaasops.io/domains": fmt.Sprintf("domain-%d.example.com", i),
					},
				},
				Data: map[string][]byte{
					"tls.crt": certPEM,
					"tls.key": []byte("dummy-key"),
				},
			}
			store.SetSecret(secret)
		}
	})

	b.Run("without_domain_annotation", func(b *testing.B) {
		store := NewOptimizedStore()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("secret-%d", i),
					Namespace: "default",
				},
				Data: map[string][]byte{
					"tls.crt": certPEM,
					"tls.key": []byte("dummy-key"),
				},
			}
			store.SetSecret(secret)
		}
	})
}

// benchTestCertPEM is a pre-generated self-signed certificate for benchmarking
var benchTestCertPEM = []byte(`-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQDU+pQ4P2dG5jANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC7
o5e7RvS3xfNbT1QuyLNdJCIHjLMFKlOoVHxDvO3FvOXfGCJLGkGgRtJyL+h8dN9w
pRGLBJhuNgJCNLCUP6E3X4E9LdZLi/QSEM6EEICSsxL4WUKw7FDLBq+p6yKBGRx0
hK4rF0JK/gYqvNhc7c9ZtqEQ3wShLgXQKULlJRLQN1P9gJOGaKP3aL1FARa5E8oA
Q3+8B1AUVQDRQM5xsoriPBauLYAJY1K3j+eIlJ7dS/sXpPq0Z0nb4NYOKH3Fr1FQ
tB+bsqFP3byP2JrCLHqnDqLh9FHlleF3wAoKP0/0Z9kNyD2zX3G3Z1lGJHYK3YqN
z1sMqkVnBEX1UbEwvQRXAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAISk0n3gXjuO
RwHBUVL8UKS1cXHBz3TdqmRcLKYLGNkWlDlRABnLVCqLJ1fCZBB6B4N8dQTKiOPQ
QCoX8nUTAPLxzL3TgXJPPGd9gLrxYYQVpBVdQA3vJPSgrIRcPBBbN0W5al1fPaLv
HH0TUXR7GQr5rj7DEBZF8xChVXhJKjFHKgPXTWaMNFCDVfktbVpdhGUV/qJPGg7w
uJLFQoS3H0YV7VYsrY8kBx1bPK3zdH1fPkQy9LkuG2T4rDFATU3hsBmq3SrdVp+K
lOvFQupWMXpHiYHFdgVFBDAi2T2LnzJiLhQ8ky6/2Y7BF7Z67yJLLVTwQAYdWBMw
KkkjG4LVMSA=
-----END CERTIFICATE-----`)
