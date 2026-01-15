package secrets

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/utils"
)

// Builder handles TLS configuration for secrets
type Builder struct {
	store store.Store
}

// NewBuilder creates a new secret builder
func NewBuilder(store store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// GetTLSType determines the TLS configuration type from TlsConfig
func GetTLSType(tlsConfig *v1alpha1.TlsConfig) (string, error) {
	if tlsConfig == nil {
		return "", fmt.Errorf("TLS config is nil")
	}

	var configCount int
	var configType string

	if tlsConfig.SecretRef != nil {
		configCount++
		configType = utils.SecretRefType
	}

	if tlsConfig.AutoDiscovery != nil && *tlsConfig.AutoDiscovery {
		configCount++
		configType = utils.AutoDiscoveryType
	}

	switch configCount {
	case 0:
		return "", fmt.Errorf("no TLS configuration specified")
	case 1:
		return configType, nil
	default:
		return "", fmt.Errorf("multiple TLS configuration types specified (only one allowed)")
	}
}

// TLSBuilder interface implementation

// GetTLSType determines the TLS configuration type from TlsConfig
func (b *Builder) GetTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error) {
	return GetTLSType(vsTLSConfig)
}

// GetSecretNameToDomains maps domains to secrets based on the VirtualService's TLS configuration
func (b *Builder) GetSecretNameToDomains(
	vs *v1alpha1.VirtualService,
	domains []string,
) (map[helpers.NamespacedName][]string, error) {
	if vs.Spec.TlsConfig == nil {
		return nil, fmt.Errorf("TLS configuration is missing in VirtualService")
	}

	tlsType, err := GetTLSType(vs.Spec.TlsConfig)
	if err != nil {
		return nil, err
	}

	switch tlsType {
	case utils.SecretRefType:
		return b.getSecretNameToDomainsViaSecretRef(vs.Spec.TlsConfig.SecretRef, vs.Namespace, domains), nil
	case utils.AutoDiscoveryType:
		return b.getSecretNameToDomainsViaAutoDiscovery(domains, vs.Namespace)
	default:
		return nil, fmt.Errorf("unknown TLS type: %s", tlsType)
	}
}

// getSecretNameToDomainsViaSecretRef maps domains to a single secret for secretRef type
func (b *Builder) getSecretNameToDomainsViaSecretRef(
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

// getSecretNameToDomainsViaAutoDiscovery maps domains to secrets based on auto-discovery.
// Uses GetDomainSecretWithWildcardFallback to prefer valid wildcard certificates
// over expired exact certificates.
func (b *Builder) getSecretNameToDomainsViaAutoDiscovery(
	domains []string,
	preferredNamespace string,
) (map[helpers.NamespacedName][]string, error) {
	logger := log.Log.WithName("secrets-builder")
	m := make(map[helpers.NamespacedName][]string)

	for _, domain := range domains {
		// Use wildcard fallback method with detailed info for logging
		result := b.store.GetDomainSecretWithWildcardFallbackInfo(domain, preferredNamespace)

		if result.Secret == nil {
			return nil, fmt.Errorf("can't find secret for domain %s", domain)
		}

		// Log when wildcard fallback is used due to expired/unknown exact cert
		if result.UsedWildcard && result.FallbackReason != "" {
			logger.Info("Using wildcard certificate fallback",
				"domain", domain,
				"reason", result.FallbackReason,
				"exactSecret", result.ExactSecretName,
				"wildcardSecret", result.Secret.Namespace+"/"+result.Secret.Name,
				"namespace", preferredNamespace,
			)
		}

		nn := helpers.NamespacedName{Namespace: result.Secret.Namespace, Name: result.Secret.Name}
		m[nn] = append(m[nn], domain)
	}

	return m, nil
}
