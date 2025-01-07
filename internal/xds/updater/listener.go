package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertListener(ctx context.Context, listener *v1alpha1.Listener) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevListener := c.store.Listeners[helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}]
	if prevListener == nil {
		c.store.Listeners[helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}] = listener
		return c.buildCache(ctx)
	}
	if prevListener.IsEqual(listener) {
		return nil
	}
	c.store.Listeners[helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}] = listener
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeleteListener(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.Listeners[helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}] == nil {
		return nil
	}
	delete(c.store.Listeners, helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.buildCache(ctx)
}
