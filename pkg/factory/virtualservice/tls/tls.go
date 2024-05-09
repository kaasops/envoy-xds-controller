package tls

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

var (
	secretLabel = labels.Set{options.SecretLabelKey: options.SdsSecretLabelValue}
)

type TlsFactory struct {
	*v1alpha1.TlsConfig

	client        client.Client
	defaultIssuer string
	Namespace     string

	CertificatesIndex map[string]corev1.Secret

	log logr.Logger
}

func NewTlsFactory(
	ctx context.Context,
	tlsConfig *v1alpha1.TlsConfig,
	client client.Client,
	defaultIssuer string,
	namespace string,
	index map[string]corev1.Secret,
) *TlsFactory {
	tf := &TlsFactory{
		TlsConfig:         tlsConfig,
		client:            client,
		Namespace:         namespace,
		CertificatesIndex: index,
		defaultIssuer:     defaultIssuer,
	}

	tf.log = log.Log.WithValues("factory", "virtualservice", "package", "tls")

	return tf
}

func (tf *TlsFactory) Provide(ctx context.Context, domains []string) (map[string][]string, error) {
	tlsType, err := tf.GetTLSType()
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get TlsConfig type")
	}

	switch tlsType {
	case v1alpha1.SecretRefType:
		return tf.provideSecretRef(ctx, domains)
	case v1alpha1.AutoDiscoveryType:
		return tf.provideAutoDiscovery(ctx, domains)
	}

	return nil, nil
}

func (tf *TlsFactory) provideSecretRef(ctx context.Context, domains []string) (map[string][]string, error) {
	// Create domain row
	secretName := fmt.Sprintf("%s-%s",
		tf.Namespace,
		tf.TlsConfig.SecretRef.Name,
	)

	return map[string][]string{
		secretName: domains,
	}, nil
}

func (tf *TlsFactory) provideAutoDiscovery(ctx context.Context, domains []string) (map[string][]string, error) {
	CertificatesWithDomains := make(map[string][]string)

	for _, domain := range domains {
		var secret corev1.Secret
		// Validate certificate exist in index!
		secret, ok := tf.CertificatesIndex[domain]
		if !ok {
			wildcardDomain := utils.GetWildcardDomain(domain)
			secret, ok = tf.CertificatesIndex[wildcardDomain]
			if !ok {
				return CertificatesWithDomains, errors.Newf(fmt.Sprintf("domain: %v. %v", domain, errors.DicoverNotFoundMessage))
			}
		}

		d, ok := CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, secret.Name)]
		if ok {
			d = append(d, domain)
			CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, secret.Name)] = d
		} else {
			CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, secret.Name)] = []string{domain}
		}
	}

	return CertificatesWithDomains, nil
}
