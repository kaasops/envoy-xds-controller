package tls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/avast/retry-go/v4"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	k8s_utils "github.com/kaasops/k8s-utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrManyParam = errors.New(`not supported using more then 1 param for configure TLS.
	You can choose one of 'sdsName', 'secretRef', 'certManager'`)
	ErrZeroParam = errors.New(`need choose one 1 param for configure TLS. \
	You can choose one of 'sdsName', 'secretRef', 'certManager'.\
	If you don't want use TLS for connection - don't install tlsConfig`)
	ErrNodeIDsEmpty           = errors.New("NodeID not set")
	ErrTlsConfigNotExist      = errors.New("tls Config not set")
	ErrSecretNotTLSType       = errors.New("kuberentes Secret is not a type TLS")
	ErrControlLabelNotExist   = errors.New("kuberentes Secret doesn't have control label")
	ErrControlLabelWrong      = errors.New("kubernetes Secret have label, but value not true")
	ErrCertManaferCRDNotExist = errors.New("cert Manager CRDs not exist. Perhaps Cert Manager is not installed in the Kubernetes cluster")
	ErrTlsConfigManyParam     = errors.New("сannot be installed Issuer and ClusterIssuer in 1 config")

	secretRefType   = "secretRef"
	certManagerType = "certManagetType"

	SecretLabel = "envoy.kaasops.io/sds-cached"

	certManagerKinds = []string{
		cmapi.ClusterIssuerKind,
		cmapi.IssuerKind,
		cmapi.CertificateKind,
		cmapi.CertificateRequestKind,
	}
)

type TlsConfigController struct {
	client          client.Client
	DiscoveryClient *discovery.DiscoveryClient
	TlsConfig       *v1alpha1.TlsConfig
	VirtualHost     *routev3.VirtualHost
	NodeIDs         []string
	Config          config.Config
	Namespace       string
}

func New(
	client client.Client,
	dc *discovery.DiscoveryClient,
	tlsConfig *v1alpha1.TlsConfig,
	vh *routev3.VirtualHost,
	nodeIDs []string,
	config config.Config,
	namespace string,

) *TlsConfigController {
	return &TlsConfigController{
		client:          client,
		DiscoveryClient: dc,
		TlsConfig:       tlsConfig,
		VirtualHost:     vh,
		NodeIDs:         nodeIDs,
		Config:          config,
		Namespace:       namespace,
	}
}

// Provide return map[string][]string where:
// key - name of TLS Certificate (is sDS cache (<NAMESPACE>-<NAME>)
// value - domains
func (cc *TlsConfigController) Provide(ctx context.Context) (map[string][]string, error) {
	err := cc.Validate(ctx)
	if err != nil {
		return nil, err
	}

	tlsType, _ := cc.getTLSType()

	if tlsType == secretRefType {
		secretName := fmt.Sprintf("%s-%s",
			cc.TlsConfig.SecretRef.Namespace,
			cc.TlsConfig.SecretRef.Name,
		)
		return map[string][]string{
			secretName: cc.VirtualHost.Domains,
		}, nil
	}

	if tlsType == certManagerType {
		return cc.certManagerProvide(ctx)
	}

	return nil, nil
}

func (cc *TlsConfigController) certManagerProvide(ctx context.Context) (map[string][]string, error) {
	certs := map[string][]string{}
	for _, domain := range cc.VirtualHost.Domains {
		objName := strings.ReplaceAll(domain, ".", "-")

		if err := cc.createCertificate(ctx, domain, objName); err != nil {
			return nil, err
		}

		certs[fmt.Sprintf("%s-%s", cc.Namespace, objName)] = []string{domain}
	}

	return certs, nil
}

func (cc *TlsConfigController) createCertificate(ctx context.Context, domain, objName string) error {
	iType, iName, err := cc.getIssuer()
	if err != nil {
		return err
	}

	cert := &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: cc.Namespace,
		},
		Spec: cmapi.CertificateSpec{
			SecretName: objName,
			IsCA:       false,
			DNSNames:   []string{domain},
			IssuerRef: cmmeta.ObjectReference{
				Name: iName,
				Kind: iType,
			},
		},
	}

	if err := cc.client.Create(ctx, cert); err != nil {
		if api_errors.IsAlreadyExists(err) {
			existing := &cmapi.Certificate{}
			err := cc.client.Get(ctx, client.ObjectKeyFromObject(cert), existing)
			if err != nil {
				return err
			}

			// init Interface for compare
			desiredFields := []interface{}{
				cert.GetAnnotations(),
				cert.GetLabels(),
				cert.Spec,
			}
			existingFields := []interface{}{
				existing.GetAnnotations(),
				existing.GetLabels(),
				existing.Spec,
			}

			// Compare
			if !equality.Semantic.DeepDerivative(desiredFields, existingFields) {
				// Update if not equal
				existing.Labels = cert.Labels
				existing.Annotations = cert.Annotations
				existing.Spec = cert.Spec
				return cc.client.Update(ctx, existing)
			}
			return nil

		}
	}

	// Check secret created
	secretNamespacedName := types.NamespacedName{
		Name:      objName,
		Namespace: cc.Namespace,
	}
	err = retry.Do(
		func() error {
			err := cc.checkKubernetesSecret(ctx, secretNamespacedName)
			if api_errors.IsNotFound(err) {
				return nil
			}
			return err
		},
		retry.Attempts(5),
	)

	return err
}

// ValidateTls check TlsConfig for Virtual Service.
// Tls can be provide by 2 types:
// 1. SecretRef - Use TLS from exist Kubernetes Secret
// 2. CertManager - Use CertManager for create Kubernetes Secret with certificate and
func (cc *TlsConfigController) Validate(ctx context.Context) error {
	// Check if TLS not used
	if cc.TlsConfig == nil {
		return ErrTlsConfigNotExist
	}

	if len(cc.NodeIDs) == 0 {
		return ErrNodeIDsEmpty
	}

	tlsType, err := cc.getTLSType()
	if err != nil {
		return err
	}

	if tlsType == secretRefType {
		err := cc.checkSecretRef(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	if tlsType == certManagerType {
		err := cc.checkCertManager(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	return ErrZeroParam
}

// Check if SecretRef set. Checked only present secret in Kubernetes and have TLS type
func (cc *TlsConfigController) checkSecretRef(ctx context.Context) error {
	namespacedName := types.NamespacedName{
		Name:      cc.TlsConfig.SecretRef.Name,
		Namespace: cc.TlsConfig.SecretRef.Namespace,
	}

	return cc.checkKubernetesSecret(ctx, namespacedName)
}

func (cc *TlsConfigController) checkKubernetesSecret(ctx context.Context, nn types.NamespacedName) error {
	secret := &corev1.Secret{}

	// Check secret exist in Kubernetes
	err := cc.client.Get(ctx, nn, secret)
	if err != nil {
		return err
	}

	// Check Secret type
	if secret.Type != corev1.SecretTypeTLS {
		return fmt.Errorf("%w. %s/%s", ErrSecretNotTLSType, nn.Name, nn.Namespace)
	}

	// Check control label
	labels := secret.Labels
	value, ok := labels[SecretLabel]
	if !ok {
		return fmt.Errorf("%w. %s/%s", ErrControlLabelNotExist, nn.Name, nn.Namespace)
	}
	if value != "true" {
		return fmt.Errorf("%w. %s/%s", ErrControlLabelWrong, nn.Name, nn.Namespace)
	}

	return nil

}

// Check if CertManager set.
func (cc *TlsConfigController) checkCertManager(ctx context.Context) error {
	// certManager installed in cluster (check CertManager CR)
	for _, kind := range certManagerKinds {
		exist, err := k8s_utils.ResourceExists(cc.DiscoveryClient, cmapi.SchemeGroupVersion.String(), kind)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("%w. CRD: %s", ErrCertManaferCRDNotExist, kind)
		}
	}

	// Check Issuer exist in Kubernetes
	iType, iName, err := cc.getIssuer()
	if err != nil {
		return err
	}
	namespacedName := types.NamespacedName{
		Name:      iName,
		Namespace: cc.Namespace,
	}

	if iType == cmapi.IssuerKind {
		issuer := &cmapi.Issuer{}
		err := cc.client.Get(ctx, namespacedName, issuer)
		if err != nil {
			return err
		}
	}
	if iType == cmapi.ClusterIssuerKind {
		issuer := &cmapi.ClusterIssuer{}
		err := cc.client.Get(ctx, namespacedName, issuer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cc *TlsConfigController) getIssuer() (iType, iName string, err error) {
	if cc.TlsConfig.CertManager.Issuer != nil {
		if cc.TlsConfig.CertManager.ClusterIssuer != nil {
			err = ErrTlsConfigManyParam
			return
		}
		iType = cmapi.IssuerKind
		iName = *cc.TlsConfig.CertManager.Issuer
		return
	}

	if cc.TlsConfig.CertManager.ClusterIssuer != nil {
		iType = cmapi.ClusterIssuerKind
		iName = *cc.TlsConfig.CertManager.ClusterIssuer
		return
	}

	if *cc.TlsConfig.CertManager.Enabled {
		if cc.Config.GetDefaultIssuer() != "" {
			iType = cmapi.ClusterIssuerKind
			iName = cc.Config.GetDefaultIssuer()
		}
		return
	}

	err = fmt.Errorf("issuer for Certificate not set")
	return
}

func (cc *TlsConfigController) getTLSType() (string, error) {
	if cc.TlsConfig.SecretRef != nil {
		if cc.TlsConfig.CertManager != nil {
			return "", ErrManyParam
		}

		return secretRefType, nil
	}

	if cc.TlsConfig.CertManager != nil {
		return certManagerType, nil
	}
	return "", ErrZeroParam
}
