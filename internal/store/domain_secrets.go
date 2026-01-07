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
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

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
	validityExpired validityPriority = iota // Known expired certificate - lowest priority
	validityUnknown                         // Could not parse certificate - medium priority (better than nothing)
	validityValid                           // Known valid (non-expired) certificate - highest priority
)

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
	entries, exists := idx[domain]
	if !exists || len(entries) == 0 {
		return nil
	}

	// Fast path: single secret
	if len(entries) == 1 {
		for nn := range entries {
			// Defensive nil check for consistency between index and secrets map
			if secret := secrets[nn]; secret != nil {
				return secret
			}
			return nil
		}
	}

	now := time.Now()

	// Collect entries into a slice for sorting
	type candidateEntry struct {
		nn       helpers.NamespacedName
		notAfter time.Time
		validity validityPriority
		isSameNs bool
	}

	candidates := make([]candidateEntry, 0, len(entries))
	for nn, entry := range entries {
		// Skip entries that don't exist in the secrets map (defensive check)
		if secrets[nn] == nil {
			continue
		}

		var validity validityPriority
		switch {
		case entry.NotAfter.IsZero():
			// Certificate parsing failed - treat as unknown validity
			validity = validityUnknown
		case entry.NotAfter.After(now):
			// Certificate is valid (not expired)
			validity = validityValid
		default:
			// Certificate is expired
			validity = validityExpired
		}

		candidates = append(candidates, candidateEntry{
			nn:       nn,
			notAfter: entry.NotAfter,
			validity: validity,
			isSameNs: nn.Namespace == preferredNamespace,
		})
	}

	// No valid candidates found
	if len(candidates) == 0 {
		return nil
	}

	// Sort candidates by priority:
	// 1. Higher validity priority first (valid > unknown > expired)
	// 2. Same namespace first
	// 3. Alphabetically by namespace/name
	sort.Slice(candidates, func(i, j int) bool {
		// Higher validity priority first
		if candidates[i].validity != candidates[j].validity {
			return candidates[i].validity > candidates[j].validity
		}
		// Same namespace first
		if candidates[i].isSameNs != candidates[j].isSameNs {
			return candidates[i].isSameNs
		}
		// Alphabetically by namespace
		if candidates[i].nn.Namespace != candidates[j].nn.Namespace {
			return candidates[i].nn.Namespace < candidates[j].nn.Namespace
		}
		// Alphabetically by name
		return candidates[i].nn.Name < candidates[j].nn.Name
	})

	// Return the best candidate
	return secrets[candidates[0].nn]
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
func ParseCertificateNotAfter(secret *corev1.Secret) time.Time {
	if secret == nil {
		return time.Time{}
	}

	certData, ok := secret.Data[corev1.TLSCertKey]
	if !ok || len(certData) == 0 {
		return time.Time{}
	}

	var minNotAfter time.Time
	rest := certData

	// Parse all PEM blocks in the certificate chain
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining

		// Skip non-certificate blocks (e.g., private keys that might be included)
		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		// Track the minimum (earliest) NotAfter time
		if minNotAfter.IsZero() || cert.NotAfter.Before(minNotAfter) {
			minNotAfter = cert.NotAfter
		}
	}

	return minNotAfter
}
