package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyAccessLogConfig(ctx context.Context, alc *v1alpha1.AccessLogConfig) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevALC := c.store.GetAccessLog(helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name})
	if prevALC == nil {
		c.store.SetAccessLog(alc)
		return c.rebuildSnapshot(ctx)
	}
	if prevALC.IsEqual(alc) {
		return nil
	}
	c.store.SetAccessLog(alc)
	return c.rebuildSnapshot(ctx)
}

func (c *CacheUpdater) DeleteAccessLogConfig(ctx context.Context, alc types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingAccessLog(helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name}) {
		return nil
	}
	c.store.DeleteAccessLog(helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name})
	return c.rebuildSnapshot(ctx)
}
