package v1alpha1

import (
	"bytes"

	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
)

func (p *Policy) UnmarshalV3() (*rbacv3.Policy, error) {
	return p.unmarshalV3()
}

func (p *Policy) unmarshalV3() (*rbacv3.Policy, error) {
	if p.Spec == nil {
		return nil, ErrSpecNil
	}
	var policy rbacv3.Policy
	if err := protoutil.Unmarshaler.Unmarshal(p.Spec.Raw, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (p *Policy) UnmarshalV3AndValidate() (*rbacv3.Policy, error) {
	policy, err := p.unmarshalV3()
	if err != nil {
		return nil, err
	}
	if err := policy.ValidateAll(); err != nil {
		return nil, err
	}
	return policy, nil
}

func (p *Policy) IsEqual(other *Policy) bool {
	if p == nil && other == nil {
		return true
	}
	if p == nil || other == nil {
		return false
	}
	if p.Spec == nil && other.Spec == nil {
		return true
	}
	if p.Spec == nil || other.Spec == nil {
		return false
	}
	if p.Spec.Raw == nil && other.Spec.Raw == nil {
		return true
	}
	if p.Spec.Raw == nil || other.Spec.Raw == nil {
		return false
	}
	return bytes.Equal(p.Spec.Raw, other.Spec.Raw)
}

func (p *Policy) GetAccessGroup() string {
	accessGroup := p.GetLabels()[LabelAccessGroup]
	if accessGroup == "" {
		return GeneralAccessGroup
	}
	return accessGroup
}

func (p *Policy) GetDescription() string {
	return p.Annotations[annotationDescription]
}
