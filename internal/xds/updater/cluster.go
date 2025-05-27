package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyCluster(ctx context.Context, cl *v1alpha1.Cluster) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevCluster := c.store.GetCluster(helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name})
	if prevCluster == nil {
		c.store.SetCluster(cl)
		return c.rebuildSnapshot(ctx)
	}
	if prevCluster.IsEqual(cl) {
		return nil
	}
	c.store.SetCluster(cl)
	return c.rebuildSnapshot(ctx)
}

func (c *CacheUpdater) DeleteCluster(ctx context.Context, cl types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingCluster(helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name}) {
		return nil
	}
	c.store.DeleteCluster(helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name})
	return c.rebuildSnapshot(ctx)
}

func (c *CacheUpdater) GetSpecCluster(specCluster string) *v1alpha1.Cluster {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.store.GetSpecCluster(specCluster)
}
