package updater

import (
	"context"
	"time"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplyVirtualServiceTemplate stores the template and rebuilds snapshots if changed.
// Returns true if the template was updated (existed before and spec changed),
// signaling that dependent VirtualServices should be re-reconciled.
// Returns false for new templates or unchanged templates.
func (c *CacheUpdater) ApplyVirtualServiceTemplate(
	ctx context.Context,
	vst *v1alpha1.VirtualServiceTemplate,
) (isUpdate bool) {
	lockStart := time.Now()
	c.mx.Lock()
	defer c.mx.Unlock()
	lockAcquireDuration := time.Since(lockStart)
	rlog := log.FromContext(ctx).WithName("cache-updater")
	if lockAcquireDuration > 100*time.Millisecond {
		rlog.Info("ApplyVST: lock contention detected",
			"template", vst.Name, "lockWaitDuration", lockAcquireDuration.String())
	}

	vst.NormalizeSpec()
	nn := helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}
	prevVST := c.store.GetVirtualServiceTemplate(nn)
	if prevVST == nil {
		c.store.SetVirtualServiceTemplate(vst)
		_ = c.rebuildSnapshots(ctx)
		return false
	}
	if prevVST.IsEqual(vst) {
		rlog.V(1).Info("Skipping unchanged VirtualServiceTemplate", "namespace", vst.Namespace, "name", vst.Name)
		return false
	}
	c.store.SetVirtualServiceTemplate(vst)
	_ = c.rebuildSnapshots(ctx)
	return true
}

func (c *CacheUpdater) DeleteVirtualServiceTemplate(ctx context.Context, nn types.NamespacedName) error {
	lockStart := time.Now()
	c.mx.Lock()
	defer c.mx.Unlock()
	lockAcquireDuration := time.Since(lockStart)
	if lockAcquireDuration > 100*time.Millisecond {
		rlog := log.FromContext(ctx).WithName("cache-updater")
		rlog.Info("DeleteVST: lock contention detected",
			"template", nn.String(), "lockWaitDuration", lockAcquireDuration.String())
	}

	if !c.store.IsExistingVirtualServiceTemplate(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteVirtualServiceTemplate(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshots(ctx)
}
