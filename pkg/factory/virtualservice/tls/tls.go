package tls

import (
	"context"
	"fmt"
	"strings"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

var (
	secretLabel = labels.Set{options.SecretLabelKey: options.SdsSecretLabelValue}

	certManagerKinds = []string{
		cmapi.ClusterIssuerKind,
		cmapi.IssuerKind,
		cmapi.CertificateKind,
		cmapi.CertificateRequestKind,
	}
)

type Tls struct {
	ErrorDomains            map[string]string
	CertificatesWithDomains map[string][]string
}

type TlsFactory struct {
	*v1alpha1.TlsConfig

	client          client.Client
	DiscoveryClient *discovery.DiscoveryClient
	defaultIssuer   string
	Namespace       string
	Domains         []string

	CertificatesIndex map[string]corev1.Secret

	log logr.Logger
}

func NewTlsFactory(
	ctx context.Context,
	tlsConfig *v1alpha1.TlsConfig,
	client client.Client,
	dc *discovery.DiscoveryClient,
	defaultIssuer string,
	namespace string,
	index map[string]corev1.Secret,
) *TlsFactory {
	tf := &TlsFactory{
		TlsConfig:         tlsConfig,
		client:            client,
		DiscoveryClient:   dc,
		Namespace:         namespace,
		CertificatesIndex: index,
		defaultIssuer:     defaultIssuer,
	}

	tf.log = log.Log.WithValues("factory", "virtualservice", "package", "tls")

	return tf
}

func (tf *TlsFactory) Provide(ctx context.Context, domains []string) (*Tls, error) {
	tls := &Tls{
		ErrorDomains:            map[string]string{},
		CertificatesWithDomains: map[string][]string{},
	}

	tlsType, err := tf.getTLSType()
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get TlsConfig type")
	}

	switch tlsType {
	case v1alpha1.SecretRefType:
		err := tf.provideSecretRef(ctx, tls)
		if err != nil {
			return nil, errors.Wrap(err, "cannot provide SecretRef")
		}
		return tls, nil
	case v1alpha1.CertManagerType:
		err := tf.provideCertManager(ctx, tls)
		if err != nil {
			return nil, errors.Wrap(err, "cannot provide CertManager")
		}
	case v1alpha1.AutoDiscoveryType:
		err := tf.provideAutoDiscovery(ctx, tls)
		if err != nil {
			return nil, errors.Wrap(err, "cannot provide AutoDicsovery")
		}
	}

	return tls, nil
}

func (tf *TlsFactory) provideSecretRef(ctx context.Context, tls *Tls) error {
	namespacedName := types.NamespacedName{
		Name:      tf.TlsConfig.SecretRef.Name,
		Namespace: tf.Namespace,
	}

	// Check Secret exist in Kubernetes
	secret := &corev1.Secret{}
	err := tf.client.Get(ctx, namespacedName, secret)
	if err != nil {
		return errors.NewUKS("cannot get secret with TLS certificate")
	}

	// Check Secret type
	if secret.Type != corev1.SecretTypeTLS {
		return errors.NewUKS("secret has wrong type")
	}

	// Create domain row
	secretName := fmt.Sprintf("%s-%s",
		tf.Namespace,
		tf.TlsConfig.SecretRef.Name,
	)
	tls.CertificatesWithDomains[secretName] = tf.Domains

	return nil
}

func (tf *TlsFactory) provideCertManager(ctx context.Context, tls *Tls) error {
	// Check CertManager CRs exist in Kubernetes
	for _, kind := range certManagerKinds {
		exist, err := k8s.ResourceExists(tf.DiscoveryClient, cmapi.SchemeGroupVersion.String(), kind)
		if err != nil {
			return err
		}
		if !exist {
			return errors.NewUKS(errors.CertManaferCRDNotExistMessage)
		}
	}

	// Get Issuer Name and Type
	iType, iName, err := tf.getIssuerTypeName()
	if err != nil {
		return errors.WrapUKS(err, "cannot get issuer name and type")
	}

	// Check Issuer exist in Kubernetes
	namespacedName := types.NamespacedName{
		Name:      iName,
		Namespace: tf.Namespace,
	}

	if iType == cmapi.IssuerKind {
		issuer := &cmapi.Issuer{}
		err := tf.client.Get(ctx, namespacedName, issuer)
		if err != nil {
			return errors.WrapUKS(err, "cannot get issuer in Kubernetes")
		}
	}
	if iType == cmapi.ClusterIssuerKind {
		issuer := &cmapi.ClusterIssuer{}
		err := tf.client.Get(ctx, namespacedName, issuer)
		if err != nil {
			return errors.WrapUKS(err, "cannot get cluster issuer in Kubernetes")
		}
	}

	// TODO: collect dif domains with same wildcard to 1 certificate
	// Create Certificates for all domains
	for _, domain := range tf.Domains {
		objName := strings.ToLower(strings.ReplaceAll(domain, ".", "-"))

		if err := tf.createCertificate(ctx, domain, objName); err != nil {
			tls.ErrorDomains[domain] = errors.CreateCertificateMessage
			tf.log.WithValues("Domain", domain).Error(err, errors.CreateCertificateMessage)
		}
		tls.CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, objName)] = []string{domain}
	}

	return nil
}

func (tf *TlsFactory) provideAutoDiscovery(ctx context.Context, tls *Tls) error {

	for _, domain := range tf.Domains {
		// If domain alredy exist in Error List - skip
		_, ok := tls.ErrorDomains[domain]
		if ok {
			continue
		}

		// TODO (Webhook or validate all)
		if strings.Contains(domain, "^") || strings.Contains(domain, "~") {
			tls.ErrorDomains[domain] = errors.RegexDomainMessage
		}

		// Validate certificate exist in index!
		secret, ok := tf.CertificatesIndex[domain]
		if ok {
			d, ok := tls.CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, secret.Name)]
			if ok {
				d = append(d, domain)
				tls.CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, secret.Name)] = d
			} else {
				tls.CertificatesWithDomains[fmt.Sprintf("%s-%s", tf.Namespace, secret.Name)] = []string{domain}
			}
		} else {
			tls.ErrorDomains[domain] = errors.DicoverNotFoundMessage
		}
	}

	return nil
}
