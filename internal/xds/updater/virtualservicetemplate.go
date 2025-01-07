package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertVirtualServiceTemplate(ctx context.Context, vst *v1alpha1.VirtualServiceTemplate) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevVST := c.store.VirtualServiceTemplates[helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}]
	if prevVST == nil {
		c.store.VirtualServiceTemplates[helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}] = vst
		return c.buildCache(ctx)
	}
	if prevVST.IsEqual(vst) {
		return nil
	}
	c.store.VirtualServiceTemplates[helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}] = vst
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeleteVirtualServiceTemplate(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.VirtualServiceTemplates[helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}] == nil {
		return nil
	}
	delete(c.store.VirtualServiceTemplates, helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.buildCache(ctx)
}
