package filters

import (
	"fmt"

	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"k8s.io/apimachinery/pkg/runtime"
)

// BuildRBACFilter builds an RBAC filter from VirtualService RBAC configuration
func BuildRBACFilter(vs *v1alpha1.VirtualService, store *store.Store) (*rbacFilter.RBAC, error) {
	if vs.Spec.RBAC == nil {
		return nil, nil
	}

	// Validate RBAC action
	if vs.Spec.RBAC.Action == "" {
		return nil, fmt.Errorf("RBAC action is empty")
	}

	action, ok := rbacv3.RBAC_Action_value[vs.Spec.RBAC.Action]
	if !ok {
		return nil, fmt.Errorf("invalid RBAC action %s", vs.Spec.RBAC.Action)
	}

	// Validate that at least one policy is specified
	if len(vs.Spec.RBAC.Policies) == 0 && len(vs.Spec.RBAC.AdditionalPolicies) == 0 {
		return nil, fmt.Errorf("RBAC policies is empty")
	}

	// Initialize RBAC rules with estimated capacity
	totalPolicies := len(vs.Spec.RBAC.Policies) + len(vs.Spec.RBAC.AdditionalPolicies)
	rules := &rbacv3.RBAC{
		Action:   rbacv3.RBAC_Action(action),
		Policies: make(map[string]*rbacv3.Policy, totalPolicies),
	}

	// Process inline policies
	for policyName, rawPolicy := range vs.Spec.RBAC.Policies {
		policy, err := buildInlineRBACPolicy(policyName, rawPolicy)
		if err != nil {
			return nil, err
		}
		rules.Policies[policyName] = policy
	}

	// Process referenced policies
	for _, policyRef := range vs.Spec.RBAC.AdditionalPolicies {
		policy, err := buildReferencedRBACPolicy(policyRef, vs.Namespace, store)
		if err != nil {
			return nil, err
		}

		// Check for policy name conflicts
		if _, exists := rules.Policies[policy.name]; exists {
			return nil, fmt.Errorf("policy '%s' already exists in RBAC", policy.name)
		}

		rules.Policies[policy.name] = policy.policy
	}

	return &rbacFilter.RBAC{Rules: rules}, nil
}

// buildInlineRBACPolicy builds an RBAC policy from inline configuration
func buildInlineRBACPolicy(policyName string, rawPolicy *runtime.RawExtension) (*rbacv3.Policy, error) {
	policy := &rbacv3.Policy{}
	if err := protoutil.Unmarshaler.Unmarshal(rawPolicy.Raw, policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC policy %s: %w", policyName, err)
	}
	if err := policy.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate RBAC policy %s: %w", policyName, err)
	}
	return policy, nil
}

// referencedRBACPolicy holds a referenced policy with its name
type referencedRBACPolicy struct {
	name   string
	policy *rbacv3.Policy
}

// buildReferencedRBACPolicy builds an RBAC policy from a reference
func buildReferencedRBACPolicy(policyRef *v1alpha1.ResourceRef, vsNamespace string, store *store.Store) (*referencedRBACPolicy, error) {
	ns := helpers.GetNamespace(policyRef.Namespace, vsNamespace)
	policy := store.GetPolicy(helpers.NamespacedName{Namespace: ns, Name: policyRef.Name})
	if policy == nil {
		return nil, fmt.Errorf("RBAC policy %s/%s not found", ns, policyRef.Name)
	}

	rbacPolicy := &rbacv3.Policy{}
	if err := protoutil.Unmarshaler.Unmarshal(policy.Spec.Raw, rbacPolicy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC policy %s/%s: %w", ns, policyRef.Name, err)
	}
	if err := rbacPolicy.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate RBAC policy %s/%s: %w", ns, policyRef.Name, err)
	}

	return &referencedRBACPolicy{
		name:   policy.Name,
		policy: rbacPolicy,
	}, nil
}
