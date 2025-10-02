package updater

import (
	"context"
	"time"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (c *CacheUpdater) ApplyVirtualServiceTemplate(ctx context.Context, vst *v1alpha1.VirtualServiceTemplate) (isUpdate bool) {
	lockStart := time.Now()
	c.mx.Lock()
	defer c.mx.Unlock()
	lockAcquireDuration := time.Since(lockStart)
	if lockAcquireDuration > 100*time.Millisecond {
		rlog := log.FromContext(ctx).WithName("cache-updater")
		rlog.Info("ApplyVST: lock contention detected", "template", vst.Name, "lockWaitDuration", lockAcquireDuration.String())
	}

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
	lockStart := time.Now()
	c.mx.Lock()
	defer c.mx.Unlock()
	lockAcquireDuration := time.Since(lockStart)
	if lockAcquireDuration > 100*time.Millisecond {
		rlog := log.FromContext(ctx).WithName("cache-updater")
		rlog.Info("DeleteVST: lock contention detected", "template", nn.String(), "lockWaitDuration", lockAcquireDuration.String())
	}

	if !c.store.IsExistingVirtualServiceTemplate(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return nil
	}
	c.store.DeleteVirtualServiceTemplate(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.rebuildSnapshots(ctx)
}
