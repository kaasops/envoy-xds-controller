package updater

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

func (c *CacheUpdater) UpsertPolicy(ctx context.Context, policy *v1alpha1.Policy) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	prevPolicy := c.store.Policies[helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}]
	if prevPolicy == nil {
		c.store.Policies[helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = policy
		return c.buildCache(ctx)
	}
	if prevPolicy.IsEqual(policy) {
		return nil
	}
	c.store.Policies[helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = policy
	return c.buildCache(ctx)
}

func (c *CacheUpdater) DeletePolicy(ctx context.Context, nn types.NamespacedName) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.store.Policies[helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name}] == nil {
		return nil
	}
	delete(c.store.Policies, helpers.NamespacedName{Namespace: nn.Namespace, Name: nn.Name})
	return c.buildCache(ctx)
}
