package secrets

import (
	"fmt"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	v1 "k8s.io/api/core/v1"
)

// Builder handles the construction of TLS secrets
type Builder struct {
	store store.Store
}

// NewBuilder creates a new secret builder
func NewBuilder(store store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// BuildSecrets builds TLS secrets from VirtualService configuration
func (b *Builder) BuildSecrets(vs *v1alpha1.VirtualService, secretNameToDomains map[helpers.NamespacedName][]string) ([]*tlsv3.Secret, []helpers.NamespacedName, error) {
	if len(secretNameToDomains) == 0 {
		return nil, nil, nil
	}

	secrets := make([]*tlsv3.Secret, 0, len(secretNameToDomains))
	usedSecrets := make([]helpers.NamespacedName, 0, len(secretNameToDomains))

	for secretName := range secretNameToDomains {
		secret, err := b.buildSecret(secretName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build secret %s: %w", secretName.String(), err)
		}
		secrets = append(secrets, secret)
		usedSecrets = append(usedSecrets, secretName)
	}

	return secrets, usedSecrets, nil
}

// buildSecret builds a single TLS secret from Kubernetes secret
func (b *Builder) buildSecret(secretName helpers.NamespacedName) (*tlsv3.Secret, error) {
	k8sSecret := b.store.GetSecret(secretName)
	if k8sSecret == nil {
		return nil, fmt.Errorf("Kubernetes secret %s not found", secretName.String())
	}

	// Validate secret type and data
	if err := b.validateSecretData(k8sSecret); err != nil {
		return nil, fmt.Errorf("invalid secret data for %s: %w", secretName.String(), err)
	}

	// Extract certificate and private key
	certData, keyData, err := b.extractCertificateData(k8sSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to extract certificate data from %s: %w", secretName.String(), err)
	}

	// Build TLS certificate configuration
	tlsCert := &tlsv3.TlsCertificate{
		CertificateChain: &corev3.DataSource{
			Specifier: &corev3.DataSource_InlineBytes{
				InlineBytes: certData,
			},
		},
		PrivateKey: &corev3.DataSource{
			Specifier: &corev3.DataSource_InlineBytes{
				InlineBytes: keyData,
			},
		},
	}

	// Create Envoy TLS secret
	secret := &tlsv3.Secret{
		Name: secretName.String(),
		Type: &tlsv3.Secret_TlsCertificate{
			TlsCertificate: tlsCert,
		},
	}

	// Validate the constructed secret
	if err := secret.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate TLS secret %s: %w", secretName.String(), err)
	}

	return secret, nil
}

// validateSecretData validates that the Kubernetes secret contains required TLS data
func (b *Builder) validateSecretData(secret *v1.Secret) error {
	if secret.Type != v1.SecretTypeTLS && secret.Type != v1.SecretTypeOpaque {
		return fmt.Errorf("unsupported secret type: %s (expected %s or %s)",
			secret.Type, v1.SecretTypeTLS, v1.SecretTypeOpaque)
	}

	if secret.Data == nil {
		return fmt.Errorf("secret data is nil")
	}

	// Check for required certificate data
	if _, exists := secret.Data[v1.TLSCertKey]; !exists {
		return fmt.Errorf("missing certificate data (key: %s)", v1.TLSCertKey)
	}

	// Check for required private key data
	if _, exists := secret.Data[v1.TLSPrivateKeyKey]; !exists {
		return fmt.Errorf("missing private key data (key: %s)", v1.TLSPrivateKeyKey)
	}

	// Validate certificate data is not empty
	certData := secret.Data[v1.TLSCertKey]
	if len(certData) == 0 {
		return fmt.Errorf("certificate data is empty")
	}

	// Validate private key data is not empty
	keyData := secret.Data[v1.TLSPrivateKeyKey]
	if len(keyData) == 0 {
		return fmt.Errorf("private key data is empty")
	}

	return nil
}

// extractCertificateData extracts certificate and private key data from Kubernetes secret
func (b *Builder) extractCertificateData(secret *v1.Secret) ([]byte, []byte, error) {
	certData, exists := secret.Data[v1.TLSCertKey]
	if !exists {
		return nil, nil, fmt.Errorf("certificate data not found")
	}

	keyData, exists := secret.Data[v1.TLSPrivateKeyKey]
	if !exists {
		return nil, nil, fmt.Errorf("private key data not found")
	}

	// Make copies to avoid potential mutation issues
	certCopy := make([]byte, len(certData))
	copy(certCopy, certData)

	keyCopy := make([]byte, len(keyData))
	copy(keyCopy, keyData)

	return certCopy, keyCopy, nil
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

// ValidateTLSConfiguration validates TLS configuration without building secrets
func (b *Builder) ValidateTLSConfiguration(tlsConfig *v1alpha1.TlsConfig, domains []string, store store.Store) error {
	if tlsConfig == nil {
		return nil // TLS is optional
	}

	tlsType, err := GetTLSType(tlsConfig)
	if err != nil {
		return err
	}

	switch tlsType {
	case utils.SecretRefType:
		return b.validateSecretRefConfiguration(tlsConfig.SecretRef, store)
	case utils.AutoDiscoveryType:
		return b.validateAutoDiscoveryConfiguration(domains, store)
	default:
		return fmt.Errorf("unknown TLS configuration type: %s", tlsType)
	}
}

// validateSecretRefConfiguration validates SecretRef configuration
func (b *Builder) validateSecretRefConfiguration(secretRef *v1alpha1.ResourceRef, store store.Store) error {
	if secretRef == nil {
		return fmt.Errorf("secretRef is nil")
	}

	if secretRef.Name == "" {
		return fmt.Errorf("secretRef name is empty")
	}

	// Check if secret exists in store
	secretNamespace := helpers.GetNamespace(secretRef.Namespace, "")
	if secretNamespace == "" {
		return fmt.Errorf("secretRef namespace is required")
	}

	secretName := helpers.NamespacedName{
		Namespace: secretNamespace,
		Name:      secretRef.Name,
	}

	secret := store.GetSecret(secretName)
	if secret == nil {
		return fmt.Errorf("secret %s not found", secretName.String())
	}

	return b.validateSecretData(secret)
}

// validateAutoDiscoveryConfiguration validates auto-discovery configuration
func (b *Builder) validateAutoDiscoveryConfiguration(domains []string, store store.Store) error {
	if len(domains) == 0 {
		return fmt.Errorf("no domains specified for auto-discovery")
	}

	domainSecretsMap := store.MapDomainSecrets()
	if len(domainSecretsMap) == 0 {
		return fmt.Errorf("no domain-secret mappings available for auto-discovery")
	}

	// Validate that secrets exist for all non-wildcard domains
	for _, domain := range domains {
		if domain == "" || domain == "*" {
			continue
		}

		secret, exists := domainSecretsMap[domain]
		if !exists {
			return fmt.Errorf("no secret mapping found for domain %s", domain)
		}

		// Create NamespacedName from secret
		secretName := helpers.NamespacedName{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}

		k8sSecret := store.GetSecret(secretName)
		if k8sSecret == nil {
			return fmt.Errorf("secret %s referenced by domain %s not found", secretName.String(), domain)
		}

		if err := b.validateSecretData(k8sSecret); err != nil {
			return fmt.Errorf("invalid secret %s for domain %s: %w", secretName.String(), domain, err)
		}
	}

	return nil
}
