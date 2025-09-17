package http_filters

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filters"
	"google.golang.org/protobuf/types/known/anypb"
)

// Builder implements the HTTPFilterBuilder interface
type Builder struct {
	store *store.Store
}

// NewBuilder creates a new HTTP filter builder
func NewBuilder(store *store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// generateHTTPFiltersCacheKey creates a cache key for HTTP filters
func generateHTTPFiltersCacheKey(vs *v1alpha1.VirtualService, store *store.Store) string {
	hasher := sha256.New()

	// Include RBAC configuration if present
	if vs.Spec.RBAC != nil {
		if rbacData, err := json.Marshal(vs.Spec.RBAC); err == nil {
			hasher.Write(rbacData)
		}
	}

	// Include inline HTTP filters
	for _, filter := range vs.Spec.HTTPFilters {
		hasher.Write(filter.Raw)
	}

	// Include additional HTTP filter references and their content
	for _, filterRef := range vs.Spec.AdditionalHttpFilters {
		refNs := helpers.GetNamespace(filterRef.Namespace, vs.Namespace)
		hasher.Write([]byte(fmt.Sprintf("%s/%s", refNs, filterRef.Name)))

		// Include the actual filter content from store
		if hf := store.GetHTTPFilter(helpers.NamespacedName{Namespace: refNs, Name: filterRef.Name}); hf != nil {
			for _, spec := range hf.Spec {
				hasher.Write(spec.Raw)
			}
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// BuildHTTPFilters builds HTTP filters from a VirtualService
func (b *Builder) BuildHTTPFilters(vs *v1alpha1.VirtualService) ([]*hcmv3.HttpFilter, error) {
	// Check cache first
	cacheKey := generateHTTPFiltersCacheKey(vs, b.store)
	httpFilterCache := filters.GetGlobalHTTPFilterCache()
	if cached, exists := httpFilterCache.Get(cacheKey); exists {
		return cached, nil
	}

	httpFilters := make([]*hcmv3.HttpFilter, 0, len(vs.Spec.HTTPFilters)+len(vs.Spec.AdditionalHttpFilters))

	rbacF, err := b.BuildRBACFilter(vs)
	if err != nil {
		return nil, fmt.Errorf("failed to build RBAC filter: %w", err)
	}
	
	if rbacF != nil {
		configType := &hcmv3.HttpFilter_TypedConfig{
			TypedConfig: &anypb.Any{},
		}
		if err := configType.TypedConfig.MarshalFrom(rbacF); err != nil {
			return nil, fmt.Errorf("failed to marshal RBAC filter: %w", err)
		}
		httpFilters = append(httpFilters, &hcmv3.HttpFilter{
			Name:       "exc.filters.http.rbac",
			ConfigType: configType,
		})
	}

	// Process inline HTTP filters
	for _, httpFilter := range vs.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := protoutil.Unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, fmt.Errorf("failed to unmarshal HTTP filter: %w", err)
		}
		if err := hf.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate HTTP filter: %w", err)
		}
		httpFilters = append(httpFilters, hf)
	}

	// Process additional HTTP filters
	if len(vs.Spec.AdditionalHttpFilters) > 0 {
		for _, httpFilterRef := range vs.Spec.AdditionalHttpFilters {
			httpFilterRefNs := helpers.GetNamespace(httpFilterRef.Namespace, vs.Namespace)
			hf := b.store.GetHTTPFilter(helpers.NamespacedName{Namespace: httpFilterRefNs, Name: httpFilterRef.Name})
			if hf == nil {
				return nil, fmt.Errorf("HTTP filter %s/%s not found", httpFilterRefNs, httpFilterRef.Name)
			}
			for _, filter := range hf.Spec {
				xdsHttpFilter := &hcmv3.HttpFilter{}
				if err := protoutil.Unmarshaler.Unmarshal(filter.Raw, xdsHttpFilter); err != nil {
					return nil, fmt.Errorf("failed to unmarshal HTTP filter reference %s/%s: %w", 
						httpFilterRefNs, httpFilterRef.Name, err)
				}
				if err := xdsHttpFilter.ValidateAll(); err != nil {
					return nil, fmt.Errorf("failed to validate HTTP filter reference %s/%s: %w", 
						httpFilterRefNs, httpFilterRef.Name, err)
				}
				httpFilters = append(httpFilters, xdsHttpFilter)
			}
		}
	}

	// Router filter must be at the end
	var routerIdxs []int
	for i, f := range httpFilters {
		if tc := f.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router" {
				routerIdxs = append(routerIdxs, i)
			}
		}
	}

	switch {
	case len(routerIdxs) > 1:
		return nil, fmt.Errorf("multiple router HTTP filters found, only one is allowed")
	case len(routerIdxs) == 1 && routerIdxs[0] != len(httpFilters)-1:
		// Move router filter to the end
		index := routerIdxs[0]
		router := httpFilters[index]
		httpFilters = append(httpFilters[:index], httpFilters[index+1:]...)
		httpFilters = append(httpFilters, router)
	}

	// Store result in cache before returning
	httpFilterCache.Set(cacheKey, httpFilters)

	return httpFilters, nil
}

// BuildRBACFilter builds an RBAC filter from a VirtualService
func (b *Builder) BuildRBACFilter(vs *v1alpha1.VirtualService) (*rbacFilter.RBAC, error) {
	if vs.Spec.RBAC == nil {
		return nil, nil
	}

	if vs.Spec.RBAC.Action == "" {
		return nil, fmt.Errorf("RBAC action is empty")
	}

	action, ok := rbacv3.RBAC_Action_value[vs.Spec.RBAC.Action]
	if !ok {
		return nil, fmt.Errorf("invalid RBAC action %s", vs.Spec.RBAC.Action)
	}

	if len(vs.Spec.RBAC.Policies) == 0 && len(vs.Spec.RBAC.AdditionalPolicies) == 0 {
		return nil, fmt.Errorf("RBAC policies is empty")
	}

	rules := &rbacv3.RBAC{
		Action:   rbacv3.RBAC_Action(action),
		Policies: make(map[string]*rbacv3.Policy, len(vs.Spec.RBAC.Policies)),
	}

	// Process inline policies
	for policyName, rawPolicy := range vs.Spec.RBAC.Policies {
		policy := &rbacv3.Policy{}
		if err := protoutil.Unmarshaler.Unmarshal(rawPolicy.Raw, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal RBAC policy %s: %w", policyName, err)
		}
		if err := policy.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate RBAC policy %s: %w", policyName, err)
		}
		rules.Policies[policyName] = policy
	}

	// Process additional policies
	for _, policyRef := range vs.Spec.RBAC.AdditionalPolicies {
		ns := helpers.GetNamespace(policyRef.Namespace, vs.Namespace)
		policy := b.store.GetPolicy(helpers.NamespacedName{Namespace: ns, Name: policyRef.Name})
		if policy == nil {
			return nil, fmt.Errorf("RBAC policy %s/%s not found", ns, policyRef.Name)
		}
		if _, ok := rules.Policies[policy.Name]; ok {
			return nil, fmt.Errorf("policy '%s' already exists in RBAC", policy.Name)
		}
		rbacPolicy := &rbacv3.Policy{}
		if err := protoutil.Unmarshaler.Unmarshal(policy.Spec.Raw, rbacPolicy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal RBAC policy %s/%s: %w", ns, policyRef.Name, err)
		}
		if err := rbacPolicy.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate RBAC policy %s/%s: %w", ns, policyRef.Name, err)
		}
		rules.Policies[policy.Name] = rbacPolicy
	}

	return &rbacFilter.RBAC{Rules: rules}, nil
}