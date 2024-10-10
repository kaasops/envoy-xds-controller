package v1alpha1

import (
	"context"
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
)

func (p *Policy) Validate(_ context.Context) error {
	if p.Spec == nil {
		return errors.New(errors.PolicyCannotBeEmptyMessage)
	}
	policy := &rbacv3.Policy{}
	if err := options.Unmarshaler.Unmarshal(p.Spec.Raw, policy); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	return policy.ValidateAll()
}
