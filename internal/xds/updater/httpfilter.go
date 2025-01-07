package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertHTTPFilter(ctx context.Context, httpFilter *v1alpha1.HttpFilter) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevHTTPFilter := c.store.HTTPFilters[helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name}]
	if prevHTTPFilter == nil {
		c.store.HTTPFilters[helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name}] = httpFilter
		return c.buildCache(ctx)
	}
	if prevHTTPFilter.IsEqual(httpFilter) {
		return nil
	}
	c.store.HTTPFilters[helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name}] = httpFilter
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeleteHTTPFilter(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.HTTPFilters[helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}] == nil {
		return nil
	}
	delete(c.store.HTTPFilters, helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.buildCache(ctx)
}
