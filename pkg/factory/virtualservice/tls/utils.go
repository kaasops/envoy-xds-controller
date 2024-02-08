package tls

import (
	"context"
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/k8s-utils"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
