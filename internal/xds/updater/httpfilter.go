package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyHTTPFilter(ctx context.Context, httpFilter *v1alpha1.HttpFilter) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevHTTPFilter := c.store.GetHTTPFilter(helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name})
	if prevHTTPFilter == nil {
		c.store.SetHTTPFilter(httpFilter)
		return c.rebuildSnapshot(ctx)
	}
	if prevHTTPFilter.IsEqual(httpFilter) {
		return nil
	}
	c.store.SetHTTPFilter(httpFilter)
	return c.rebuildSnapshot(ctx)
}

func (c *CacheUpdater) DeleteHTTPFilter(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingHTTPFilter(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteHTTPFilter(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshot(ctx)
}
