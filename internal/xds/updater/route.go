package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyRoute(ctx context.Context, route *v1alpha1.Route) {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevRoute := c.store.GetRoute(helpers.NamespacedName{Namespace: route.Namespace, Name: route.Name})
	if prevRoute == nil {
		c.store.SetRoute(route)
		_ = c.rebuildSnapshots(ctx)
		return
	}
	if prevRoute.IsEqual(route) {
		return
	}
	c.store.SetRoute(route)
	_ = c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeleteRoute(ctx context.Context, nn types.NamespacedName) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingRoute(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return
	}
	c.store.DeleteRoute(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	_ = c.rebuildSnapshots(ctx)
}
