package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyVirtualServiceTemplate(ctx context.Context, vst *v1alpha1.VirtualServiceTemplate) (isUpdate bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevVST := c.store.GetVirtualServiceTemplate(helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name})
	if prevVST == nil {
		c.store.SetVirtualServiceTemplate(vst)
		_ = c.rebuildSnapshots(ctx)
		return false
	}
	if prevVST.IsEqual(vst) {
		return false
	}
	c.store.SetVirtualServiceTemplate(vst)
	_ = c.rebuildSnapshots(ctx)
	return true
}

func (c *CacheUpdater) DeleteVirtualServiceTemplate(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingVirtualServiceTemplate(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteVirtualServiceTemplate(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshots(ctx)
}
