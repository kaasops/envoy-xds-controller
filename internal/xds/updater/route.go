package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyRoute(ctx context.Context, route *v1alpha1.Route) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevRoute := c.store.GetRoute(helpers.NamespacedName{Namespace: route.Namespace, Name: route.Name})
	if prevRoute == nil {
		c.store.SetRoute(route)
		return c.rebuildSnapshot(ctx)
	}
	if prevRoute.IsEqual(route) {
		return nil
	}
	c.store.SetRoute(route)
	return c.rebuildSnapshot(ctx)
}

func (c *CacheUpdater) DeleteRoute(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingRoute(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteRoute(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshot(ctx)
}
