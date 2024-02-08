package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
)

func NodeIDs(obj client.Object) []string {
	annotation := NodeIDsAnnotation(obj)
	if annotation == "" {
		return nil
	}
	return strings.Split(annotation, ",")
}

func NodeIDsAnnotation(obj client.Object) string {
	annotations := obj.GetAnnotations()

	annotation, ok := annotations[options.NodeIDAnnotation]
	if !ok {
		return ""
	}

	return annotation
}

func NodeIDsContains(s1, s2 []string) bool {

	if len(s1) > len(s2) {
		return false
	}

	for _, e := range s1 {
		if !slices.Contains(s2, e) {
			return false
		}
	}

	return true
}

func ListSecrets(ctx context.Context, cl client.Client, listOpts client.ListOptions) ([]corev1.Secret, error) {
	secretList := corev1.SecretList{}
	err := cl.List(ctx, &secretList, &listOpts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list kubernetes secrets")
	}
	return secretList.Items, nil
}

// ResourceExists returns true if the given resource kind exists
// in the given api groupversion
func ResourceExists(dc discovery.DiscoveryInterface, apiGroupVersion, kind string) (bool, error) {
	apiList, err := dc.ServerResourcesForGroupVersion(apiGroupVersion)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, r := range apiList.APIResources {
		if r.Kind == kind {
			return true, nil
		}
	}

	return false, nil
}

// indexCertificateSecrets indexed all certificates for cache
func IndexCertificateSecrets(ctx context.Context, cl client.Client, namespace string) (map[string]corev1.Secret, error) {
	indexedSecrets := make(map[string]corev1.Secret)
	secrets, err := GetCertificateSecrets(ctx, cl, namespace)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		for _, domain := range strings.Split(secret.Annotations[options.DomainAnnotation], ",") {
			_, ok := indexedSecrets[domain]
			if ok {
				continue
			}
			indexedSecrets[domain] = secret
		}
	}
	return indexedSecrets, nil
}

// getCertificateSecrets gets all certificates from Kubernetes secrets
func GetCertificateSecrets(ctx context.Context, cl client.Client, namespace string) ([]corev1.Secret, error) {
	requirement, err := labels.NewRequirement(options.AutoDiscoveryLabel, "==", []string{"true"})
	if err != nil {
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	listOpt := client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     namespace,
	}
	return ListSecrets(ctx, cl, listOpt)
}

func ResourceName(namespace, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}
