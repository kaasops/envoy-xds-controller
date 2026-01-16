package handlers

import (
	"testing"
)

func TestParseSecretName(t *testing.T) {
	tests := []struct {
		name              string
		secretName        string
		expectedNamespace string
		expectedName      string
	}{
		{
			name:              "with namespace",
			secretName:        "default/my-secret",
			expectedNamespace: "default",
			expectedName:      "my-secret",
		},
		{
			name:              "with different namespace",
			secretName:        "production/tls-cert",
			expectedNamespace: "production",
			expectedName:      "tls-cert",
		},
		{
			name:              "without namespace",
			secretName:        "my-secret",
			expectedNamespace: "",
			expectedName:      "my-secret",
		},
		{
			name:              "empty string",
			secretName:        "",
			expectedNamespace: "",
			expectedName:      "",
		},
		{
			name:              "multiple slashes (returns as-is since K8s names don't contain slashes)",
			secretName:        "ns/name/extra",
			expectedNamespace: "",
			expectedName:      "ns/name/extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, name := parseSecretName(tt.secretName)
			if namespace != tt.expectedNamespace {
				t.Errorf("parseSecretName(%q) namespace = %q, want %q", tt.secretName, namespace, tt.expectedNamespace)
			}
			if name != tt.expectedName {
				t.Errorf("parseSecretName(%q) name = %q, want %q", tt.secretName, name, tt.expectedName)
			}
		})
	}
}

func TestCalculateCertStatus(t *testing.T) {
	tests := []struct {
		name            string
		daysUntilExpiry int
		expectedStatus  string
	}{
		{
			name:            "expired (negative days)",
			daysUntilExpiry: -5,
			expectedStatus:  CertStatusExpired,
		},
		{
			name:            "expired (zero days)",
			daysUntilExpiry: 0,
			expectedStatus:  CertStatusExpired,
		},
		{
			name:            "critical (1 day)",
			daysUntilExpiry: 1,
			expectedStatus:  CertStatusCritical,
		},
		{
			name:            "critical (7 days)",
			daysUntilExpiry: 7,
			expectedStatus:  CertStatusCritical,
		},
		{
			name:            "warning (8 days)",
			daysUntilExpiry: 8,
			expectedStatus:  CertStatusWarning,
		},
		{
			name:            "warning (30 days)",
			daysUntilExpiry: 30,
			expectedStatus:  CertStatusWarning,
		},
		{
			name:            "ok (31 days)",
			daysUntilExpiry: 31,
			expectedStatus:  CertStatusOK,
		},
		{
			name:            "ok (365 days)",
			daysUntilExpiry: 365,
			expectedStatus:  CertStatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := calculateCertStatus(tt.daysUntilExpiry)
			if status != tt.expectedStatus {
				t.Errorf("calculateCertStatus(%d) = %q, want %q", tt.daysUntilExpiry, status, tt.expectedStatus)
			}
		})
	}
}

func TestCalculateSummary(t *testing.T) {
	tests := []struct {
		name         string
		endpoints    []EndpointInfo
		certificates []CertificateInfo
		expected     OverviewSummary
	}{
		{
			name:         "empty inputs",
			endpoints:    []EndpointInfo{},
			certificates: []CertificateInfo{},
			expected: OverviewSummary{
				TotalDomains:         0,
				TotalEndpoints:       0,
				TotalCertificates:    0,
				CertificatesWarning:  0,
				CertificatesCritical: 0,
				CertificatesExpired:  0,
			},
		},
		{
			name: "only wildcard domains (should not count)",
			endpoints: []EndpointInfo{
				{Domain: "*", Port: 80},
				{Domain: "*", Port: 443},
			},
			certificates: []CertificateInfo{},
			expected: OverviewSummary{
				TotalDomains:      0,
				TotalEndpoints:    2,
				TotalCertificates: 0,
			},
		},
		{
			name: "unique domains with duplicates",
			endpoints: []EndpointInfo{
				{Domain: "example.com", Port: 80},
				{Domain: "example.com", Port: 443},
				{Domain: "api.example.com", Port: 443},
				{Domain: "*", Port: 8080},
			},
			certificates: []CertificateInfo{},
			expected: OverviewSummary{
				TotalDomains:   2, // example.com and api.example.com
				TotalEndpoints: 4,
			},
		},
		{
			name:      "certificates with different statuses",
			endpoints: []EndpointInfo{},
			certificates: []CertificateInfo{
				{Name: "cert1", Status: CertStatusOK},
				{Name: "cert2", Status: CertStatusOK},
				{Name: "cert3", Status: CertStatusWarning},
				{Name: "cert4", Status: CertStatusCritical},
				{Name: "cert5", Status: CertStatusExpired},
				{Name: "cert6", Status: CertStatusExpired},
			},
			expected: OverviewSummary{
				TotalDomains:         0,
				TotalEndpoints:       0,
				TotalCertificates:    6,
				CertificatesWarning:  1,
				CertificatesCritical: 1,
				CertificatesExpired:  2,
			},
		},
		{
			name: "mixed scenario",
			endpoints: []EndpointInfo{
				{Domain: "web.example.com", Port: 443},
				{Domain: "api.example.com", Port: 443},
				{Domain: "web.example.com", Port: 80},
			},
			certificates: []CertificateInfo{
				{Name: "wildcard-cert", Status: CertStatusOK},
				{Name: "expiring-cert", Status: CertStatusWarning},
			},
			expected: OverviewSummary{
				TotalDomains:         2,
				TotalEndpoints:       3,
				TotalCertificates:    2,
				CertificatesWarning:  1,
				CertificatesCritical: 0,
				CertificatesExpired:  0,
			},
		},
	}

	h := &handler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.calculateSummary(tt.endpoints, tt.certificates)

			if result.TotalDomains != tt.expected.TotalDomains {
				t.Errorf("TotalDomains = %d, want %d", result.TotalDomains, tt.expected.TotalDomains)
			}
			if result.TotalEndpoints != tt.expected.TotalEndpoints {
				t.Errorf("TotalEndpoints = %d, want %d", result.TotalEndpoints, tt.expected.TotalEndpoints)
			}
			if result.TotalCertificates != tt.expected.TotalCertificates {
				t.Errorf("TotalCertificates = %d, want %d", result.TotalCertificates, tt.expected.TotalCertificates)
			}
			if result.CertificatesWarning != tt.expected.CertificatesWarning {
				t.Errorf("CertificatesWarning = %d, want %d", result.CertificatesWarning, tt.expected.CertificatesWarning)
			}
			if result.CertificatesCritical != tt.expected.CertificatesCritical {
				t.Errorf("CertificatesCritical = %d, want %d", result.CertificatesCritical, tt.expected.CertificatesCritical)
			}
			if result.CertificatesExpired != tt.expected.CertificatesExpired {
				t.Errorf("CertificatesExpired = %d, want %d", result.CertificatesExpired, tt.expected.CertificatesExpired)
			}
		})
	}
}

func TestBuildCertificatesList(t *testing.T) {
	h := &handler{}

	certMap := map[string]*CertificateInfo{
		"ns1/cert1": {
			Name:      "cert1",
			Namespace: "ns1",
			Status:    CertStatusOK,
		},
		"ns2/cert2": {
			Name:      "cert2",
			Namespace: "ns2",
			Status:    CertStatusWarning,
		},
	}

	result := h.buildCertificatesList(certMap)

	if len(result) != 2 {
		t.Errorf("Expected 2 certificates, got %d", len(result))
	}

	// Verify each certificate is present (order is not guaranteed)
	foundCert1, foundCert2 := false, false
	for _, cert := range result {
		if cert.Name == "cert1" && cert.Namespace == "ns1" {
			foundCert1 = true
		}
		if cert.Name == "cert2" && cert.Namespace == "ns2" {
			foundCert2 = true
		}
	}

	if !foundCert1 {
		t.Error("cert1 not found in result")
	}
	if !foundCert2 {
		t.Error("cert2 not found in result")
	}
}

func TestBuildCertificatesList_EmptyMap(t *testing.T) {
	h := &handler{}

	result := h.buildCertificatesList(map[string]*CertificateInfo{})

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(result))
	}
}

func TestCertificateStatusConstants(t *testing.T) {
	// Verify constants are as expected
	if CertStatusOK != "ok" {
		t.Errorf("CertStatusOK = %q, want %q", CertStatusOK, "ok")
	}
	if CertStatusWarning != "warning" {
		t.Errorf("CertStatusWarning = %q, want %q", CertStatusWarning, "warning")
	}
	if CertStatusCritical != "critical" {
		t.Errorf("CertStatusCritical = %q, want %q", CertStatusCritical, "critical")
	}
	if CertStatusExpired != "expired" {
		t.Errorf("CertStatusExpired = %q, want %q", CertStatusExpired, "expired")
	}
}

func TestCertificateThresholds(t *testing.T) {
	// Verify thresholds
	if CertWarningThresholdDays != 30 {
		t.Errorf("CertWarningThresholdDays = %d, want %d", CertWarningThresholdDays, 30)
	}
	if CertCriticalThresholdDays != 7 {
		t.Errorf("CertCriticalThresholdDays = %d, want %d", CertCriticalThresholdDays, 7)
	}
}
