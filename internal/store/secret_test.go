package store

import (
	"testing"

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
