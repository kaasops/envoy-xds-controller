package tls

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/avast/retry-go/v4"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/go-logr/logr"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/k8s-utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/labels"
)

const (
	SecretLabelKey          = "envoy.kaasops.io/secret-type"
	SdsSecretLabelValue     = "sds-cached"
	WebhookSecretLabelValue = "webhook"
)

var (
	ErrManyParam = errors.New(`not supported using more then 1 param for configure TLS.
	You can choose one of 'secretRef', 'certManager', 'autoDiscovery'`)
	ErrZeroParam = errors.New(`need choose one 1 param for configure TLS. \
	You can choose one of 'secretRef', 'certManager', 'autoDiscovery'.\
	If you don't want use TLS for connection - don't install tlsConfig`)
	// ErrNodeIDsEmpty           = errors.New("NodeID not set")
	ErrSecretNotTLSType       = errors.New("kuberentes Secret is not a type TLS")
	ErrControlLabelNotExist   = errors.New("kuberentes Secret doesn't have control label")
	ErrControlLabelWrong      = errors.New("kubernetes Secret have label, but value not true")
	ErrCertManaferCRDNotExist = errors.New("cert Manager CRDs not exist. Perhaps Cert Manager is not installed in the Kubernetes cluster")
	ErrTlsConfigManyParam     = errors.New("сannot be installed Issuer and ClusterIssuer in 1 config")
	ErrDicoverNotFound        = errors.New("the secret with the certificate was not found for the domain")

	secretRefType     = "secretRef"
	certManagerType   = "certManagetType"
	autoDiscoveryType = "autoDiscoveryType"

	SecretLabel        = "envoy.kaasops.io/sds-cached"
	autoDiscoveryLabel = "envoy.kaasops.io/autoDiscovery"
	domainAnnotation   = "envoy.kaasops.io/domains"

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
	Config          *config.Config
	Namespace       string
	mu              sync.Mutex

	log logr.Logger
}

func New(
	client client.Client,
	dc *discovery.DiscoveryClient,
	config *config.Config,
	namespace string,

) *TlsConfigController {
	tcc := &TlsConfigController{
		client:          client,
		DiscoveryClient: dc,
		Config:          config,
		Namespace:       namespace,
	}

	tcc.log = log.Log.WithValues("package", "tls")

	return tcc
}

// Provide return map[string][]string where:
// key - name of TLS Certificate (is sDS cache (<NAMESPACE>-<NAME>)
// value - domains
func (cc *TlsConfigController) Provide(ctx context.Context, index map[string]corev1.Secret, vh *routev3.VirtualHost, tlsConfig *v1alpha1.TlsConfig) (map[string][]string, error) {
	errorList, err := cc.Validate(ctx, index, vh, tlsConfig)
	if err != nil {
		return nil, err
	}

	tlsType, _ := cc.getTLSType(tlsConfig)

	switch tlsType {
	case secretRefType:
		secretName := fmt.Sprintf("%s-%s",
			cc.Namespace,
			tlsConfig.SecretRef.Name,
		)
		return map[string][]string{
			secretName: vh.Domains,
		}, nil
	case certManagerType:
		return cc.certManagerProvide(ctx, vh, tlsConfig)
	case autoDiscoveryType:
		return cc.autoDiscoveryProvide(ctx, errorList, index, vh, tlsConfig)
	}

	return nil, nil
}

func (cc *TlsConfigController) certManagerProvide(ctx context.Context, vh *routev3.VirtualHost, tlsConfig *v1alpha1.TlsConfig) (map[string][]string, error) {
	certs := map[string][]string{}

	var wg sync.WaitGroup
	limit := make(chan struct{}, 10)
	for _, domain := range vh.Domains {
		wg.Add(1)
		limit <- struct{}{}
		go func(log logr.Logger, domain string) {
			defer func() {
				wg.Done()
				<-limit
			}()

			objName := strings.ToLower(strings.ReplaceAll(domain, ".", "-"))

			if err := cc.createCertificate(ctx, domain, objName, tlsConfig); err != nil {
				log.WithValues("Domain", domain).Error(err, "Error to create certificate for Domain")
			}

			cc.mu.Lock()
			certs[fmt.Sprintf("%s-%s", cc.Namespace, objName)] = []string{domain}
			cc.mu.Unlock()
		}(cc.log, domain)
	}

	wg.Wait()

	return certs, nil
}

func (cc *TlsConfigController) autoDiscoveryProvide(ctx context.Context, errorList map[string]string, index map[string]corev1.Secret, vh *routev3.VirtualHost, tlsConfig *v1alpha1.TlsConfig) (map[string][]string, error) {
	certs := map[string][]string{}

	var wg sync.WaitGroup
	limit := make(chan struct{}, 10)
	// time.Sleep(1 * time.Minute)
	for _, domain := range vh.Domains {
		// TODO: add logic for regexp, like: "~^v2-(?<projectid>\\d+)-(?<branch>\\w+\\-\\d+).site.com"
		if strings.Contains(domain, "^") || strings.Contains(domain, "~") {
			continue
		}
		wg.Add(1)
		limit <- struct{}{}
		go func(log logr.Logger, domain string, errorList map[string]string) {
			defer func() {
				wg.Done()
				<-limit
			}()

			cc.mu.Lock()
			_, ok := errorList[domain]
			cc.mu.Unlock()
			if ok {
				return
			}

			flag := false
			secret, ok := index[domain]

			if ok {
				cc.mu.Lock()
				v, ok := certs[fmt.Sprintf("%s-%s", cc.Namespace, secret.Name)]
				cc.mu.Unlock()
				if ok {
					v = append(v, domain)
					cc.mu.Lock()
					certs[fmt.Sprintf("%s-%s", cc.Namespace, secret.Name)] = v
					cc.mu.Unlock()
				} else {
					cc.mu.Lock()
					certs[fmt.Sprintf("%s-%s", cc.Namespace, secret.Name)] = []string{domain}
					cc.mu.Unlock()
				}
				flag = true
			}

			if !flag {
				log.Error(ErrDicoverNotFound, domain)
			}
		}(cc.log, domain, errorList)
	}
	wg.Wait()

	return certs, nil
}

func (cc *TlsConfigController) createCertificate(ctx context.Context, domain, objName string, tlsConfig *v1alpha1.TlsConfig) error {
	iType, iName, err := cc.getIssuer(tlsConfig)
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
			SecretTemplate: &cmapi.CertificateSecretTemplate{
				Labels: map[string]string{
					SecretLabel: "true",
				},
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
// 3. AutoDiscovery - try to find secret with TLS secret (based on domain annotation)
func (cc *TlsConfigController) Validate(ctx context.Context, index map[string]corev1.Secret, vh *routev3.VirtualHost, tlsConfig *v1alpha1.TlsConfig) (map[string]string, error) {
	if err := tlsConfig.Validate(); err != nil {
		return nil, err
	}

	tlsType, err := cc.getTLSType(tlsConfig)
	if err != nil {
		return nil, err
	}

	switch tlsType {
	case secretRefType:
		return nil, cc.checkSecretRef(ctx, vh, tlsConfig)
	case certManagerType:
		return nil, cc.checkCertManager(ctx, tlsConfig)
	case autoDiscoveryType:
		return cc.checkAutoDiscovery(ctx, index, vh, tlsConfig)
	}

	return nil, ErrZeroParam
}

// Check if SecretRef set. Checked only present secret in Kubernetes and have TLS type
func (cc *TlsConfigController) checkSecretRef(ctx context.Context, vh *routev3.VirtualHost, tlsConfig *v1alpha1.TlsConfig) error {
	namespacedName := types.NamespacedName{
		Name:      tlsConfig.SecretRef.Name,
		Namespace: cc.Namespace,
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
func (cc *TlsConfigController) checkCertManager(ctx context.Context, tlsConfig *v1alpha1.TlsConfig) error {
	// certManager installed in cluster (check CertManager CR)
	for _, kind := range certManagerKinds {
		exist, err := k8s.ResourceExists(cc.DiscoveryClient, cmapi.SchemeGroupVersion.String(), kind)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("%w. CRD: %s", ErrCertManaferCRDNotExist, kind)
		}
	}

	// Check Issuer exist in Kubernetes
	iType, iName, err := cc.getIssuer(tlsConfig)
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

// Check if AutoDiscovery set.
func (cc *TlsConfigController) checkAutoDiscovery(ctx context.Context, index map[string]corev1.Secret, vh *routev3.VirtualHost, tlsConfig *v1alpha1.TlsConfig) (map[string]string, error) {
	errorList := map[string]string{}
	var wg sync.WaitGroup
	limit := make(chan struct{}, 50)
	for _, vhDomain := range vh.Domains {
		wg.Add(1)
		limit <- struct{}{}

		go func(vhDomain string) {
			defer func() {
				wg.Done()
				<-limit
			}()
			flag := false
			_, ok := index[vhDomain]
			if ok {
				flag = true
			}
			if !flag {
				cc.mu.Lock()
				errorList[vhDomain] = fmt.Sprint(ErrDicoverNotFound)
				cc.mu.Unlock()
			}
		}(vhDomain)
	}
	wg.Wait()

	return errorList, nil
}

func (cc *TlsConfigController) getIssuer(tlsConfig *v1alpha1.TlsConfig) (iType, iName string, err error) {
	if tlsConfig.CertManager.Issuer != nil {
		if tlsConfig.CertManager.ClusterIssuer != nil {
			err = ErrTlsConfigManyParam
			return
		}
		iType = cmapi.IssuerKind
		iName = *tlsConfig.CertManager.Issuer
		return
	}

	if tlsConfig.CertManager.ClusterIssuer != nil {
		iType = cmapi.ClusterIssuerKind
		iName = *tlsConfig.CertManager.ClusterIssuer
		return
	}

	if *tlsConfig.CertManager.Enabled {
		if cc.Config.GetDefaultIssuer() != "" {
			iType = cmapi.ClusterIssuerKind
			iName = cc.Config.GetDefaultIssuer()
		}
		return
	}

	err = fmt.Errorf("issuer for Certificate not set")
	return
}

func (cc *TlsConfigController) getTLSType(tlsConfig *v1alpha1.TlsConfig) (string, error) {
	if tlsConfig.SecretRef != nil {
		if tlsConfig.CertManager != nil || tlsConfig.AutoDiscovery != nil {
			return "", ErrManyParam
		}
		return secretRefType, nil
	}

	if tlsConfig.CertManager != nil {
		if tlsConfig.AutoDiscovery != nil {
			return "", ErrManyParam
		}
		return certManagerType, nil
	}

	if tlsConfig.AutoDiscovery != nil {
		return autoDiscoveryType, nil
	}

	return "", ErrZeroParam
}

func (cc *TlsConfigController) IndexCertificateSecrets(ctx context.Context) (map[string]corev1.Secret, error) {
	indexedSecrets := make(map[string]corev1.Secret)
	secrets, err := cc.getCertificateSecrets(ctx)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		for _, domain := range strings.Split(secret.Annotations[domainAnnotation], ",") {
			_, ok := indexedSecrets[domain]
			if ok {
				cc.log.Info("Dublicate domain", "Domain:", domain)
				continue
			}
			indexedSecrets[domain] = secret
		}
	}
	return indexedSecrets, nil
}

func (cc *TlsConfigController) getCertificateSecrets(ctx context.Context) ([]corev1.Secret, error) {
	requirement, err := labels.NewRequirement(autoDiscoveryLabel, "==", []string{"true"})
	if err != nil {
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	listOpt := client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     cc.Namespace,
	}
	return k8s.ListSecret(ctx, cc.client, listOpt)
}
