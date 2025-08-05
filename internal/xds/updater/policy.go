package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) ApplyPolicy(ctx context.Context, policy *v1alpha1.Policy) {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevPolicy := c.store.GetPolicy(helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name})
	if prevPolicy == nil {
		c.store.SetPolicy(policy)
		_ = c.rebuildSnapshots(ctx)
		return
	}
	if prevPolicy.IsEqual(policy) {
		return
	}
	c.store.SetPolicy(policy)
	_ = c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) DeletePolicy(ctx context.Context, nn types.NamespacedName) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if !c.store.IsExistingPolicy(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}) {
		return
	}
	c.store.DeletePolicy(helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	_ = c.rebuildSnapshots(ctx)
}
