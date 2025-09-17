package tls

import (
	"fmt"
	"strings"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	v1 "k8s.io/api/core/v1"
)

// Builder handles TLS configuration
type Builder struct {
	store *store.Store
}

// NewBuilder creates a new TLS builder
func NewBuilder(store *store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// GetTLSType determines the TLS configuration type from TlsConfig
func (b *Builder) GetTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error) {
	if vsTLSConfig == nil {
		return "", fmt.Errorf("TLS configuration is missing: please provide TLS parameters")
	}
	if vsTLSConfig.SecretRef != nil {
		if vsTLSConfig.AutoDiscovery != nil && *vsTLSConfig.AutoDiscovery {
			return "", fmt.Errorf("TLS configuration conflict: cannot use both secretRef and autoDiscovery simultaneously")
		}
		return utils.SecretRefType, nil
	}
	if vsTLSConfig.AutoDiscovery != nil {
		if !*vsTLSConfig.AutoDiscovery {
			return "", fmt.Errorf("invalid TLS configuration: cannot use autoDiscovery=false without specifying secretRef")
		}
		return utils.AutoDiscoveryType, nil
	}
	return "", fmt.Errorf("empty TLS configuration: either secretRef or autoDiscovery must be specified")
}

// GetSecretNameToDomains maps domains to secrets based on VirtualService TLS configuration
func (b *Builder) GetSecretNameToDomains(vs *v1alpha1.VirtualService, domains []string) (map[helpers.NamespacedName][]string, error) {
	if vs.Spec.TlsConfig == nil {
		return nil, fmt.Errorf("TLS configuration is missing in VirtualService")
	}

	tlsType, err := b.GetTLSType(vs.Spec.TlsConfig)
	if err != nil {
		return nil, err
	}

	switch tlsType {
	case utils.SecretRefType:
		return b.getSecretNameToDomainsViaSecretRef(vs.Spec.TlsConfig.SecretRef, vs.Namespace, domains), nil
	case utils.AutoDiscoveryType:
		return b.getSecretNameToDomainsViaAutoDiscovery(domains, b.store.MapDomainSecrets())
	default:
		return nil, fmt.Errorf("unknown TLS type: %s", tlsType)
	}
}

// getSecretNameToDomainsViaSecretRef maps domains to a single secret for secretRef type
func (b *Builder) getSecretNameToDomainsViaSecretRef(secretRef *v1alpha1.ResourceRef, vsNamespace string, domains []string) map[helpers.NamespacedName][]string {
	m := make(map[helpers.NamespacedName][]string)

	var secretNamespace string

	if secretRef.Namespace != nil {
		secretNamespace = *secretRef.Namespace
	} else {
		secretNamespace = vsNamespace
	}

	m[helpers.NamespacedName{Namespace: secretNamespace, Name: secretRef.Name}] = domains
	return m
}

// getSecretNameToDomainsViaAutoDiscovery maps domains to secrets based on auto-discovery
func (b *Builder) getSecretNameToDomainsViaAutoDiscovery(domains []string, domainToSecretMap map[string]v1.Secret) (map[helpers.NamespacedName][]string, error) {
	m := make(map[helpers.NamespacedName][]string)

	for _, domain := range domains {
		var secret v1.Secret
		secret, ok := domainToSecretMap[domain]
		if !ok {
			secret, ok = domainToSecretMap[b.getWildcardDomain(domain)]
			if !ok {
				return nil, fmt.Errorf("can't find secret for domain %s", domain)
			}
		}

		domainsFromMap, ok := m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}]
		if ok {
			m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = append(domainsFromMap, domain)
		} else {
			m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = []string{domain}
		}
	}

	return m, nil
}

// getWildcardDomain converts a domain to its wildcard form
func (b *Builder) getWildcardDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	parts[0] = "*"
	return strings.Join(parts, ".")
}