package adapters

import (
	"fmt"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/secrets"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	v1 "k8s.io/api/core/v1"
)

// TLSAdapter adapts the secrets package functions to implement the TLSBuilder interface
type TLSAdapter struct {
	store store.Store
}

// NewTLSAdapter creates a new adapter for TLS functionality
func NewTLSAdapter(store store.Store) interfaces.TLSBuilder {
	return &TLSAdapter{
		store: store,
	}
}

// GetTLSType determines the TLS configuration type from the TlsConfig
func (a *TLSAdapter) GetTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error) {
	// Delegate to the secrets.GetTLSType function
	return secrets.GetTLSType(vsTLSConfig)
}

// GetSecretNameToDomains maps domains to secrets based on the VirtualService's TLS configuration
func (a *TLSAdapter) GetSecretNameToDomains(vs *v1alpha1.VirtualService, domains []string) (map[helpers.NamespacedName][]string, error) {
	if vs.Spec.TlsConfig == nil {
		return nil, fmt.Errorf("TLS configuration is missing in VirtualService")
	}

	tlsType, err := a.GetTLSType(vs.Spec.TlsConfig)
	if err != nil {
		return nil, err
	}

	switch tlsType {
	case utils.SecretRefType:
		return a.getSecretNameToDomainsViaSecretRef(vs.Spec.TlsConfig.SecretRef, vs.Namespace, domains), nil
	case utils.AutoDiscoveryType:
		return a.getSecretNameToDomainsViaAutoDiscovery(domains, a.store.MapDomainSecrets())
	default:
		return nil, fmt.Errorf("unknown TLS type: %s", tlsType)
	}
}

// getSecretNameToDomainsViaSecretRef maps domains to a single secret for secretRef type
func (a *TLSAdapter) getSecretNameToDomainsViaSecretRef(
	secretRef *v1alpha1.ResourceRef,
	vsNamespace string,
	domains []string,
) map[helpers.NamespacedName][]string {
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
func (a *TLSAdapter) getSecretNameToDomainsViaAutoDiscovery(
	domains []string,
	domainToSecretMap map[string]v1.Secret,
) (map[helpers.NamespacedName][]string, error) {
	m := make(map[helpers.NamespacedName][]string)

	for _, domain := range domains {
		var secret v1.Secret
		secret, ok := domainToSecretMap[domain]
		if !ok {
			secret, ok = domainToSecretMap[utils.GetWildcardDomain(domain)]
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
