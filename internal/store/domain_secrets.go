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

// DomainSecretsIndex maps domains to sets of secrets
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

// GetBestSecret returns the best secret for a domain with preference for the given namespace.
// Selection logic:
// 1. If only one secret exists - return it
// 2. Filter to valid (non-expired) secrets
// 3. Among valid: prefer same namespace, then alphabetically by namespace/name
// 4. If all expired: prefer same namespace, then alphabetically
func (idx DomainSecretsIndex) GetBestSecret(domain string, preferredNamespace string, secrets map[helpers.NamespacedName]*corev1.Secret) *corev1.Secret {
	entries, exists := idx[domain]
	if !exists || len(entries) == 0 {
		return nil
	}

	// Fast path: single secret
	if len(entries) == 1 {
		for nn := range entries {
			return secrets[nn]
		}
	}

	now := time.Now()

	// Collect entries into a slice for sorting
	type candidateEntry struct {
		nn       helpers.NamespacedName
		notAfter time.Time
		isValid  bool
		isSameNs bool
	}

	candidates := make([]candidateEntry, 0, len(entries))
	for nn, entry := range entries {
		candidates = append(candidates, candidateEntry{
			nn:       nn,
			notAfter: entry.NotAfter,
			isValid:  entry.NotAfter.IsZero() || entry.NotAfter.After(now),
			isSameNs: nn.Namespace == preferredNamespace,
		})
	}

	// Sort candidates by priority:
	// 1. Valid secrets first
	// 2. Same namespace first
	// 3. Alphabetically by namespace/name
	sort.Slice(candidates, func(i, j int) bool {
		// Valid secrets first
		if candidates[i].isValid != candidates[j].isValid {
			return candidates[i].isValid
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
func (idx DomainSecretsIndex) GetAnySecret(domain string, secrets map[helpers.NamespacedName]*corev1.Secret) *corev1.Secret {
	return idx.GetBestSecret(domain, "", secrets)
}

// ParseCertificateNotAfter extracts the NotAfter time from a TLS secret
// Returns zero time if parsing fails
func ParseCertificateNotAfter(secret *corev1.Secret) time.Time {
	if secret == nil {
		return time.Time{}
	}

	certData, ok := secret.Data[corev1.TLSCertKey]
	if !ok || len(certData) == 0 {
		return time.Time{}
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		return time.Time{}
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}
	}

	return cert.NotAfter
}
