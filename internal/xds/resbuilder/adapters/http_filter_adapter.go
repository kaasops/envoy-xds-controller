package adapters

import (
	"fmt"

	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/filters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/interfaces"
)

// HTTPFilterAdapter adapts the filters.Builder to implement the HTTPFilterBuilder interface
type HTTPFilterAdapter struct {
	builder *filters.Builder
	store   store.Store
}

// NewHTTPFilterAdapter creates a new adapter for the filters.Builder
func NewHTTPFilterAdapter(builder *filters.Builder, store store.Store) interfaces.HTTPFilterBuilder {
	return &HTTPFilterAdapter{
		builder: builder,
		store:   store,
	}
}

// BuildHTTPFilters delegates to the wrapped builder's BuildHTTPFilters method
func (a *HTTPFilterAdapter) BuildHTTPFilters(vs *v1alpha1.VirtualService) ([]*hcmv3.HttpFilter, error) {
	return a.builder.BuildHTTPFilters(vs)
}

// BuildRBACFilter implements the RBAC filter building logic based on the VirtualService configuration
func (a *HTTPFilterAdapter) BuildRBACFilter(vs *v1alpha1.VirtualService) (*rbacFilter.RBAC, error) {
	// If no RBAC config is specified, return nil
	if vs.Spec.RBAC == nil {
		return nil, nil
	}

	// Validate RBAC action is specified
	if vs.Spec.RBAC.Action == "" {
		return nil, fmt.Errorf("rbac action is empty")
	}

	// Validate RBAC action is valid
	action, ok := rbacv3.RBAC_Action_value[vs.Spec.RBAC.Action]
	if !ok {
		return nil, fmt.Errorf("invalid rbac action %s", vs.Spec.RBAC.Action)
	}

	// Ensure at least one policy is defined
	if len(vs.Spec.RBAC.Policies) == 0 && len(vs.Spec.RBAC.AdditionalPolicies) == 0 {
		return nil, fmt.Errorf("rbac policies is empty")
	}

	// Create RBAC rules with the specified action
	rules := &rbacv3.RBAC{
		Action:   rbacv3.RBAC_Action(action),
		Policies: make(map[string]*rbacv3.Policy, len(vs.Spec.RBAC.Policies)),
	}

	// Process inline policies
	for policyName, rawPolicy := range vs.Spec.RBAC.Policies {
		policy := &rbacv3.Policy{}
		if err := protoutil.Unmarshaler.Unmarshal(rawPolicy.Raw, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rbac policy %s: %w", policyName, err)
		}
		if err := policy.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate rbac policy %s: %w", policyName, err)
		}
		rules.Policies[policyName] = policy
	}

	// Process additional policies from references
	for _, policyRef := range vs.Spec.RBAC.AdditionalPolicies {
		ns := helpers.GetNamespace(policyRef.Namespace, vs.Namespace)
		policy := a.store.GetPolicy(helpers.NamespacedName{Namespace: ns, Name: policyRef.Name})
		if policy == nil {
			return nil, fmt.Errorf("rbac policy %s/%s not found", ns, policyRef.Name)
		}
		if _, ok := rules.Policies[policy.Name]; ok {
			return nil, fmt.Errorf("policy '%s' already exists in RBAC", policy.Name)
		}
		rbacPolicy := &rbacv3.Policy{}
		if err := protoutil.Unmarshaler.Unmarshal(policy.Spec.Raw, rbacPolicy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rbac policy %s/%s: %w", ns, policyRef.Name, err)
		}
		if err := rbacPolicy.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate rbac policy %s/%s: %w", ns, policyRef.Name, err)
		}
		rules.Policies[policy.Name] = rbacPolicy
	}

	// Return RBAC filter with constructed rules
	return &rbacFilter.RBAC{Rules: rules}, nil
}
