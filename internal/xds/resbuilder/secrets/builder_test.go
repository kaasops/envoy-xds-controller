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

package secrets

import (
	"testing"
	"time"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

// =============================================================================
// GetTLSType tests
// =============================================================================

func TestGetTLSType_Nil(t *testing.T) {
	_, err := GetTLSType(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TLS config is nil")
}

func TestGetTLSType_NoConfig(t *testing.T) {
	tlsConfig := &v1alpha1.TlsConfig{}
	_, err := GetTLSType(tlsConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TLS configuration specified")
}

func TestGetTLSType_SecretRef(t *testing.T) {
	tlsConfig := &v1alpha1.TlsConfig{
		SecretRef: &v1alpha1.ResourceRef{
			Name: "my-secret",
		},
	}
	tlsType, err := GetTLSType(tlsConfig)
	assert.NoError(t, err)
	assert.Equal(t, "secretRef", tlsType)
}

func TestGetTLSType_AutoDiscovery(t *testing.T) {
	tlsConfig := &v1alpha1.TlsConfig{
		AutoDiscovery: boolPtr(true),
	}
	tlsType, err := GetTLSType(tlsConfig)
	assert.NoError(t, err)
	assert.Equal(t, "autoDiscovery", tlsType)
}

func TestGetTLSType_AutoDiscoveryFalse(t *testing.T) {
	// AutoDiscovery set to false should be treated as no config
	tlsConfig := &v1alpha1.TlsConfig{
		AutoDiscovery: boolPtr(false),
	}
	_, err := GetTLSType(tlsConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TLS configuration specified")
}

func TestGetTLSType_MultipleConfigs(t *testing.T) {
	tlsConfig := &v1alpha1.TlsConfig{
		SecretRef: &v1alpha1.ResourceRef{
			Name: "my-secret",
		},
		AutoDiscovery: boolPtr(true),
	}
	_, err := GetTLSType(tlsConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple TLS configuration types specified")
}

// =============================================================================
// GetSecretNameToDomains tests - SecretRef mode
// =============================================================================

func TestGetSecretNameToDomains_SecretRef_SameNamespace(t *testing.T) {
	s := store.NewOptimizedStore()
	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					SecretRef: &v1alpha1.ResourceRef{
						Name: "my-tls-secret",
					},
				},
			},
		},
	}

	domains := []string{"example.com", "api.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "my-tls-secret"}
	assert.Len(t, result, 1)
	assert.Equal(t, domains, result[expectedNN])
}

func TestGetSecretNameToDomains_SecretRef_ExplicitNamespace(t *testing.T) {
	s := store.NewOptimizedStore()
	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					SecretRef: &v1alpha1.ResourceRef{
						Name:      "my-tls-secret",
						Namespace: stringPtr("cert-manager"),
					},
				},
			},
		},
	}

	domains := []string{"example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	expectedNN := helpers.NamespacedName{Namespace: "cert-manager", Name: "my-tls-secret"}
	assert.Len(t, result, 1)
	assert.Equal(t, domains, result[expectedNN])
}

func TestGetSecretNameToDomains_NilTlsConfig(t *testing.T) {
	s := store.NewOptimizedStore()
	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: nil,
			},
		},
	}

	_, err := builder.GetSecretNameToDomains(vs, []string{"example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TLS configuration is missing")
}

// =============================================================================
// GetSecretNameToDomains tests - AutoDiscovery mode
// =============================================================================

func TestGetSecretNameToDomains_AutoDiscovery_ExactMatch(t *testing.T) {
	s := store.NewOptimizedStore()

	// Create a valid certificate
	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}
	s.SetSecret(secret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "example-tls"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"example.com"}, result[expectedNN])
}

func TestGetSecretNameToDomains_AutoDiscovery_WildcardMatch(t *testing.T) {
	s := store.NewOptimizedStore()

	// Create a valid wildcard certificate
	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "*.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}
	s.SetSecret(secret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	// Request api.example.com which should match *.example.com
	domains := []string{"api.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "wildcard-tls"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"api.example.com"}, result[expectedNN])
}

func TestGetSecretNameToDomains_AutoDiscovery_NotFound(t *testing.T) {
	s := store.NewOptimizedStore()
	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"unknown.example.com"}

	_, err := builder.GetSecretNameToDomains(vs, domains)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't find secret for domain unknown.example.com")
}

func TestGetSecretNameToDomains_AutoDiscovery_MultipleDomainsSameSecret(t *testing.T) {
	s := store.NewOptimizedStore()

	// Create a wildcard certificate
	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "*.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}
	s.SetSecret(secret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	// Multiple subdomains should all map to the same wildcard secret
	domains := []string{"api.example.com", "www.example.com", "admin.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "wildcard-tls"}
	assert.Len(t, result, 1)
	assert.ElementsMatch(t, domains, result[expectedNN])
}

func TestGetSecretNameToDomains_AutoDiscovery_MultipleDomainsDifferentSecrets(t *testing.T) {
	s := store.NewOptimizedStore()

	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))

	// Create secret for example.com
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	// Create secret for other.com
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "other.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	s.SetSecret(secret1)
	s.SetSecret(secret2)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"example.com", "other.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	assert.Len(t, result, 2)

	nn1 := helpers.NamespacedName{Namespace: "default", Name: "example-tls"}
	nn2 := helpers.NamespacedName{Namespace: "default", Name: "other-tls"}

	assert.Equal(t, []string{"example.com"}, result[nn1])
	assert.Equal(t, []string{"other.com"}, result[nn2])
}

// =============================================================================
// Wildcard Fallback tests
// =============================================================================

func TestGetSecretNameToDomains_AutoDiscovery_WildcardFallback_ExactExpired(t *testing.T) {
	s := store.NewOptimizedStore()

	// Create expired exact certificate
	expiredCert := testutil.GenerateTestCertificate(time.Now().Add(-24 * time.Hour))
	exactSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exact-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "api.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": expiredCert,
			"tls.key": []byte("key"),
		},
	}

	// Create valid wildcard certificate
	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))
	wildcardSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "*.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	s.SetSecret(exactSecret)
	s.SetSecret(wildcardSecret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"api.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	// Should fallback to wildcard because exact is expired
	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "wildcard-tls"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"api.example.com"}, result[expectedNN])
}

func TestGetSecretNameToDomains_AutoDiscovery_WildcardFallback_ExactValid(t *testing.T) {
	s := store.NewOptimizedStore()

	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))

	// Create valid exact certificate
	exactSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exact-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "api.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	// Create valid wildcard certificate
	wildcardSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "*.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	s.SetSecret(exactSecret)
	s.SetSecret(wildcardSecret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"api.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	// Should use exact certificate because it's valid
	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "exact-tls"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"api.example.com"}, result[expectedNN])
}

func TestGetSecretNameToDomains_AutoDiscovery_WildcardFallback_ExactUnknown(t *testing.T) {
	s := store.NewOptimizedStore()

	// Create unparseable exact certificate
	exactSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exact-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "api.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("not a valid certificate"),
			"tls.key": []byte("key"),
		},
	}

	// Create valid wildcard certificate
	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))
	wildcardSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "*.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	s.SetSecret(exactSecret)
	s.SetSecret(wildcardSecret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"api.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	// Should fallback to wildcard because exact is unparseable
	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "wildcard-tls"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"api.example.com"}, result[expectedNN])
}

func TestGetSecretNameToDomains_AutoDiscovery_WildcardFallback_BothExpired(t *testing.T) {
	s := store.NewOptimizedStore()

	expiredCert := testutil.GenerateTestCertificate(time.Now().Add(-24 * time.Hour))

	// Create expired exact certificate
	exactSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exact-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "api.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": expiredCert,
			"tls.key": []byte("key"),
		},
	}

	// Create expired wildcard certificate
	wildcardSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard-tls",
			Namespace: "default",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "*.example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": expiredCert,
			"tls.key": []byte("key"),
		},
	}

	s.SetSecret(exactSecret)
	s.SetSecret(wildcardSecret)

	builder := NewBuilder(s)

	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"api.example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	// Should use exact certificate because both are expired (exact is more specific)
	expectedNN := helpers.NamespacedName{Namespace: "default", Name: "exact-tls"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"api.example.com"}, result[expectedNN])
}

// =============================================================================
// Namespace preference tests
// =============================================================================

func TestGetSecretNameToDomains_AutoDiscovery_NamespacePreference(t *testing.T) {
	s := store.NewOptimizedStore()

	validCert := testutil.GenerateTestCertificate(time.Now().Add(24 * time.Hour))

	// Create secret in ns1
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-cert",
			Namespace: "ns1",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	// Create secret in ns2
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-cert",
			Namespace: "ns2",
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": "example.com",
			},
		},
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": []byte("key"),
		},
	}

	s.SetSecret(secret1)
	s.SetSecret(secret2)

	builder := NewBuilder(s)

	// VirtualService in ns2 should prefer ns2 secret
	vs := &v1alpha1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vs",
			Namespace: "ns2",
		},
		Spec: v1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
				TlsConfig: &v1alpha1.TlsConfig{
					AutoDiscovery: boolPtr(true),
				},
			},
		},
	}

	domains := []string{"example.com"}

	result, err := builder.GetSecretNameToDomains(vs, domains)
	require.NoError(t, err)

	expectedNN := helpers.NamespacedName{Namespace: "ns2", Name: "tls-cert"}
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"example.com"}, result[expectedNN])
}

// =============================================================================
// Builder method tests
// =============================================================================

func TestBuilder_GetTLSType(t *testing.T) {
	s := store.NewOptimizedStore()
	builder := NewBuilder(s)

	// Test that Builder.GetTLSType delegates to the package-level function
	tlsConfig := &v1alpha1.TlsConfig{
		SecretRef: &v1alpha1.ResourceRef{
			Name: "test",
		},
	}

	tlsType, err := builder.GetTLSType(tlsConfig)
	assert.NoError(t, err)
	assert.Equal(t, "secretRef", tlsType)
}

func TestNewBuilder(t *testing.T) {
	s := store.NewOptimizedStore()
	builder := NewBuilder(s)

	assert.NotNil(t, builder)
	assert.Equal(t, s, builder.store)
}
