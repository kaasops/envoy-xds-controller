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
	"crypto/x509"
	"encoding/pem"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

// ValidateDomainPattern checks if a domain pattern is valid.
// Returns an error message if invalid, empty string if valid.
//
// Valid patterns:
//   - example.com (exact domain)
//   - *.example.com (wildcard - asterisk followed by dot)
//
// Invalid patterns:
//   - *example.com (asterisk without dot)
//   - **.example.com (double asterisk)
//   - example.*.com (asterisk not at start)
//   - * (standalone asterisk)
func ValidateDomainPattern(domain string) string {
	if domain == "" {
		return "empty domain"
	}

	// Check for asterisk in the domain
	asteriskIdx := strings.Index(domain, "*")
	if asteriskIdx == -1 {
		// No wildcard, valid exact domain
		return ""
	}

	// Wildcard validation
	if asteriskIdx != 0 {
		return "wildcard (*) must be at the start of domain"
	}

	if len(domain) < 3 {
		// Need at least "*.x"
		return "wildcard domain too short"
	}

	if domain[1] != '.' {
		return "wildcard must be followed by dot (e.g., *.example.com, not *example.com)"
	}

	// Check for multiple asterisks
	if strings.Count(domain, "*") > 1 {
		return "multiple wildcards not allowed"
	}

	// Check that there's something after "*."
	rest := domain[2:]
	if rest == "" || rest == "." {
		return "wildcard domain must have at least one label after *."
	}

	return ""
}

// SecretDomainEntry holds information about a secret for domain lookup
type SecretDomainEntry struct {
	NamespacedName helpers.NamespacedName
	NotAfter       time.Time // Certificate expiration time, zero if parsing failed
}

// DomainSecretsIndex maps domains to sets of secrets.
// This structure is NOT thread-safe. All operations must be performed
// under OptimizedStore's mutex to ensure safe concurrent access.
type DomainSecretsIndex map[string]map[helpers.NamespacedName]SecretDomainEntry

// NewDomainSecretsIndex creates a new empty index
func NewDomainSecretsIndex(capacity int) DomainSecretsIndex {
	return make(DomainSecretsIndex, capacity)
}

// Add adds a secret entry for a domain
func (idx DomainSecretsIndex) Add(domain string, entry SecretDomainEntry) {
	if idx[domain] == nil {
		idx[domain] = make(map[helpers.NamespacedName]SecretDomainEntry)
	}
	idx[domain][entry.NamespacedName] = entry
}

// Remove removes a secret entry for a domain
func (idx DomainSecretsIndex) Remove(domain string, nn helpers.NamespacedName) {
	if idx[domain] == nil {
		return
	}
	delete(idx[domain], nn)
	// Clean up empty maps
	if len(idx[domain]) == 0 {
		delete(idx, domain)
	}
}

// validityPriority represents certificate validity states with priority ordering.
// Higher values indicate higher priority in secret selection.
type validityPriority int

const (
	validityNotFound validityPriority = iota // Domain not found in index - used only for "not found" returns
	validityExpired                          // Known expired certificate - lowest priority for actual certs
	validityUnknown                          // Could not parse certificate - medium priority (better than nothing)
	validityValid                            // Known valid (non-expired) certificate - highest priority
)

// SecretLookupResult contains the result of a secret lookup with additional metadata.
// This provides complete diagnostic information about the secret selection process.
type SecretLookupResult struct {
	Secret             *corev1.Secret
	UsedWildcard       bool   // true if wildcard secret was used instead of exact
	FallbackReason     string // reason for fallback: "expired", "unknown", or empty if no fallback
	ExactSecretName    string // name of the exact secret if it existed (format: "namespace/name")
	ExactValidity      string // validity of exact secret: "valid", "expired", "unknown", "not_found"
	WildcardSecretName string // name of the wildcard secret if it was considered (format: "namespace/name")
	WildcardValidity   string // validity of wildcard secret: "valid", "expired", "unknown", "not_found"
}

// GetBestSecretWithValidity returns the best secret for a domain along with its validity status.
// This is an optimized version that does a single traversal instead of calling
// GetBestSecret and GetSecretValidity separately.
// Returns (nil, validityNotFound) if domain not found in index.
func (idx DomainSecretsIndex) GetBestSecretWithValidity(
	domain string,
	preferredNamespace string,
	secrets map[helpers.NamespacedName]*corev1.Secret,
) (*corev1.Secret, validityPriority) {
	entries, exists := idx[domain]
	if !exists || len(entries) == 0 {
		return nil, validityNotFound
	}

	// Capture time once for consistent validity checks throughout this call
	now := time.Now()

	// Fast path: single secret
	// Note: using range to get the single element from a map is a Go idiom
	if len(entries) == 1 {
		for nn, entry := range entries {
			secret := secrets[nn]
			if secret == nil {
				return nil, validityNotFound // indexed but missing from secrets map
			}
			validity := getValidityFromEntry(entry, now)
			return secret, validity
		}
	}

	// Collect and sort candidates
	type candidateEntry struct {
		nn       helpers.NamespacedName
		validity validityPriority
		isSameNs bool
	}

	candidates := make([]candidateEntry, 0, len(entries))
	for nn, entry := range entries {
		if secrets[nn] == nil {
			continue
		}

		var validity validityPriority
		switch {
		case entry.NotAfter.IsZero():
			validity = validityUnknown
		case entry.NotAfter.After(now):
			validity = validityValid
		default:
			validity = validityExpired
		}

		candidates = append(candidates, candidateEntry{
			nn:       nn,
			validity: validity,
			isSameNs: nn.Namespace == preferredNamespace,
		})
	}

	if len(candidates) == 0 {
		return nil, validityNotFound // all indexed secrets missing from secrets map
	}

	// Sort: validity desc, same namespace first, then alphabetically
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].validity != candidates[j].validity {
			return candidates[i].validity > candidates[j].validity
		}
		if candidates[i].isSameNs != candidates[j].isSameNs {
			return candidates[i].isSameNs
		}
		if candidates[i].nn.Namespace != candidates[j].nn.Namespace {
			return candidates[i].nn.Namespace < candidates[j].nn.Namespace
		}
		return candidates[i].nn.Name < candidates[j].nn.Name
	})

	best := candidates[0]
	return secrets[best.nn], best.validity
}

// getValidityFromEntry determines validity from a single entry.
// The now parameter ensures consistent time comparison across all validity checks.
func getValidityFromEntry(entry SecretDomainEntry, now time.Time) validityPriority {
	switch {
	case entry.NotAfter.IsZero():
		return validityUnknown
	case entry.NotAfter.After(now):
		return validityValid
	default:
		return validityExpired
	}
}

// GetBestSecret returns the best secret for a domain with preference for the given namespace.
// Selection logic:
// 1. If only one secret exists - return it (if it exists in secrets map)
// 2. Priority by validity: valid > unknown > expired
// 3. Among same validity: prefer same namespace
// 4. Final tie-breaker: alphabetically by namespace/name
//
// Note: Secrets with unparseable certificates (validityUnknown) are ranked below valid
// certificates but above expired ones. This ensures we prefer known-good certificates
// while still providing fallback behavior when certificate parsing fails.
func (idx DomainSecretsIndex) GetBestSecret(
	domain string,
	preferredNamespace string,
	secrets map[helpers.NamespacedName]*corev1.Secret,
) *corev1.Secret {
	secret, _ := idx.GetBestSecretWithValidity(domain, preferredNamespace, secrets)
	return secret
}

// GetAnySecret returns any valid secret for a domain (for backward compatibility)
// Uses empty string as preferred namespace, so it just picks alphabetically first valid secret
func (idx DomainSecretsIndex) GetAnySecret(
	domain string,
	secrets map[helpers.NamespacedName]*corev1.Secret,
) *corev1.Secret {
	return idx.GetBestSecret(domain, "", secrets)
}

// ParseCertificateNotAfter extracts the minimum NotAfter time from a TLS secret.
// For certificate chains (containing multiple certificates), returns the earliest
// expiration time to ensure we consider the most restrictive validity period.
// This handles cases where the end-entity certificate expires before intermediate/root CAs.
// Returns zero time if no valid certificates could be parsed.
// Logs warnings for parsing errors to aid debugging in production.
func ParseCertificateNotAfter(secret *corev1.Secret) time.Time {
	logger := log.Log.WithName("certificate-parser")

	if secret == nil {
		return time.Time{}
	}

	secretKey := secret.Namespace + "/" + secret.Name

	certData, ok := secret.Data[corev1.TLSCertKey]
	if !ok || len(certData) == 0 {
		logger.V(1).Info("Secret missing tls.crt data",
			"secret", secretKey)
		return time.Time{}
	}

	var minNotAfter time.Time
	rest := certData
	blockIndex := 0
	parseErrors := 0

	// Parse all PEM blocks in the certificate chain
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		blockIndex++

		// Skip non-certificate blocks (e.g., private keys that might be included)
		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			parseErrors++
			logger.V(1).Info("Failed to parse certificate in chain",
				"secret", secretKey,
				"blockIndex", blockIndex,
				"error", err.Error())
			continue
		}

		// Track the minimum (earliest) NotAfter time
		if minNotAfter.IsZero() || cert.NotAfter.Before(minNotAfter) {
			minNotAfter = cert.NotAfter
		}
	}

	// Log warning if all certificates failed to parse
	if minNotAfter.IsZero() && blockIndex > 0 {
		logger.Info("Failed to parse any certificates from secret",
			"secret", secretKey,
			"totalBlocks", blockIndex,
			"parseErrors", parseErrors)
	}

	return minNotAfter
}
