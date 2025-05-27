package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevVS := c.store.GetVirtualService(helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name})
	if prevVS == nil {
		c.store.SetVirtualService(vs)
		return c.rebuildSnapshot(ctx)
	}
	if prevVS.IsEqual(vs) {
		return nil
	}
	c.store.SetVirtualService(vs)
	return c.rebuildSnapshot(ctx)
}

func (c *CacheUpdater) DeleteVirtualService(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingVirtualService(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteVirtualService(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshot(ctx)
}
