package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyListener(ctx context.Context, listener *v1alpha1.Listener) {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevListener := c.store.GetListener(helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name})
	if prevListener == nil {
		c.store.SetListener(listener)
		_ = c.rebuildSnapshots(ctx)
		return
	}
	if prevListener.IsEqual(listener) {
		return
	}
	c.store.SetListener(listener)
	_ = c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeleteListener(ctx context.Context, nn types.NamespacedName) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingListener(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return
	}
	c.store.DeleteListener(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	_ = c.rebuildSnapshots(ctx)
}
