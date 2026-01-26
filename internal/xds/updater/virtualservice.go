package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplyVirtualService stores the VirtualService and rebuilds snapshots if changed.
// If the VirtualService already exists and spec is unchanged, the operation is skipped
// to avoid unnecessary snapshot rebuilds.
func (c *CacheUpdater) ApplyVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) {
	c.mx.Lock()
	defer c.mx.Unlock()
	vs.NormalizeSpec()
	nn := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}
	prevVS := c.store.GetVirtualService(nn)
	if prevVS == nil {
		c.store.SetVirtualService(vs)
		_ = c.rebuildSnapshots(ctx)
		return
	}
	if prevVS.IsEqual(vs) {
		rlog := log.FromContext(ctx).WithName("cache-updater")
		rlog.V(1).Info("Skipping unchanged VirtualService", "namespace", vs.Namespace, "name", vs.Name)
		return
	}
	c.store.SetVirtualService(vs)
	_ = c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeleteVirtualService(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingVirtualService(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteVirtualService(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) GetVirtualServicesByTemplate(vst *v1alpha1.VirtualServiceTemplate) []*v1alpha1.VirtualService {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.store.GetVirtualServicesByTemplateNN(helpers.NamespacedName{Name: vst.Name, Namespace: vst.Namespace})
}
