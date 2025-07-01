package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplySecret(ctx context.Context, secret *v1.Secret) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevSecret := c.store.GetSecret(helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name})
	if prevSecret == nil {
		c.store.SetSecret(secret)
		return c.rebuildSnapshots(ctx)
	}
	if secretsEqual(prevSecret, secret) {
		return nil
	}
	c.store.SetSecret(secret)
	return c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeleteSecret(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingSecret(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteSecret(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshots(ctx)
}

func secretsEqual(a, b *v1.Secret) bool {
	if a.Data == nil && b.Data == nil {
		return true
	}
	if a.Data == nil || b.Data == nil {
		return false
	}
	if len(a.Data) != len(b.Data) {
		return false
	}
	for k, v := range a.Data {
		if b.Data[k] == nil {
			return false
		}
		if string(v) != string(b.Data[k]) {
			return false
		}
	}
	valA, okA := a.Annotations[v1alpha1.AnnotationSecretDomains]
	valB, okB := b.Annotations[v1alpha1.AnnotationSecretDomains]
	if okA != okB || valA != valB {
		return false
	}
	return true
}
