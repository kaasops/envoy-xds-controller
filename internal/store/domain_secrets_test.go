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
	"github.com/kaasops/envoy-xds-controller/internal/testutil"
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

func TestParseCertificateNotAfter_NilSecret(t *testing.T) {
	result := ParseCertificateNotAfter(nil)
	assert.True(t, result.IsZero(), "Should return zero time for nil secret")
}

func TestParseCertificateNotAfter_MissingCertKey(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"other-key": []byte("data"),
		},
	}
	result := ParseCertificateNotAfter(secret)
	assert.True(t, result.IsZero(), "Should return zero time when tls.crt is missing")
}

func TestParseCertificateNotAfter_EmptyCertData(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.TLSCertKey: {},
		},
	}
	result := ParseCertificateNotAfter(secret)
	assert.True(t, result.IsZero(), "Should return zero time for empty certificate data")
}

func TestParseCertificateNotAfter_InvalidPEM(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.TLSCertKey: []byte("not valid PEM data"),
		},
	}
	result := ParseCertificateNotAfter(secret)
	assert.True(t, result.IsZero(), "Should return zero time for invalid PEM data")
}

func TestParseCertificateNotAfter_SingleCertificate(t *testing.T) {
	// Generate a certificate that expires in 365 days
	expectedExpiration := time.Now().Add(365 * 24 * time.Hour)
	testCert := testutil.GenerateTestCertificate(expectedExpiration)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.TLSCertKey: testCert,
		},
	}
	result := ParseCertificateNotAfter(secret)

	// Should parse the certificate and return the NotAfter date
	assert.False(t, result.IsZero(), "Should parse single certificate successfully")
	// Allow 1 second tolerance for timing differences
	assert.WithinDuration(t, expectedExpiration, result, time.Second, "Should return correct expiration time")
}

func TestParseCertificateNotAfter_CertificateChain_ReturnsMinimum(t *testing.T) {
	// End-entity certificate expires in 30 days (shorter validity)
	endEntityExpiration := time.Now().Add(30 * 24 * time.Hour)
	endEntityCert := testutil.GenerateTestCertificate(endEntityExpiration)

	// Intermediate certificate expires in 365 days (longer validity)
	intermediateExpiration := time.Now().Add(365 * 24 * time.Hour)
	intermediateCert := testutil.GenerateTestCertificate(intermediateExpiration)

	// Create a chain with the intermediate first (non-standard order)
	// This tests that we find the MINIMUM expiration regardless of order
	chainData := append(intermediateCert, '\n')
	chainData = append(chainData, endEntityCert...)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.TLSCertKey: chainData,
		},
	}
	result := ParseCertificateNotAfter(secret)

	// Should return the minimum (end-entity's 30 days), not the intermediate's 365 days
	assert.False(t, result.IsZero(), "Should parse certificate chain successfully")
	// Allow 1 second tolerance for timing differences
	assert.WithinDuration(t, endEntityExpiration, result, time.Second, "Should return minimum expiration from chain")
}

func TestParseCertificateNotAfter_SkipsNonCertificateBlocks(t *testing.T) {
	// Generate a certificate that expires in 365 days
	expectedExpiration := time.Now().Add(365 * 24 * time.Hour)
	cert := testutil.GenerateTestCertificate(expectedExpiration)

	// Add a private key block that should be skipped
	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBOgIBAAJBALoAQ0fM+W0GfzJMr3TyZhHVrWKxZ/A7JfEfSKQJWUxuOTuKHiLZ
xrSVqQ==
-----END RSA PRIVATE KEY-----`)

	// Mix private key and certificate (private key first)
	mixedData := append(privateKey, '\n')
	mixedData = append(mixedData, cert...)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.TLSCertKey: mixedData,
		},
	}
	result := ParseCertificateNotAfter(secret)

	// Should parse only the certificate, skipping the private key block
	assert.False(t, result.IsZero(), "Should parse certificate while skipping private key")
	// Allow 1 second tolerance for timing differences
	assert.WithinDuration(t, expectedExpiration, result, time.Second, "Should return certificate expiration time")
}

func TestDomainSecretsIndex_GetBestSecret_ValidityPriority(t *testing.T) {
	// Test the three-tier validity priority: valid > unknown > expired
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nnValid := helpers.NamespacedName{Namespace: "ns1", Name: "valid-secret"}
	nnUnknown := helpers.NamespacedName{Namespace: "ns2", Name: "unknown-secret"}
	nnExpired := helpers.NamespacedName{Namespace: "ns3", Name: "expired-secret"}

	secretValid := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "valid-secret", Namespace: "ns1"},
	}
	secretUnknown := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "unknown-secret", Namespace: "ns2"},
	}
	secretExpired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "expired-secret", Namespace: "ns3"},
	}
	secrets[nnValid] = secretValid
	secrets[nnUnknown] = secretUnknown
	secrets[nnExpired] = secretExpired

	// Add all three secrets with different validity states
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnValid,
		NotAfter:       time.Now().Add(24 * time.Hour), // valid
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnUnknown,
		NotAfter:       time.Time{}, // unknown (zero time = parsing failed)
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnExpired,
		NotAfter:       time.Now().Add(-24 * time.Hour), // expired
	})

	// Valid secret should be selected regardless of namespace preference
	result := idx.GetBestSecret("example.com", "ns3", secrets)
	assert.Equal(t, secretValid, result, "Valid secret should be preferred over unknown and expired")

	result = idx.GetBestSecret("example.com", "ns2", secrets)
	assert.Equal(t, secretValid, result, "Valid secret should be preferred even when unknown is in preferred namespace")
}

func TestDomainSecretsIndex_GetBestSecret_UnknownPreferredOverExpired(t *testing.T) {
	// Test that unknown validity (parsing failed) is preferred over expired
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nnUnknown := helpers.NamespacedName{Namespace: "ns1", Name: "unknown-secret"}
	nnExpired := helpers.NamespacedName{Namespace: "ns2", Name: "expired-secret"}

	secretUnknown := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "unknown-secret", Namespace: "ns1"},
	}
	secretExpired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "expired-secret", Namespace: "ns2"},
	}
	secrets[nnUnknown] = secretUnknown
	secrets[nnExpired] = secretExpired

	// Add unknown and expired secrets
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnUnknown,
		NotAfter:       time.Time{}, // unknown (zero time = parsing failed)
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nnExpired,
		NotAfter:       time.Now().Add(-24 * time.Hour), // expired
	})

	// Unknown should be preferred over expired
	result := idx.GetBestSecret("example.com", "ns2", secrets)
	assert.Equal(t, secretUnknown, result, "Unknown validity should be preferred over expired")
}

func TestDomainSecretsIndex_GetBestSecret_DefensiveNilCheck(t *testing.T) {
	// Test that GetBestSecret handles inconsistency between index and secrets map
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn1 := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	nn2 := helpers.NamespacedName{Namespace: "ns2", Name: "secret2"}

	// Only add secret2 to the secrets map (simulating inconsistent state)
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns2"},
	}
	secrets[nn2] = secret2

	// Add both to index
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn1,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn2,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	// Should return secret2 because secret1 doesn't exist in secrets map
	result := idx.GetBestSecret("example.com", "ns1", secrets)
	assert.Equal(t, secret2, result, "Should skip entries not in secrets map")
}

func TestDomainSecretsIndex_GetBestSecret_SingleSecretNilCheck(t *testing.T) {
	// Test fast path nil check for single secret
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}

	// Add to index but NOT to secrets map
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	// Should return nil because secret doesn't exist in secrets map
	result := idx.GetBestSecret("example.com", "ns1", secrets)
	assert.Nil(t, result, "Should return nil when single secret is not in secrets map")
}

func TestDomainSecretsIndex_GetBestSecret_AllSecretsNilInMap(t *testing.T) {
	// Test that GetBestSecret returns nil when all indexed secrets are missing from map
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn1 := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	nn2 := helpers.NamespacedName{Namespace: "ns2", Name: "secret2"}

	// Add both to index but neither to secrets map
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn1,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})
	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn2,
		NotAfter:       time.Now().Add(24 * time.Hour),
	})

	// Should return nil because no secrets exist in secrets map
	result := idx.GetBestSecret("example.com", "ns1", secrets)
	assert.Nil(t, result, "Should return nil when all indexed secrets are missing from map")
}

// Tests for GetBestSecretWithValidity
func TestDomainSecretsIndex_GetBestSecretWithValidity_Valid(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secrets[nn] = secret

	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn,
		NotAfter:       time.Now().Add(24 * time.Hour), // valid
	})

	resultSecret, validity := idx.GetBestSecretWithValidity("example.com", "ns1", secrets)
	assert.Equal(t, secret, resultSecret)
	assert.Equal(t, validityValid, validity)
}

func TestDomainSecretsIndex_GetBestSecretWithValidity_Expired(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secrets[nn] = secret

	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn,
		NotAfter:       time.Now().Add(-24 * time.Hour), // expired
	})

	resultSecret, validity := idx.GetBestSecretWithValidity("example.com", "ns1", secrets)
	assert.Equal(t, secret, resultSecret)
	assert.Equal(t, validityExpired, validity)
}

func TestDomainSecretsIndex_GetBestSecretWithValidity_Unknown(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	nn := helpers.NamespacedName{Namespace: "ns1", Name: "secret1"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
	}
	secrets[nn] = secret

	idx.Add("example.com", SecretDomainEntry{
		NamespacedName: nn,
		NotAfter:       time.Time{}, // unknown (zero time)
	})

	resultSecret, validity := idx.GetBestSecretWithValidity("example.com", "ns1", secrets)
	assert.Equal(t, secret, resultSecret)
	assert.Equal(t, validityUnknown, validity)
}

func TestDomainSecretsIndex_GetBestSecretWithValidity_NotFound(t *testing.T) {
	idx := NewDomainSecretsIndex(10)
	secrets := make(map[helpers.NamespacedName]*corev1.Secret)

	resultSecret, validity := idx.GetBestSecretWithValidity("nonexistent.com", "ns1", secrets)
	assert.Nil(t, resultSecret)
	assert.Equal(t, validityNotFound, validity)
}

// =============================================================================
// ValidateDomainPattern tests
// =============================================================================

func TestValidateDomainPattern(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		wantValid bool
	}{
		// Valid patterns
		{"exact domain", "example.com", true},
		{"subdomain", "api.example.com", true},
		{"deep subdomain", "a.b.c.example.com", true},
		{"valid wildcard", "*.example.com", true},
		{"valid wildcard subdomain", "*.api.example.com", true},

		// Invalid patterns
		{"empty", "", false},
		{"wildcard without dot", "*example.com", false},
		{"double wildcard", "**.example.com", false},
		{"wildcard in middle", "api.*.example.com", false},
		{"wildcard at end", "example.*", false},
		{"standalone asterisk", "*", false},
		{"asterisk dot only", "*.", false},
		{"multiple wildcards", "*.*.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomainPattern(tt.domain)
			if tt.wantValid {
				assert.Empty(t, err, "Expected valid pattern for %q", tt.domain)
			} else {
				assert.NotEmpty(t, err, "Expected invalid pattern for %q", tt.domain)
			}
		})
	}
}
