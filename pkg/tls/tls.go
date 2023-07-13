package tls

import (
	"context"
	"errors"
	"fmt"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	k8s_utils "github.com/kaasops/k8s-utils"
	corev1 "k8s.io/api/core/v1"
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
	ErrNodeIDsEmpty = errors.New("NodeID not set")
	ErrSsdNotExist  = errors.New("")

	secretRefType   = "secretRef"
	certManagerType = "certManagetType"

	secretLabel = "envoy.kaasops.io/sds-cached"

	certManagerKinds = []string{
		cmapi.ClusterIssuerKind,
		cmapi.IssuerKind,
		cmapi.CertificateKind,
		cmapi.CertificateRequestKind,
	}
)

type TlsConfigController struct {
	client             client.Client
	DiscoveryInterface *discovery.DiscoveryInterface
	TlsConfig          *v1alpha1.TlsConfig
	VirtualHost        *routev3.VirtualHost
	NodeIDs            []string
	Config             config.Config
	Namespace          string
}

func New(
	client client.Client,
	di *discovery.DiscoveryInterface,
	tlsConfig *v1alpha1.TlsConfig,
	vh *routev3.VirtualHost,
	nodeIDs []string,
	config config.Config,
	namespace string,

) *TlsConfigController {
	return &TlsConfigController{
		client:             client,
		DiscoveryInterface: di,
		TlsConfig:          tlsConfig,
		VirtualHost:        vh,
		NodeIDs:            nodeIDs,
		Config:             config,
		Namespace:          namespace,
	}
}

// Provide return map[string]string where:
// key - name of TLS Certificate (name in sDS cache and Kubernetes Secret - the same)
// value - ndomains
func (cc *TlsConfigController) Provide(ctx context.Context) (map[string][]string, error) {
	err := cc.Validate(ctx)
	if err != nil {
		return nil, err
	}

	tlsType, err := cc.getTLSType()
	if err != nil {
		return nil, err
	}

	if tlsType == secretRefType {
		secretName := fmt.Sprintf("%s-%s",
			cc.TlsConfig.SecretRef.Name,
			cc.TlsConfig.SecretRef.Namespace,
		)
		return map[string][]string{
			secretName: cc.VirtualHost.Domains,
		}, nil
	}

	if tlsType == certManagerType {
		fmt.Println("HATE")
	}

	return nil, nil
}

// ValidateTls check TlsConfig for Virtual Service.
// Tls can be provide by 2 types:
// 1. SecretRef - Use TLS from exist Kubernetes Secret
// 2. CertManager - Use CertManager for create Kubernetes Secret with certificate and
func (cc *TlsConfigController) Validate(ctx context.Context) error {
	// Check if TLS not used
	if cc.TlsConfig == nil {
		return nil
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
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{
		Name:      cc.TlsConfig.SecretRef.Name,
		Namespace: cc.TlsConfig.SecretRef.Namespace,
	}

	// Check secret exist in Kubernetes
	err := cc.client.Get(ctx, namespacedName, secret)
	if err != nil {
		return err
	}

	// Check Secret type
	if secret.Type != corev1.SecretTypeTLS {
		return fmt.Errorf("kuberentes Secret %s in namespace %s is not a type TLS", namespacedName.Name, namespacedName.Namespace)
	}

	// Check control label
	labels := secret.Labels
	value, ok := labels[secretLabel]
	if !ok {
		return fmt.Errorf("kubernetes Secret %s in namespace %s dont't have label %s", namespacedName.Name, namespacedName.Namespace, secretLabel)
	}
	if value != "true" {
		return fmt.Errorf("kubernetes Secret %s in namespace %s have label %s, but value not True", namespacedName.Name, namespacedName.Namespace, secretLabel)
	}

	return nil
}

// Check if CertManager set.
func (cc *TlsConfigController) checkCertManager(ctx context.Context) error {
	// certManager installed in cluster (check CertManager CR)
	for _, kind := range certManagerKinds {
		exist, err := k8s_utils.ResourceExists(*cc.DiscoveryInterface, cmapi.SchemeGroupVersion.String(), kind)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("CRD %s not exist. Perhaps Cert Manager is not installed in the Kubernetes cluster", kind)
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

func (cc *TlsConfigController) getIssuer() (iType, issuer string, err error) {
	if cc.TlsConfig.CertManager.Issuer != nil {
		if cc.TlsConfig.CertManager.ClusterIssuer != nil {
			err = fmt.Errorf("сannot be installed Issuer and ClusterIssuer in 1 config")
			return
		}
		iType = cmapi.IssuerKind
		issuer = *cc.TlsConfig.CertManager.Issuer
		return
	}

	if cc.TlsConfig.CertManager.ClusterIssuer != nil {
		iType = cmapi.ClusterIssuerKind
		issuer = *cc.TlsConfig.CertManager.ClusterIssuer
		return
	}

	if cc.Config.GetDefaultIssuer() != "" {
		iType = cmapi.ClusterIssuerKind
		issuer = cc.Config.GetDefaultIssuer()
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
