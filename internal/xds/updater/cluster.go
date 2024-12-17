package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertCluster(ctx context.Context, cl *v1alpha1.Cluster) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevCluster := c.store.Clusters[helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name}]
	if prevCluster == nil {
		c.store.Clusters[helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name}] = cl
		c.store.UpdateSpecClusters()
		return c.buildCache(ctx)
	}
	if prevCluster.IsEqual(cl) {
		return nil
	}
	c.store.Clusters[helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name}] = cl
	c.store.UpdateSpecClusters()
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeleteCluster(ctx context.Context, cl types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.Clusters[helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name}] == nil {
		return nil
	}
	delete(c.store.Clusters, helpers.NamespacedName{Namespace: cl.Namespace, Name: cl.Name})
	c.store.UpdateSpecClusters()
	return c.buildCache(ctx)
}

func (c *CacheUpdater) GetSpecCluster(specCluster string) *v1alpha1.Cluster {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.store.SpecClusters[specCluster]
}
