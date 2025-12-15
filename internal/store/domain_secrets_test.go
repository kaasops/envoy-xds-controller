/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func TestDomainSecretsIndex_AddAndRemove(t *testing.T) {
	idx := NewDomainSecretsIndex(10)

	nn1 := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	nn2 := helpers.NamespacedName{Namespace: "ns2", Name: "secret2"}

	// Add first secret
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn1,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	assert.Len(t, idx["example.com"], 1)

	// Add second secret for same domain
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn2,
		NotAfter:       time.Now().Add(48 * time.Hour),
	})

	assert.Len(t, idx["example.com"], 2)

	// Remove first secret
	idx.Remove("example.com", nn1)
	assert.Len(t, idx["example.com"], 1)

	// Verify second secret is still there
	_, exists := idx["example.com"][nn2]
	assert.True(t, exists)

	// Remove second secret - domain should be removed entirely
	idx.Remove("example.com", nn2)
	_, domainExists := idx["example.com"]
	assert.False(t, domainExists)
}

func TestDomainSecretsIndex_GetBestSecret_SingleSecret(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secrets[nn] = secret

	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	result := idx.GetBestSecret("example.com", "other-ns", secrets)
	assert.Equal(t, secret, result)
}

func TestDomainSecretsIndex_GetBestSecret_PreferSameNamespace(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn1 := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	nn2 := helpers.NamespacedName{Namespace: "ns2", Name: "secret2"}

	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns2"},
	}
	secrets[nn1] = secret1
	secrets[nn2] = secret2

	// Both secrets are valid
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn1,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn2,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	// Should prefer ns2 when preferredNamespace is ns2
	result := idx.GetBestSecret("example.com", "ns2", secrets)
	assert.Equal(t, secret2, result)

	// Should prefer ns1 when preferredNamespace is ns1
	result = idx.GetBestSecret("example.com", "ns1", secrets)
	assert.Equal(t, secret1, result)
}

func TestDomainSecretsIndex_GetBestSecret_PreferValidOverExpired(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn1 := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	nn2 := helpers.NamespacedName{Namespace: "ns2", Name: "secret2"}

	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns2"},
	}
	secrets[nn1] = secret1
	secrets[nn2] = secret2

	// ns1 secret is expired, ns2 is valid
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn1,
		NotAfter:       time.Now().Add(-24 * time.Hour), // expired
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn2,
		NotAfter:       time.Now().Add(24 * time.Hour), // valid
	})

	// Even though ns1 is preferred, ns2 should be returned because ns1 is expired
	result := idx.GetBestSecret("example.com", "ns1", secrets)
	assert.Equal(t, secret2, result)
}

func TestDomainSecretsIndex_GetBestSecret_FallbackAfterRemove(t *testing.T) {
	// This is the main bug scenario we're fixing
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn1 := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	nn2 := helpers.NamespacedName{Namespace: "ns2", Name: "secret2"}

	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns2"},
	}
	secrets[nn1] = secret1
	secrets[nn2] = secret2

	// Add both secrets
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn1,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn2,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	// Initial lookup should work
	result := idx.GetBestSecret("example.com", "default", secrets)
	assert.NotNil(t, result)

	// Remove first secret
	idx.Remove("example.com", nn1)
	delete(secrets, nn1)

	// Should still find second secret (THIS WAS THE BUG!)
	result = idx.GetBestSecret("example.com", "default", secrets)
	assert.NotNil(t, result, "Should find fallback secret after primary is deleted")
	assert.Equal(t, secret2, result)
}

func TestDomainSecretsIndex_GetAnySecret(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	// Add secrets with alphabetically ordered names
	nnA := helpers.NamespacedName{Namespace: "aaa", Name: "secret"}
	nnB := helpers.NamespacedName{Namespace: "bbb", Name: "secret"}

	secretA := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "aaa"},
	}
	secretB := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "bbb"},
	}
	secrets[nnA] = secretA
	secrets[nnB] = secretB

	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnA,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnB,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	// GetAnySecret should return alphabetically first (aaa)
	result := idx.GetAnySecret("example.com", secrets)
	assert.Equal(t, secretA, result)
}
