package store

import (
	"fmt"
	"sync"
	"testing"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestMapDomainSecrets_ReturnsPointers verifies that MapDomainSecrets returns pointers
func TestMapDomainSecrets_ReturnsPointers(t *testing.T) {
	tests := []struct {
		name  string
		store Store
	}{
		{"OptimizedStore", NewOptimizedStore()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test secrets
			secret1 := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1",
					Namespace: "default",
					Annotations: map[string]string{
						"envoy.kaasops.io/domains": "domain1.com",
					},
				},
				Data: map[string][]byte{
					"tls.crt": []byte("cert1"),
					"tls.key": []byte("key1"),
				},
			}

			secret2 := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret2",
					Namespace: "default",
					Annotations: map[string]string{
						"envoy.kaasops.io/domains": "domain2.com",
					},
				},
				Data: map[string][]byte{
					"tls.crt": []byte("cert2"),
					"tls.key": []byte("key2"),
				},
			}

			tt.store.SetSecret(secret1)
			tt.store.SetSecret(secret2)

			// Get domain secrets map
			domainSecrets := tt.store.MapDomainSecrets()

			// Verify we got pointers
			assert.NotNil(t, domainSecrets)
			assert.Len(t, domainSecrets, 2)

			// Verify correct data
			s1, ok := domainSecrets["domain1.com"]
			assert.True(t, ok, "domain1.com should exist")
			assert.NotNil(t, s1, "domain1.com secret should not be nil")
			assert.Equal(t, "secret1", s1.Name)
			assert.Equal(t, []byte("cert1"), s1.Data["tls.crt"])

			s2, ok := domainSecrets["domain2.com"]
			assert.True(t, ok, "domain2.com should exist")
			assert.NotNil(t, s2, "domain2.com secret should not be nil")
			assert.Equal(t, "secret2", s2.Name)
			assert.Equal(t, []byte("cert2"), s2.Data["tls.crt"])

			// Verify that each domain has a unique pointer
			assert.NotSame(t, s1, s2, "pointers should be different")
		})
	}
}

// TestMapDomainSecrets_OptimizedStore_SamePointers verifies OptimizedStore returns same pointers for performance
func TestMapDomainSecrets_OptimizedStore_SamePointers(t *testing.T) {
	store := NewOptimizedStore()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "test.com",
			},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}

	store.SetSecret(secret)

	// Get two maps
	map1 := store.MapDomainSecrets()
	map2 := store.MapDomainSecrets()

	// Verify both have the data
	assert.Len(t, map1, 1)
	assert.Len(t, map2, 1)

	// OptimizedStore returns same pointer for performance (secrets are immutable in store)
	assert.Same(t, map1["test.com"], map2["test.com"],
		"OptimizedStore should return same secret pointer for performance")
}

// TestMapDomainSecretsForNamespace_PrefersMatchingNamespace verifies namespace preference
func TestMapDomainSecretsForNamespace_PrefersMatchingNamespace(t *testing.T) {
	store := NewOptimizedStore()

	// Create two secrets for same domain in different namespaces
	secretNs1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-secret",
			Namespace: "ns1",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("cert-ns1"),
			"tls.key": []byte("key-ns1"),
		},
	}

	secretNs2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-secret",
			Namespace: "ns2",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("cert-ns2"),
			"tls.key": []byte("key-ns2"),
		},
	}

	store.SetSecret(secretNs1)
	store.SetSecret(secretNs2)

	// Get domain secrets with ns1 preference
	mapNs1 := store.MapDomainSecretsForNamespace("ns1")
	assert.Len(t, mapNs1, 1)
	assert.Equal(t, "ns1", mapNs1["example.com"].Namespace,
		"Should prefer secret from ns1 when preferredNamespace is ns1")

	// Get domain secrets with ns2 preference
	mapNs2 := store.MapDomainSecretsForNamespace("ns2")
	assert.Len(t, mapNs2, 1)
	assert.Equal(t, "ns2", mapNs2["example.com"].Namespace,
		"Should prefer secret from ns2 when preferredNamespace is ns2")

	// Get domain secrets with no preference (should return alphabetically first)
	mapNoPreference := store.MapDomainSecretsForNamespace("")
	assert.Len(t, mapNoPreference, 1)
	assert.Equal(t, "ns1", mapNoPreference["example.com"].Namespace,
		"Should return alphabetically first namespace when no preference")
}

// TestMapDomainSecretsForNamespace_FallbackAfterDelete verifies fallback works after primary secret deletion
func TestMapDomainSecretsForNamespace_FallbackAfterDelete(t *testing.T) {
	store := NewOptimizedStore()

	// Create two secrets for same domain in different namespaces
	secretNs1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-secret",
			Namespace: "ns1",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("cert-ns1"),
			"tls.key": []byte("key-ns1"),
		},
	}

	secretNs2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-secret",
			Namespace: "ns2",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("cert-ns2"),
			"tls.key": []byte("key-ns2"),
		},
	}

	store.SetSecret(secretNs1)
	store.SetSecret(secretNs2)

	// Verify both secrets are indexed
	mapBefore := store.MapDomainSecretsForNamespace("ns1")
	assert.Len(t, mapBefore, 1)
	assert.Equal(t, "ns1", mapBefore["example.com"].Namespace)

	// Delete the first secret
	store.DeleteSecret(helpers.NamespacedName{Namespace: secretNs1.Namespace, Name: secretNs1.Name})

	// Should fallback to ns2 secret
	mapAfter := store.MapDomainSecretsForNamespace("ns1")
	assert.Len(t, mapAfter, 1, "Should still have a secret for example.com after deletion")
	assert.Equal(t, "ns2", mapAfter["example.com"].Namespace,
		"Should fallback to ns2 secret after ns1 secret is deleted")
}

// TestGetDomainSecretForNamespace_DirectLookup verifies direct domain secret lookup with namespace preference
func TestGetDomainSecretForNamespace_DirectLookup(t *testing.T) {
	store := NewOptimizedStore()

	// Create two secrets for same domain in different namespaces
	secretNs1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-secret",
			Namespace: "ns1",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("cert-ns1"),
			"tls.key": []byte("key-ns1"),
		},
	}

	secretNs2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-secret",
			Namespace: "ns2",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("cert-ns2"),
			"tls.key": []byte("key-ns2"),
		},
	}

	store.SetSecret(secretNs1)
	store.SetSecret(secretNs2)

	// Lookup with ns1 preference
	secret := store.GetDomainSecretForNamespace("example.com", "ns1")
	assert.NotNil(t, secret)
	assert.Equal(t, "ns1", secret.Namespace)

	// Lookup with ns2 preference
	secret = store.GetDomainSecretForNamespace("example.com", "ns2")
	assert.NotNil(t, secret)
	assert.Equal(t, "ns2", secret.Namespace)

	// Lookup non-existent domain
	secret = store.GetDomainSecretForNamespace("nonexistent.com", "ns1")
	assert.Nil(t, secret)
}

// TestGetDomainSecretWithWildcardFallback_ExactValid tests that valid exact cert is preferred
func TestGetDomainSecretWithWildcardFallback_ExactValid(t *testing.T) {
	store := NewOptimizedStore()

	validCert := testutil.GenerateValidCertificate()
	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", validCert))
	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", validCert))

	// Should return exact cert (both valid, exact is more specific)
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "exact-cert", result.Name)
}

// TestGetDomainSecretWithWildcardFallback_ExactExpiredWildcardValid tests fallback to wildcard
func TestGetDomainSecretWithWildcardFallback_ExactExpiredWildcardValid(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", testutil.GenerateExpiredCertificate()))
	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", testutil.GenerateValidCertificate()))

	// Should return wildcard cert (exact is expired, wildcard is valid)
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "wildcard-cert", result.Name, "Should fallback to valid wildcard when exact is expired")
}

// TestGetDomainSecretWithWildcardFallback_ExactUnknownWildcardValid tests fallback when exact is unparseable
func TestGetDomainSecretWithWildcardFallback_ExactUnknownWildcardValid(t *testing.T) {
	store := NewOptimizedStore()

	// Create unparseable exact cert (invalid data - can't use NewTLSSecret helper)
	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", []byte("not a valid certificate")))
	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", testutil.GenerateValidCertificate()))

	// Should return wildcard cert (exact is unknown/unparseable, wildcard is valid)
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "wildcard-cert", result.Name, "Should fallback to valid wildcard when exact is unparseable")
}

// TestGetDomainSecretWithWildcardFallback_BothExpired tests that exact is returned when both expired
func TestGetDomainSecretWithWildcardFallback_BothExpired(t *testing.T) {
	store := NewOptimizedStore()

	expiredCert := testutil.GenerateExpiredCertificate()
	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", expiredCert))
	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", expiredCert))

	// Should return exact cert (both expired, exact is more specific)
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "exact-cert", result.Name, "Should return exact when both are expired")
}

// TestGetDomainSecretWithWildcardFallback_ValidExactExpiredWildcard tests that valid exact
// is returned even when expired wildcard exists. This is the expected behavior because
// exact match takes precedence when it's valid.
func TestGetDomainSecretWithWildcardFallback_ValidExactExpiredWildcard(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", testutil.GenerateValidCertificate()))
	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", testutil.GenerateExpiredCertificate()))

	// Should return valid exact cert, ignoring expired wildcard
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "exact-cert", result.Name, "Should return valid exact cert when wildcard is expired")
}

// TestGetDomainSecretWithWildcardFallback_OnlyWildcard tests when only wildcard exists
func TestGetDomainSecretWithWildcardFallback_OnlyWildcard(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", testutil.GenerateValidCertificate()))

	// Should return wildcard cert
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "wildcard-cert", result.Name)
}

// TestGetDomainSecretWithWildcardFallback_OnlyExact tests when only exact exists
func TestGetDomainSecretWithWildcardFallback_OnlyExact(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", testutil.GenerateValidCertificate()))

	// Should return exact cert
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "exact-cert", result.Name)
}

// TestGetDomainSecretWithWildcardFallback_NotFound tests when no cert exists
func TestGetDomainSecretWithWildcardFallback_NotFound(t *testing.T) {
	store := NewOptimizedStore()

	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.Nil(t, result)
}

// TestGetDomainSecretWithWildcardFallback_MultiLevelSubdomain tests that multi-level
// subdomains only fallback to immediate parent wildcard (*.api.example.com),
// NOT to deeper wildcard (*.example.com).
func TestGetDomainSecretWithWildcardFallback_MultiLevelSubdomain(t *testing.T) {
	store := NewOptimizedStore()

	validCert := testutil.GenerateValidCertificate()
	expiredCert := testutil.GenerateExpiredCertificate()
	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "sub.api.example.com", expiredCert))
	store.SetSecret(testutil.NewTLSSecret("ns1", "immediate-wildcard", "*.api.example.com", validCert))
	store.SetSecret(testutil.NewTLSSecret("ns1", "deeper-wildcard", "*.example.com", validCert))

	// Should fallback to immediate parent wildcard (*.api.example.com)
	result := store.GetDomainSecretWithWildcardFallback("sub.api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "immediate-wildcard", result.Name,
		"Should fallback to immediate parent wildcard *.api.example.com, not *.example.com")
}

// TestGetDomainSecretWithWildcardFallback_MultiLevelSubdomain_NoImmediateWildcard tests
// that when immediate parent wildcard is missing, deeper wildcard is NOT used.
func TestGetDomainSecretWithWildcardFallback_MultiLevelSubdomain_NoImmediateWildcard(t *testing.T) {
	store := NewOptimizedStore()

	expiredCert := testutil.GenerateExpiredCertificate()
	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "sub.api.example.com", expiredCert))
	store.SetSecret(testutil.NewTLSSecret("ns1", "deeper-wildcard", "*.example.com", testutil.GenerateValidCertificate()))

	// Should return expired exact cert because *.example.com is NOT the immediate parent
	// of sub.api.example.com (the immediate parent would be *.api.example.com)
	result := store.GetDomainSecretWithWildcardFallback("sub.api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "exact-cert", result.Name,
		"Should return expired exact cert when immediate parent wildcard is missing")
}

// =============================================================================
// Cross-namespace fallback tests
// =============================================================================

// TestGetDomainSecretWithWildcardFallback_CrossNamespace tests that fallback works
// across namespaces: expired exact in ns1, valid wildcard in ns2
func TestGetDomainSecretWithWildcardFallback_CrossNamespace(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", testutil.GenerateExpiredCertificate()))
	store.SetSecret(testutil.NewTLSSecret("ns2", "wildcard-cert", "*.example.com", testutil.GenerateValidCertificate()))

	// VirtualService in ns1 should get valid wildcard from ns2 (validity > namespace)
	result := store.GetDomainSecretWithWildcardFallback("api.example.com", "ns1")
	assert.NotNil(t, result)
	assert.Equal(t, "wildcard-cert", result.Name,
		"Should fallback to valid wildcard in ns2 when exact in ns1 is expired")
	assert.Equal(t, "ns2", result.Namespace,
		"Wildcard should be from ns2 even though preferred namespace is ns1")
}

// =============================================================================
// SecretLookupResult info tests
// =============================================================================

// TestGetDomainSecretWithWildcardFallbackInfo_ExactValid tests info result for valid exact cert
func TestGetDomainSecretWithWildcardFallbackInfo_ExactValid(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", testutil.GenerateValidCertificate()))

	result := store.GetDomainSecretWithWildcardFallbackInfo("api.example.com", "ns1")

	assert.NotNil(t, result.Secret)
	assert.Equal(t, "exact-cert", result.Secret.Name)
	assert.False(t, result.UsedWildcard, "Should not use wildcard when exact is valid")
	assert.Empty(t, result.FallbackReason, "No fallback reason when exact is used")
	assert.Equal(t, "ns1/exact-cert", result.ExactSecretName)
	assert.Equal(t, "valid", result.ExactValidity)
}

// TestGetDomainSecretWithWildcardFallbackInfo_WildcardFallback tests info result for fallback
func TestGetDomainSecretWithWildcardFallbackInfo_WildcardFallback(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "exact-cert", "api.example.com", testutil.GenerateExpiredCertificate()))
	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", testutil.GenerateValidCertificate()))

	result := store.GetDomainSecretWithWildcardFallbackInfo("api.example.com", "ns1")

	assert.NotNil(t, result.Secret)
	assert.Equal(t, "wildcard-cert", result.Secret.Name)
	assert.True(t, result.UsedWildcard, "Should use wildcard when exact is expired")
	assert.Equal(t, "expired", result.FallbackReason, "Fallback reason should be 'expired'")
	assert.Equal(t, "ns1/exact-cert", result.ExactSecretName)
	assert.Equal(t, "expired", result.ExactValidity)
}

// TestGetDomainSecretWithWildcardFallbackInfo_WildcardOnly tests info result when only wildcard exists
func TestGetDomainSecretWithWildcardFallbackInfo_WildcardOnly(t *testing.T) {
	store := NewOptimizedStore()

	store.SetSecret(testutil.NewTLSSecret("ns1", "wildcard-cert", "*.example.com", testutil.GenerateValidCertificate()))

	result := store.GetDomainSecretWithWildcardFallbackInfo("api.example.com", "ns1")

	assert.NotNil(t, result.Secret)
	assert.Equal(t, "wildcard-cert", result.Secret.Name)
	assert.True(t, result.UsedWildcard, "Should indicate wildcard was used")
	assert.Empty(t, result.FallbackReason, "No fallback reason when exact didn't exist")
	assert.Empty(t, result.ExactSecretName, "No exact secret name when it didn't exist")
	assert.Equal(t, "not_found", result.ExactValidity)
}

// TestGetDomainSecretWithWildcardFallbackInfo_NotFound tests info result when no cert exists
func TestGetDomainSecretWithWildcardFallbackInfo_NotFound(t *testing.T) {
	store := NewOptimizedStore()

	result := store.GetDomainSecretWithWildcardFallbackInfo("api.example.com", "ns1")

	assert.Nil(t, result.Secret)
	assert.False(t, result.UsedWildcard)
	assert.Empty(t, result.FallbackReason)
	assert.Empty(t, result.ExactSecretName)
	assert.Equal(t, "not_found", result.ExactValidity)
}

// =============================================================================
// Concurrent access tests
// =============================================================================

// TestDomainSecretsIndex_ConcurrentAccess verifies that concurrent operations
// on the store are safe when using OptimizedStore's mutex protection.
// This test should be run with -race flag to detect data races.
func TestDomainSecretsIndex_ConcurrentAccess(t *testing.T) {
	store := NewOptimizedStore()

	// Create test certificates
	validCert := testutil.GenerateValidCertificate()

	// Number of concurrent goroutines
	numGoroutines := 10
	numOperations := 100

	// Use a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // writers, readers, and deleters

	// Channel to signal test start (ensures all goroutines start together)
	start := make(chan struct{})

	// Writer goroutines - add secrets
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			<-start // Wait for start signal
			for j := 0; j < numOperations; j++ {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("secret-%d-%d", id, j),
						Namespace: fmt.Sprintf("ns%d", id%3),
						Annotations: map[string]string{
							"envoy.kaasops.io/domains": fmt.Sprintf("domain%d-%d.example.com", id, j),
						},
					},
					Data: map[string][]byte{
						"tls.crt": validCert,
						"tls.key": []byte("key"),
					},
				}
				store.SetSecret(secret)
			}
		}(i)
	}

	// Reader goroutines - read domain secrets
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			<-start // Wait for start signal
			for j := 0; j < numOperations; j++ {
				// Try to get secrets for various domains
				domain := fmt.Sprintf("domain%d-%d.example.com", id, j%numOperations)
				_ = store.GetDomainSecretWithWildcardFallback(domain, fmt.Sprintf("ns%d", id%3))

				// Also test MapDomainSecrets
				_ = store.MapDomainSecrets()
			}
		}(i)
	}

	// Deleter goroutines - delete some secrets
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			<-start                                // Wait for start signal
			for j := 0; j < numOperations/2; j++ { // Delete fewer than we add
				nn := helpers.NamespacedName{
					Namespace: fmt.Sprintf("ns%d", id%3),
					Name:      fmt.Sprintf("secret-%d-%d", id, j),
				}
				store.DeleteSecret(nn)
			}
		}(i)
	}

	// Start all goroutines simultaneously
	close(start)

	// Wait for all operations to complete
	wg.Wait()

	// Verify store is in a consistent state
	secrets := store.MapDomainSecrets()
	assert.NotNil(t, secrets, "MapDomainSecrets should return non-nil after concurrent operations")
}

// TestDomainSecretsIndex_ConcurrentReadWrite verifies that concurrent reads
// and writes don't corrupt the index structure.
func TestDomainSecretsIndex_ConcurrentReadWrite(t *testing.T) {
	store := NewOptimizedStore()

	validCert := testutil.GenerateValidCertificate()

	// Pre-populate with some secrets
	for i := 0; i < 10; i++ {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("initial-secret-%d", i),
				Namespace: "default",
				Annotations: map[string]string{
					"envoy.kaasops.io/domains": fmt.Sprintf("initial%d.example.com", i),
				},
			},
			Data: map[string][]byte{
				"tls.crt": validCert,
				"tls.key": []byte("key"),
			},
		}
		store.SetSecret(secret)
	}

	var wg sync.WaitGroup
	numGoroutines := 20
	wg.Add(numGoroutines)

	// Mix of reads and writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if id%2 == 0 {
					// Writer
					secret := &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("dynamic-secret-%d-%d", id, j),
							Namespace: "default",
							Annotations: map[string]string{
								"envoy.kaasops.io/domains": fmt.Sprintf("dynamic%d-%d.example.com", id, j),
							},
						},
						Data: map[string][]byte{
							"tls.crt": validCert,
							"tls.key": []byte("key"),
						},
					}
					store.SetSecret(secret)
				} else {
					// Reader
					result := store.GetDomainSecretWithWildcardFallbackInfo(
						fmt.Sprintf("initial%d.example.com", j%10),
						"default",
					)
					// Result may or may not have a secret depending on timing
					_ = result
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify we can still read initial secrets
	for i := 0; i < 10; i++ {
		result := store.GetDomainSecretWithWildcardFallback(
			fmt.Sprintf("initial%d.example.com", i),
			"default",
		)
		assert.NotNil(t, result, "Initial secret %d should still exist", i)
	}
}
