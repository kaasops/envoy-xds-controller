package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyTracing(ctx context.Context, tracing *v1alpha1.Tracing) {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevTracing := c.store.GetTracing(helpers.NamespacedName{Namespace: tracing.Namespace, Name: tracing.Name})
	if prevTracing == nil {
		c.store.SetTracing(tracing)
		_ = c.rebuildSnapshots(ctx)
		return
	}
	if prevTracing.IsEqual(tracing) {
		return
	}
	c.store.SetTracing(tracing)
	_ = c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeleteTracing(ctx context.Context, nn types.NamespacedName) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingTracing(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return
	}
	c.store.DeleteTracing(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	_ = c.rebuildSnapshots(ctx)
}
