package filters

import (
	"crypto/sha256"
	"fmt"
	"sync"

	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"google.golang.org/protobuf/types/known/anypb"
)

// HTTPFilterBuilder handles the construction of HTTP filters with caching
type HTTPFilterBuilder struct {
	cache *filterCache
}

// NewHTTPFilterBuilder creates a new HTTP filter builder with caching enabled
func NewHTTPFilterBuilder() *HTTPFilterBuilder {
	return &HTTPFilterBuilder{
		cache: newFilterCache(),
	}
}

// filterCache provides thread-safe caching of HTTP filter build results
type filterCache struct {
	mu      sync.RWMutex
	cache   map[string][]*hcmv3.HttpFilter
	maxSize int
}

// newFilterCache creates a new HTTP filters cache with default settings
func newFilterCache() *filterCache {
	return &filterCache{
		cache:   make(map[string][]*hcmv3.HttpFilter),
		maxSize: 1000, // Limit cache size
	}
}

// get retrieves cached HTTP filters for the given key
func (c *filterCache) get(key string) ([]*hcmv3.HttpFilter, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	filters, exists := c.cache[key]
	if !exists {
		return nil, false
	}
	
	// Return deep copies to avoid mutation issues
	result := make([]*hcmv3.HttpFilter, len(filters))
	for i, filter := range filters {
		result[i] = &hcmv3.HttpFilter{}
		*result[i] = *filter // Shallow copy should be sufficient for protobuf messages
	}
	
	return result, true
}

// set stores HTTP filters in the cache for the given key
func (c *filterCache) set(key string, filters []*hcmv3.HttpFilter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Simple eviction: if cache is full, clear it
	if len(c.cache) >= c.maxSize {
		c.cache = make(map[string][]*hcmv3.HttpFilter)
	}
	
	// Store deep copies to avoid mutation issues
	cached := make([]*hcmv3.HttpFilter, len(filters))
	for i, filter := range filters {
		cached[i] = &hcmv3.HttpFilter{}
		*cached[i] = *filter // Shallow copy should be sufficient for protobuf messages
	}
	
	c.cache[key] = cached
}

// BuildHTTPFilters builds HTTP filters for a VirtualService with caching
func (b *HTTPFilterBuilder) BuildHTTPFilters(vs *v1alpha1.VirtualService, store *store.Store) ([]*hcmv3.HttpFilter, error) {
	// Check cache first
	cacheKey := b.generateCacheKey(vs, store)
	if cached, exists := b.cache.get(cacheKey); exists {
		return cached, nil
	}
	
	// Estimate capacity for pre-allocation
	estimatedCapacity := len(vs.Spec.HTTPFilters) + len(vs.Spec.AdditionalHttpFilters) + 1 // +1 for potential RBAC
	httpFilters := make([]*hcmv3.HttpFilter, 0, estimatedCapacity)

	// 1. Build RBAC filter if configured
	rbacFilter, err := BuildRBACFilter(vs, store)
	if err != nil {
		return nil, fmt.Errorf("failed to build RBAC filter: %w", err)
	}
	if rbacFilter != nil {
		configType := &hcmv3.HttpFilter_TypedConfig{
			TypedConfig: &anypb.Any{},
		}
		if err := configType.TypedConfig.MarshalFrom(rbacFilter); err != nil {
			return nil, fmt.Errorf("failed to marshal RBAC filter: %w", err)
		}
		httpFilters = append(httpFilters, &hcmv3.HttpFilter{
			Name:       "exc.filters.http.rbac",
			ConfigType: configType,
		})
	}

	// 2. Process inline HTTP filters
	for _, httpFilter := range vs.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := protoutil.Unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, fmt.Errorf("failed to unmarshal inline HTTP filter: %w", err)
		}
		if err := hf.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate inline HTTP filter: %w", err)
		}
		httpFilters = append(httpFilters, hf)
	}

	// 3. Process referenced HTTP filters
	if len(vs.Spec.AdditionalHttpFilters) > 0 {
		for _, httpFilterRef := range vs.Spec.AdditionalHttpFilters {
			refFilters, err := b.buildReferencedHTTPFilters(httpFilterRef, vs.Namespace, store)
			if err != nil {
				return nil, fmt.Errorf("failed to build referenced HTTP filter %s/%s: %w", 
					helpers.GetNamespace(httpFilterRef.Namespace, vs.Namespace), httpFilterRef.Name, err)
			}
			httpFilters = append(httpFilters, refFilters...)
		}
	}

	// 4. Ensure router filter is at the end
	if err := b.ensureRouterFilterAtEnd(httpFilters); err != nil {
		return nil, err
	}

	// Store result in cache before returning
	b.cache.set(cacheKey, httpFilters)
	
	return httpFilters, nil
}

// buildReferencedHTTPFilters builds HTTP filters from a reference
func (b *HTTPFilterBuilder) buildReferencedHTTPFilters(httpFilterRef *v1alpha1.ResourceRef, vsNamespace string, store *store.Store) ([]*hcmv3.HttpFilter, error) {
	httpFilterRefNs := helpers.GetNamespace(httpFilterRef.Namespace, vsNamespace)
	hf := store.GetHTTPFilter(helpers.NamespacedName{Namespace: httpFilterRefNs, Name: httpFilterRef.Name})
	if hf == nil {
		return nil, fmt.Errorf("HTTP filter %s/%s not found", httpFilterRefNs, httpFilterRef.Name)
	}
	
	filters := make([]*hcmv3.HttpFilter, 0, len(hf.Spec))
	for idx, filter := range hf.Spec {
		xdsHttpFilter := &hcmv3.HttpFilter{}
		if err := protoutil.Unmarshaler.Unmarshal(filter.Raw, xdsHttpFilter); err != nil {
			return nil, fmt.Errorf("failed to unmarshal HTTP filter %s/%s[%d]: %w", httpFilterRefNs, httpFilterRef.Name, idx, err)
		}
		if err := xdsHttpFilter.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate HTTP filter %s/%s[%d]: %w", httpFilterRefNs, httpFilterRef.Name, idx, err)
		}
		filters = append(filters, xdsHttpFilter)
	}
	
	return filters, nil
}

// ensureRouterFilterAtEnd ensures that the router filter is positioned at the end
func (b *HTTPFilterBuilder) ensureRouterFilterAtEnd(httpFilters []*hcmv3.HttpFilter) error {
	const routerTypeURL = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	
	var routerIdxs []int
	for i, f := range httpFilters {
		if tc := f.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == routerTypeURL {
				routerIdxs = append(routerIdxs, i)
			}
		}
	}

	switch {
	case len(routerIdxs) > 1:
		return fmt.Errorf("multiple router HTTP filters found")
	case len(routerIdxs) == 1 && routerIdxs[0] != len(httpFilters)-1:
		// Move router filter to the end
		index := routerIdxs[0]
		routerFilter := httpFilters[index]
		copy(httpFilters[index:], httpFilters[index+1:])
		httpFilters[len(httpFilters)-1] = routerFilter
	}

	return nil
}

// generateCacheKey creates a hash-based cache key for HTTP filters configuration
func (b *HTTPFilterBuilder) generateCacheKey(vs *v1alpha1.VirtualService, store *store.Store) string {
	hasher := sha256.New()
	
	// Include VirtualService namespace and name for uniqueness
	hasher.Write([]byte(fmt.Sprintf("%s/%s", vs.Namespace, vs.Name)))
	
	// Include RBAC configuration if present
	if vs.Spec.RBAC != nil {
		hasher.Write([]byte("rbac:"))
		hasher.Write([]byte(vs.Spec.RBAC.Action))
		
		// Include inline policies
		for policyName, policy := range vs.Spec.RBAC.Policies {
			hasher.Write([]byte(fmt.Sprintf("policy:%s:", policyName)))
			hasher.Write(policy.Raw)
		}
		
		// Include referenced policies
		for _, policyRef := range vs.Spec.RBAC.AdditionalPolicies {
			refNs := helpers.GetNamespace(policyRef.Namespace, vs.Namespace)
			hasher.Write([]byte(fmt.Sprintf("policyRef:%s/%s", refNs, policyRef.Name)))
			
			// Include actual policy content from store
			if policy := store.GetPolicy(helpers.NamespacedName{Namespace: refNs, Name: policyRef.Name}); policy != nil {
				hasher.Write(policy.Spec.Raw)
			}
		}
	}
	
	// Include inline HTTP filters
	for i, filter := range vs.Spec.HTTPFilters {
		hasher.Write([]byte(fmt.Sprintf("inline:%d:", i)))
		hasher.Write(filter.Raw)
	}
	
	// Include additional HTTP filter references and their content
	for _, filterRef := range vs.Spec.AdditionalHttpFilters {
		refNs := helpers.GetNamespace(filterRef.Namespace, vs.Namespace)
		hasher.Write([]byte(fmt.Sprintf("ref:%s/%s:", refNs, filterRef.Name)))
		
		// Include the actual filter content from store
		if hf := store.GetHTTPFilter(helpers.NamespacedName{Namespace: refNs, Name: filterRef.Name}); hf != nil {
			for i, spec := range hf.Spec {
				hasher.Write([]byte(fmt.Sprintf("spec:%d:", i)))
				hasher.Write(spec.Raw)
			}
		}
	}
	
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Clear clears all cached entries (useful for testing or memory management)
func (b *HTTPFilterBuilder) Clear() {
	b.cache.mu.Lock()
	defer b.cache.mu.Unlock()
	
	b.cache.cache = make(map[string][]*hcmv3.HttpFilter)
}

// Size returns the current cache size (useful for monitoring)
func (b *HTTPFilterBuilder) Size() int {
	b.cache.mu.RLock()
	defer b.cache.mu.RUnlock()
	
	return len(b.cache.cache)
}