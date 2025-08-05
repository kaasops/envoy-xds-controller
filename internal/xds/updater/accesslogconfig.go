package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyAccessLogConfig(ctx context.Context, alc *v1alpha1.AccessLogConfig) {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevALC := c.store.GetAccessLog(helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name})
	if prevALC == nil {
		c.store.SetAccessLog(alc)
		_ = c.rebuildSnapshots(ctx)
		return
	}
	if prevALC.IsEqual(alc) {
		return
	}
	c.store.SetAccessLog(alc)
	_ = c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeleteAccessLogConfig(ctx context.Context, alc types.NamespacedName) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingAccessLog(helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name}) {
		return
	}
	c.store.DeleteAccessLog(helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name})
	_ = c.rebuildSnapshots(ctx)
}
