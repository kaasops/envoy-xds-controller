package tls

import (
	"context"
	"strings"

	"github.com/avast/retry-go/v4"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/k8s-utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (tf *TlsFactory) getTLSType() (string, error) {
	if tf.TlsConfig.SecretRef != nil {
		if tf.TlsConfig.CertManager != nil || tf.TlsConfig.AutoDiscovery != nil {
			return "", errors.NewUKS(errors.ManyParamMessage)
		}
		return v1alpha1.SecretRefType, nil
	}

	if tf.TlsConfig.CertManager != nil {
		if tf.TlsConfig.AutoDiscovery != nil {
			return "", errors.NewUKS(errors.ManyParamMessage)
		}
		return v1alpha1.CertManagerType, nil
	}

	if tf.TlsConfig.AutoDiscovery != nil {
		return v1alpha1.AutoDiscoveryType, nil
	}

	return "", errors.NewUKS(errors.ZeroParamMessage)
}

// indexCertificateSecrets indexed all certificates for cache
func (cc *TlsFactory) indexCertificateSecrets(ctx context.Context) (map[string]corev1.Secret, error) {
	indexedSecrets := make(map[string]corev1.Secret)
	secrets, err := cc.getCertificateSecrets(ctx)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		for _, domain := range strings.Split(secret.Annotations[options.DomainAnnotation], ",") {
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

// getCertificateSecrets gets all certificates from Kubernetes secrets
func (tf *TlsFactory) getCertificateSecrets(ctx context.Context) ([]corev1.Secret, error) {
	requirement, err := labels.NewRequirement(options.AutoDiscoveryLabel, "==", []string{"true"})
	if err != nil {
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	listOpt := client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     tf.Namespace,
	}
	return k8s.ListSecret(ctx, tf.client, listOpt)
}

// getIssuerTypeName gets Cert Manager Issuer Type and Name
func (tf *TlsFactory) getIssuerTypeName() (string, string, error) {
	if tf.TlsConfig.CertManager.Issuer != nil {
		if tf.TlsConfig.CertManager.ClusterIssuer != nil {
			return "", "", errors.New(errors.TlsConfigManyParamMessage)
		}
		iType := cmapi.IssuerKind
		iName := *tf.TlsConfig.CertManager.Issuer
		return iType, iName, nil
	}

	if tf.TlsConfig.CertManager.ClusterIssuer != nil {
		iType := cmapi.ClusterIssuerKind
		iName := *tf.TlsConfig.CertManager.ClusterIssuer
		return iType, iName, nil
	}

	if *tf.TlsConfig.CertManager.Enabled {
		if tf.Config.GetDefaultIssuer() != "" {
			iType := cmapi.ClusterIssuerKind
			iName := tf.Config.GetDefaultIssuer()
			return iType, iName, nil
		}
	}

	return "", "", errors.New("issuer for Certificate not set")
}

func (tf *TlsFactory) createCertificate(ctx context.Context, domain, objName string) error {
	iType, iName, err := tf.getIssuerTypeName()
	if err != nil {
		return errors.WrapUKS(err, "cannot get issuer name and type")
	}

	cert := &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: tf.Namespace,
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
				Labels: secretLabel,
			},
		},
	}

	if err := tf.client.Create(ctx, cert); err != nil {
		if api_errors.IsAlreadyExists(err) {
			existing := &cmapi.Certificate{}
			err := tf.client.Get(ctx, client.ObjectKeyFromObject(cert), existing)
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
				return tf.client.Update(ctx, existing)
			}
			return nil

		}
	}

	// Check secret created
	secretNamespacedName := types.NamespacedName{
		Name:      objName,
		Namespace: tf.Namespace,
	}
	err = retry.Do(
		func() error {
			secret := &corev1.Secret{}

			// Check secret exist in Kubernetes
			err := tf.client.Get(ctx, secretNamespacedName, secret)
			if err != nil {
				return err
			}
			return nil
		},
		retry.Attempts(5),
	)

	return err
}
