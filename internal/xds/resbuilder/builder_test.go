package resbuilder

import (
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestApplyVirtualServiceTemplate(t *testing.T) {
	testCases := []struct {
		name          string
		vs            *v1alpha1.VirtualService
		templates     []*v1alpha1.VirtualServiceTemplate
		expectError   bool
		errorContains string
	}{
		{
			name: "No template",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{},
			},
			templates:   nil,
			expectError: false,
		},
		{
			name: "Template not found",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					Template: &v1alpha1.ResourceRef{
						Name: "non-existent",
					},
				},
			},
			templates:     nil,
			expectError:   true,
			errorContains: "not found",
		},
		{
			name: "Valid template",
			vs: &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
				},
				Spec: v1alpha1.VirtualServiceSpec{
					Template: &v1alpha1.ResourceRef{
						Name: "test-template",
					},
				},
			},
			templates: []*v1alpha1.VirtualServiceTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "default",
					},
					Spec: v1alpha1.VirtualServiceTemplateSpec{
						VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
							Listener: &v1alpha1.ResourceRef{
								Name: "test-listener",
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock store
			mockStore := store.New()
			for _, template := range tc.templates {
				mockStore.SetVirtualServiceTemplate(template)
			}

			// Call the function
			result, err := applyVirtualServiceTemplate(tc.vs, mockStore)

			// Check the result
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.vs.Spec.Template == nil {
					assert.Equal(t, tc.vs, result)
				} else {
					assert.NotEqual(t, tc.vs, result)
				}
			}
		})
	}
}

func TestCheckFilterChainsConflicts(t *testing.T) {
	testCases := []struct {
		name          string
		vs            *v1alpha1.VirtualService
		expectError   bool
		errorContains string
	}{
		{
			name: "No conflicts",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{},
			},
			expectError: false,
		},
		{
			name: "Conflict with VirtualHost",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						VirtualHost: &runtime.RawExtension{},
					},
				},
			},
			expectError:   true,
			errorContains: "virtual host is set",
		},
		{
			name: "Conflict with AdditionalRoutes",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						AdditionalRoutes: []*v1alpha1.ResourceRef{
							{Name: "test-route"},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "additional routes are set",
		},
		{
			name: "Conflict with HTTPFilters",
			vs: &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
						HTTPFilters: []*runtime.RawExtension{
							{},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "http filters are set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFilterChainsConflicts(tc.vs)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractClustersFromFilterChains(t *testing.T) {
	// This test would require mocking the listenerv3.FilterChain and store.Store
	// For simplicity, we'll just test the error cases
	t.Run("Empty filter chains", func(t *testing.T) {
		mockStore := store.New()
		clusters, err := extractClustersFromFilterChains(nil, mockStore)
		assert.NoError(t, err)
		assert.Empty(t, clusters)
	})
}

func TestGetWildcardDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{
			name:     "Simple domain",
			domain:   "example.com",
			expected: "*.com",
		},
		{
			name:     "Subdomain",
			domain:   "sub.example.com",
			expected: "*.example.com",
		},
		{
			name:     "Already wildcard",
			domain:   "*.example.com",
			expected: "*.example.com",
		},
		{
			name:     "Empty domain",
			domain:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWildcardDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckAllDomainsUnique(t *testing.T) {
	tests := []struct {
		name        string
		domains     []string
		expectError bool
	}{
		{
			name:        "All unique domains",
			domains:     []string{"example.com", "test.com", "another.com"},
			expectError: false,
		},
		{
			name:        "Duplicate domains",
			domains:     []string{"example.com", "test.com", "example.com"},
			expectError: true,
		},
		{
			name:        "Empty domains",
			domains:     []string{},
			expectError: false,
		},
		{
			name:        "Single domain",
			domains:     []string{"example.com"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAllDomainsUnique(tt.domains)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsTLSListener(t *testing.T) {
	// Test with nil listener
	result := isTLSListener(nil)
	assert.False(t, result, "nil listener should not be considered a TLS listener")
}

func TestGetTLSType(t *testing.T) {
	tests := []struct {
		name        string
		tlsConfig   *v1alpha1.TlsConfig
		expected    string
		expectError bool
	}{
		{
			name:        "Nil config",
			tlsConfig:   nil,
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty config",
			tlsConfig:   &v1alpha1.TlsConfig{},
			expected:    "",
			expectError: true,
		},
		{
			name: "SecretRef type",
			tlsConfig: &v1alpha1.TlsConfig{
				SecretRef: &v1alpha1.ResourceRef{
					Name: "test-secret",
				},
			},
			expected:    SecretRefType,
			expectError: false,
		},
		{
			name: "AutoDiscovery type",
			tlsConfig: &v1alpha1.TlsConfig{
				AutoDiscovery: func() *bool { b := true; return &b }(),
			},
			expected:    AutoDiscoveryType,
			expectError: false,
		},
		{
			name: "Both types specified",
			tlsConfig: &v1alpha1.TlsConfig{
				SecretRef: &v1alpha1.ResourceRef{
					Name: "test-secret",
				},
				AutoDiscovery: func() *bool { b := true; return &b }(),
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getTLSType(tt.tlsConfig)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
