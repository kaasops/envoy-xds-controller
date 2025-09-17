package filters

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/apimachinery/pkg/runtime"
)

// Cache for buildHTTPFilters results to avoid expensive re-computation
type httpFiltersCache struct {
	mu      sync.RWMutex
	cache   map[string][]*hcmv3.HttpFilter
	maxSize int
}

func newHTTPFiltersCache() *httpFiltersCache {
	return &httpFiltersCache{
		cache:   make(map[string][]*hcmv3.HttpFilter),
		maxSize: 1000, // Limit cache size
	}
}

func (c *httpFiltersCache) get(key string) ([]*hcmv3.HttpFilter, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	filters, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Get a slice from the pool
	resultPtr := utils.GetHTTPFilterSlice()

	// Create deep copies to avoid mutation issues
	for _, filter := range filters {
		*resultPtr = append(*resultPtr, proto.Clone(filter).(*hcmv3.HttpFilter))
	}

	// Create a new slice to return to the caller - we can't return the pooled slice directly
	finalResult := make([]*hcmv3.HttpFilter, len(*resultPtr))
	copy(finalResult, *resultPtr)

	// Return the slice to the pool
	utils.PutHTTPFilterSlice(resultPtr)

	return finalResult, true
}

func (c *httpFiltersCache) set(key string, filters []*hcmv3.HttpFilter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: if cache is full, clear it
	if len(c.cache) >= c.maxSize {
		c.cache = make(map[string][]*hcmv3.HttpFilter)
	}

	// Get a slice from the pool
	cachedPtr := utils.GetHTTPFilterSlice()

	// Store deep copies to avoid mutation issues
	for _, filter := range filters {
		*cachedPtr = append(*cachedPtr, proto.Clone(filter).(*hcmv3.HttpFilter))
	}

	// Create a permanent slice for the cache - we can't store the pooled slice
	permanentCached := make([]*hcmv3.HttpFilter, len(*cachedPtr))
	copy(permanentCached, *cachedPtr)
	c.cache[key] = permanentCached

	// Return the slice to the pool
	utils.PutHTTPFilterSlice(cachedPtr)
}

var globalHTTPFiltersCache = newHTTPFiltersCache()

// Builder handles filter building operations
type Builder struct {
	store *store.Store
	cache *httpFiltersCache
}

// NewBuilder creates a new filter builder
func NewBuilder(store *store.Store) *Builder {
	return &Builder{
		store: store,
		cache: globalHTTPFiltersCache,
	}
}

// generateHTTPFiltersCacheKey creates a hash-based cache key for HTTP filters configuration
func (b *Builder) generateHTTPFiltersCacheKey(vs *v1alpha1.VirtualService) string {
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
		if hf := b.store.GetHTTPFilter(helpers.NamespacedName{Namespace: refNs, Name: filterRef.Name}); hf != nil {
			for _, spec := range hf.Spec {
				hasher.Write(spec.Raw)
			}
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// BuildHTTPFilters builds HTTP filters for a VirtualService
func (b *Builder) BuildHTTPFilters(vs *v1alpha1.VirtualService) ([]*hcmv3.HttpFilter, error) {
	// Check cache first
	cacheKey := b.generateHTTPFiltersCacheKey(vs)
	if cached, exists := b.cache.get(cacheKey); exists {
		return cached, nil
	}

	httpFilters := make([]*hcmv3.HttpFilter, 0, len(vs.Spec.HTTPFilters)+len(vs.Spec.AdditionalHttpFilters))

	rbacF, err := b.buildRBACFilter(vs)
	if err != nil {
		return nil, err
	}
	if rbacF != nil {
		configType := &hcmv3.HttpFilter_TypedConfig{
			TypedConfig: &anypb.Any{},
		}
		if err := configType.TypedConfig.MarshalFrom(rbacF); err != nil {
			return nil, err
		}
		httpFilters = append(httpFilters, &hcmv3.HttpFilter{
			Name:       "exc.filters.http.rbac",
			ConfigType: configType,
		})
	}

	for _, httpFilter := range vs.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := protoutil.Unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, fmt.Errorf("failed to unmarshal http filter: %w", err)
		}
		if err := hf.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate http filter: %w", err)
		}
		httpFilters = append(httpFilters, hf)
	}

	if len(vs.Spec.AdditionalHttpFilters) > 0 {
		for _, httpFilterRef := range vs.Spec.AdditionalHttpFilters {
			httpFilterRefNs := helpers.GetNamespace(httpFilterRef.Namespace, vs.Namespace)
			hf := b.store.GetHTTPFilter(helpers.NamespacedName{Namespace: httpFilterRefNs, Name: httpFilterRef.Name})
			if hf == nil {
				return nil, fmt.Errorf("http filter %s/%s not found", httpFilterRefNs, httpFilterRef.Name)
			}
			for _, filter := range hf.Spec {
				xdsHttpFilter := &hcmv3.HttpFilter{}
				if err := protoutil.Unmarshaler.Unmarshal(filter.Raw, xdsHttpFilter); err != nil {
					return nil, err
				}
				if err := xdsHttpFilter.ValidateAll(); err != nil {
					return nil, err
				}
				httpFilters = append(httpFilters, xdsHttpFilter)
			}
		}
	}

	// filter with type type.googleapis.com/envoy.extensions.filters.http.router.v3.Router must be in the end
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
		return nil, fmt.Errorf("multiple root router http filters")
	case len(routerIdxs) == 1 && routerIdxs[0] != len(httpFilters)-1:
		index := routerIdxs[0]
		route := httpFilters[index]
		httpFilters = append(httpFilters[:index], httpFilters[index+1:]...)
		httpFilters = append(httpFilters, route)
	}

	// Store result in cache before returning
	b.cache.set(cacheKey, httpFilters)

	return httpFilters, nil
}

// buildRBACFilter builds RBAC filter if RBAC is configured
func (b *Builder) buildRBACFilter(vs *v1alpha1.VirtualService) (*rbacFilter.RBAC, error) {
	if vs.Spec.RBAC == nil {
		return nil, nil
	}

	if vs.Spec.RBAC.Action == "" {
		return nil, fmt.Errorf("rbac action is empty")
	}

	action, ok := rbacv3.RBAC_Action_value[vs.Spec.RBAC.Action]
	if !ok {
		return nil, fmt.Errorf("invalid rbac action %s", vs.Spec.RBAC.Action)
	}

	if len(vs.Spec.RBAC.Policies) == 0 && len(vs.Spec.RBAC.AdditionalPolicies) == 0 {
		return nil, fmt.Errorf("rbac policies is empty")
	}

	rules := &rbacv3.RBAC{Action: rbacv3.RBAC_Action(action), Policies: make(map[string]*rbacv3.Policy, len(vs.Spec.RBAC.Policies))}
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

	for _, policyRef := range vs.Spec.RBAC.AdditionalPolicies {
		ns := helpers.GetNamespace(policyRef.Namespace, vs.Namespace)
		policy := b.store.GetPolicy(helpers.NamespacedName{Namespace: ns, Name: policyRef.Name})
		if policy == nil {
			return nil, fmt.Errorf("rbac policy %s/%s not found", ns, policyRef.Name)
		}
		if _, ok := rules.Policies[policy.Name]; ok {
			return nil, fmt.Errorf("policy '%s' already exist in RBAC", policy.Name)
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

	return &rbacFilter.RBAC{Rules: rules}, nil
}

// BuildUpgradeConfigs builds upgrade configurations
func (b *Builder) BuildUpgradeConfigs(rawUpgradeConfigs []*runtime.RawExtension) ([]*hcmv3.HttpConnectionManager_UpgradeConfig, error) {
	upgradeConfigs := make([]*hcmv3.HttpConnectionManager_UpgradeConfig, 0, len(rawUpgradeConfigs))
	for _, upgradeConfig := range rawUpgradeConfigs {
		uc := &hcmv3.HttpConnectionManager_UpgradeConfig{}
		if err := protoutil.Unmarshaler.Unmarshal(upgradeConfig.Raw, uc); err != nil {
			return upgradeConfigs, err
		}
		if err := uc.ValidateAll(); err != nil {
			return upgradeConfigs, err
		}
		upgradeConfigs = append(upgradeConfigs, uc)
	}

	return upgradeConfigs, nil
}

// BuildAccessLogConfigs builds access log configurations
func (b *Builder) BuildAccessLogConfigs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	var i int

	if vs.Spec.AccessLog != nil {
		i++
	}
	if vs.Spec.AccessLogConfig != nil {
		i++
	}
	if len(vs.Spec.AccessLogs) > 0 {
		i++
	}
	if len(vs.Spec.AccessLogConfigs) > 0 {
		i++
	}
	if i == 0 {
		return nil, nil
	}
	if i > 1 {
		return nil, fmt.Errorf("can't use accessLog, accessLogConfig, accessLogs and accessLogConfigs at the same time")
	}

	// Pre-allocate based on the configuration type
	var capacity int
	if vs.Spec.AccessLog != nil || vs.Spec.AccessLogConfig != nil {
		capacity = 1
	} else if len(vs.Spec.AccessLogs) > 0 {
		capacity = len(vs.Spec.AccessLogs)
	} else if len(vs.Spec.AccessLogConfigs) > 0 {
		capacity = len(vs.Spec.AccessLogConfigs)
	}
	accessLogConfigs := make([]*accesslogv3.AccessLog, 0, capacity)

	if vs.Spec.AccessLog != nil {
		vs.UpdateStatus(false, "accessLog is deprecated, use accessLogs instead")
		var accessLog accesslogv3.AccessLog
		if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, &accessLog); err != nil {
			return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
		}
		if err := accessLog.ValidateAll(); err != nil {
			return nil, err
		}
		accessLogConfigs = append(accessLogConfigs, &accessLog)
		return accessLogConfigs, nil
	}

	if vs.Spec.AccessLogConfig != nil {
		vs.UpdateStatus(false, "accessLogConfig is deprecated, use accessLogConfigs instead")
		accessLogNs := helpers.GetNamespace(vs.Spec.AccessLogConfig.Namespace, vs.Namespace)
		accessLogConfig := b.store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: vs.Spec.AccessLogConfig.Name})
		if accessLogConfig == nil {
			return nil, fmt.Errorf("can't find accessLogConfig %s/%s", accessLogNs, vs.Spec.AccessLogConfig.Name)
		}
		accessLog, err := accessLogConfig.UnmarshalAndValidateV3(v1alpha1.WithAccessLogFileName(vs.Name))
		if err != nil {
			return nil, err
		}
		accessLogConfigs = append(accessLogConfigs, accessLog)
		return accessLogConfigs, nil
	}

	if len(vs.Spec.AccessLogs) > 0 {
		for _, accessLog := range vs.Spec.AccessLogs {
			var accessLogV3 accesslogv3.AccessLog
			if err := protoutil.Unmarshaler.Unmarshal(accessLog.Raw, &accessLogV3); err != nil {
				return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
			}
			if err := accessLogV3.ValidateAll(); err != nil {
				return nil, err
			}
			accessLogConfigs = append(accessLogConfigs, &accessLogV3)
		}
		return accessLogConfigs, nil
	}

	for _, accessLogConfig := range vs.Spec.AccessLogConfigs {
		accessLogNs := helpers.GetNamespace(accessLogConfig.Namespace, vs.Namespace)
		accessLog := b.store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: accessLogConfig.Name})
		if accessLog == nil {
			return nil, fmt.Errorf("can't find accessLogConfig %s/%s", accessLogNs, accessLogConfig.Name)
		}
		accessLogV3, err := accessLog.UnmarshalAndValidateV3(v1alpha1.WithAccessLogFileName(vs.Name))
		if err != nil {
			return nil, err
		}
		accessLogConfigs = append(accessLogConfigs, accessLogV3)
	}
	return accessLogConfigs, nil
}
