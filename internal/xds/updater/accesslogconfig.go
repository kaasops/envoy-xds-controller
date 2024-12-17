package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertAccessLogConfig(ctx context.Context, alc *v1alpha1.AccessLogConfig) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevALC := c.store.AccessLogs[helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name}]
	if prevALC == nil {
		c.store.AccessLogs[helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name}] = alc
		return c.buildCache(ctx)
	}
	if prevALC.IsEqual(alc) {
		return nil
	}
	c.store.AccessLogs[helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name}] = alc
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeleteAccessLogConfig(ctx context.Context, alc types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.AccessLogs[helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name}] == nil {
		return nil
	}
	delete(c.store.AccessLogs, helpers.NamespacedName{Namespace: alc.Namespace, Name: alc.Name})
	return c.buildCache(ctx)
}
