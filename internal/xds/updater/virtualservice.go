package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevVS := c.store.VirtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}]
	if prevVS == nil {
		c.store.VirtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}] = vs
		return c.buildCache(ctx)
	}
	if prevVS.IsEqual(vs) {
		return nil
	}
	c.store.VirtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}] = vs
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeleteVirtualService(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.VirtualServices[helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}] == nil {
		return nil
	}
	delete(c.store.VirtualServices, helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.buildCache(ctx)
}
