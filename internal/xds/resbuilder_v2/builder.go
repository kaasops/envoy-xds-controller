package resbuilder_v2

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/clusters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/config"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/main_builder"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/routes"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/secrets"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v1 "k8s.io/api/core/v1"
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
	result := *resultPtr

	// Create deep copies to avoid mutation issues
	for _, filter := range filters {
		*resultPtr = append(*resultPtr, proto.Clone(filter).(*hcmv3.HttpFilter))
	}

	// Create a new slice to return to the caller - we can't return the pooled slice directly
	finalResult := make([]*hcmv3.HttpFilter, len(result))
	copy(finalResult, result)

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

var httpFiltersGlobalCache = newHTTPFiltersCache()

// generateHTTPFiltersCacheKey creates a hash-based cache key for HTTP filters configuration
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

type FilterChainsParams struct {
	VSName               string
	UseRemoteAddress     bool
	XFFNumTrustedHops    *uint32
	RouteConfigName      string
	StatPrefix           string
	HTTPFilters          []*hcmv3.HttpFilter
	UpgradeConfigs       []*hcmv3.HttpConnectionManager_UpgradeConfig
	AccessLogs           []*accesslogv3.AccessLog
	Domains              []string
	DownstreamTLSContext *tlsv3.DownstreamTlsContext
	SecretNameToDomains  map[helpers.NamespacedName][]string
	IsTLS                bool
	Tracing              *hcmv3.HttpConnectionManager_Tracing
}

// ResourceBuilder provides a modular approach to building Envoy resources
type ResourceBuilder struct {
	store           *store.Store
	clustersBuilder *clusters.Builder
	filtersBuilder  *filters.Builder
	routesBuilder   *routes.Builder
	secretsBuilder  *secrets.Builder
	mainBuilder     interfaces.MainBuilder
	useMainBuilder  bool // Flag to control which implementation to use
}

// NewResourceBuilder creates a new ResourceBuilder with all modular components
func NewResourceBuilder(store *store.Store) *ResourceBuilder {
	// Create a ResourceBuilder with default settings
	rb := &ResourceBuilder{
		store:           store,
		clustersBuilder: clusters.NewBuilder(store),
		filtersBuilder:  filters.NewBuilder(store),
		routesBuilder:   routes.NewBuilder(store),
		secretsBuilder:  secrets.NewBuilder(store),
		useMainBuilder:  false, // Default to original implementation
	}

	// Apply feature flags from environment
	rb.UpdateFeatureFlags()

	return rb
}

// EnableMainBuilder toggles the use of the MainBuilder implementation
// When enabled, BuildResources will use the new modular MainBuilder
// When disabled, it will use the original implementation
func (rb *ResourceBuilder) EnableMainBuilder(enable bool) {
	rb.useMainBuilder = enable

	// Initialize MainBuilder if it's not already set and we're enabling it
	if enable && rb.mainBuilder == nil {
		// Use the resource_builder_adapter.go functionality to set up the MainBuilder
		UpdateResourceBuilder(rb)
	}
}

// UpdateFeatureFlags updates the ResourceBuilder configuration based on current feature flags
// This can be called periodically to pick up changes in environment variables
func (rb *ResourceBuilder) UpdateFeatureFlags() {
	// Get feature flags from environment
	flags := config.GetFeatureFlags()

	// Update useMainBuilder flag
	if flags.EnableMainBuilder != rb.useMainBuilder {
		rb.EnableMainBuilder(flags.EnableMainBuilder)
	}
}

// BuildResources builds all Envoy resources using modular builders
func (rb *ResourceBuilder) BuildResources(vs *v1alpha1.VirtualService) (*Resources, error) {
	// Get namespaced name for the VirtualService
	nn := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}

	// Determine whether to use MainBuilder based on feature flags and rollout strategy
	// If useMainBuilder is true, always use it regardless of percentage
	// Otherwise, use the percentage-based rollout strategy
	flags := config.GetFeatureFlags()
	useMainBuilder := rb.useMainBuilder || config.ShouldUseMainBuilder(flags, nn.String())

	// If MainBuilder should be used, use it
	if useMainBuilder {
		return rb.buildResourcesWithMainBuilder(vs)
	}

	// Otherwise, use the original implementation
	var err error

	vsPtr := vs

	// Apply template if specified
	vs, err = rb.applyVirtualServiceTemplate(vs)
	if err != nil {
		return nil, err
	}

	// Build listener
	listenerNN, err := vs.GetListenerNamespacedName()
	if err != nil {
		return nil, err
	}

	xdsListener, err := rb.buildListener(listenerNN)
	if err != nil {
		return nil, err
	}

	// If the listener already has filter chains, use them
	if len(xdsListener.FilterChains) > 0 {
		return rb.buildResourcesFromExistingFilterChains(vs, xdsListener, listenerNN)
	}

	// Otherwise, build resources from virtual service configuration
	resources, err := rb.buildResourcesFromVirtualService(vs, xdsListener, listenerNN, nn)
	if err != nil {
		return nil, err
	}

	if vs.Status.Message != "" {
		vsPtr.UpdateStatus(vs.Status.Invalid, vs.Status.Message)
	}

	return resources, nil
}

// buildResourcesWithMainBuilder builds resources using the MainBuilder implementation
func (rb *ResourceBuilder) buildResourcesWithMainBuilder(vs *v1alpha1.VirtualService) (*Resources, error) {
	// Input validation
	if vs == nil {
		return nil, fmt.Errorf("virtual service cannot be nil")
	}

	// Make sure MainBuilder is initialized
	if rb.mainBuilder == nil {
		UpdateResourceBuilder(rb)

		// Double-check initialization was successful
		if rb.mainBuilder == nil {
			return nil, fmt.Errorf("failed to initialize MainBuilder")
		}
	}

	// Call MainBuilder.BuildResources with timeout and panic recovery
	var result interface{}
	var err error

	// Use panic recovery to handle any unexpected panics in the MainBuilder
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in MainBuilder.BuildResources: %v", r)
			}
		}()

		result, err = rb.mainBuilder.BuildResources(vs)
	}()

	// Check for errors from BuildResources or from panic recovery
	if err != nil {
		return nil, fmt.Errorf("MainBuilder.BuildResources failed: %w", err)
	}

	// Check for nil result
	if result == nil {
		return nil, fmt.Errorf("MainBuilder.BuildResources returned nil result")
	}

	// Convert result from interface{} to *main_builder.Resources
	// Type assertion to get the concrete type
	mainResources, ok := result.(*main_builder.Resources)
	if !ok {
		return nil, fmt.Errorf("unexpected result type from MainBuilder: %T", result)
	}

	// Validate required fields
	if mainResources.Listener.Name == "" {
		return nil, fmt.Errorf("invalid result from MainBuilder: Listener name is empty")
	}

	if len(mainResources.FilterChain) == 0 {
		return nil, fmt.Errorf("invalid result from MainBuilder: FilterChain is empty")
	}

	// Optional fields validation with warnings
	if mainResources.RouteConfig == nil && len(mainResources.Clusters) == 0 {
		// This is a warning rather than an error because some configurations might be valid without these
		// But it's unusual enough to log
		fmt.Printf("Warning: MainBuilder returned resources without RouteConfig and Clusters for %s\n",
			mainResources.Listener.String())
	}

	// Convert from main_builder.Resources to resbuilder_v2.Resources
	resources := &Resources{
		Listener:    mainResources.Listener,
		FilterChain: mainResources.FilterChain,
		RouteConfig: mainResources.RouteConfig,
		Clusters:    mainResources.Clusters,
		Secrets:     mainResources.Secrets,
		UsedSecrets: mainResources.UsedSecrets,
		Domains:     mainResources.Domains,
	}

	return resources, nil
}

type Resources struct {
	Listener    helpers.NamespacedName
	FilterChain []*listenerv3.FilterChain
	RouteConfig *routev3.RouteConfiguration
	Clusters    []*cluster.Cluster
	Secrets     []*tlsv3.Secret
	UsedSecrets []helpers.NamespacedName
	Domains     []string
}

// BuildResources is the main entry point for building Envoy resources using the modular architecture
func BuildResources(vs *v1alpha1.VirtualService, store *store.Store) (*Resources, error) {
	// Create a ResourceBuilder instance with all modular components
	builder := NewResourceBuilder(store)

	// Delegate to the modular BuildResources method
	return builder.BuildResources(vs)
}

// applyVirtualServiceTemplate applies a template to the virtual service if specified
func (rb *ResourceBuilder) applyVirtualServiceTemplate(vs *v1alpha1.VirtualService) (*v1alpha1.VirtualService, error) {
	if vs.Spec.Template == nil {
		return vs, nil
	}

	templateNamespace := helpers.GetNamespace(vs.Spec.Template.Namespace, vs.Namespace)
	templateName := vs.Spec.Template.Name
	templateNN := helpers.NamespacedName{Namespace: templateNamespace, Name: templateName}

	vst := rb.store.GetVirtualServiceTemplate(templateNN)
	if vst == nil {
		return nil, fmt.Errorf("virtual service template %s/%s not found", templateNamespace, templateName)
	}

	vsCopy := vs.DeepCopy()
	if err := vsCopy.FillFromTemplate(vst, vs.Spec.TemplateOptions...); err != nil {
		return nil, err
	}

	return vsCopy, nil
}

// buildResourcesFromExistingFilterChains builds resources using existing filter chains from the listener
func (rb *ResourceBuilder) buildResourcesFromExistingFilterChains(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, listenerNN helpers.NamespacedName) (*Resources, error) {
	// Check for conflicts with virtual service configuration
	if err := checkFilterChainsConflicts(vs); err != nil {
		return nil, err
	}

	if len(xdsListener.FilterChains) > 1 {
		return nil, fmt.Errorf("multiple filter chains found")
	}

	// Extract clusters from filter chains
	clusters, err := clusters.ExtractClustersFromFilterChains(xdsListener.FilterChains, rb.store)
	if err != nil {
		return nil, err
	}

	return &Resources{
		Listener:    listenerNN,
		FilterChain: xdsListener.FilterChains,
		Clusters:    clusters,
	}, nil
}

// checkFilterChainsConflicts checks for conflicts between existing filter chains and virtual service configuration
func checkFilterChainsConflicts(vs *v1alpha1.VirtualService) error {
	conflicts := []struct {
		condition bool
		message   string
	}{
		{vs.Spec.VirtualHost != nil, "virtual host is set, but filter chains are found in listener"},
		{len(vs.Spec.AdditionalRoutes) > 0, "additional routes are set, but filter chains are found in listener"},
		{len(vs.Spec.HTTPFilters) > 0, "http filters are set, but filter chains are found in listener"},
		{len(vs.Spec.AdditionalHttpFilters) > 0, "additional http filters are set, but filter chains are found in listener"},
		{vs.Spec.TlsConfig != nil, "tls config is set, but filter chains are found in listener"},
		{vs.Spec.RBAC != nil, "rbac is set, but filter chains are found in listener"},
		{vs.Spec.UseRemoteAddress != nil, "use remote address is set, but filter chains are found in listener"},
		{vs.Spec.XFFNumTrustedHops != nil, "xff_num_trusted_hops is set, but filter chains are found in listener"},
		{vs.Spec.UpgradeConfigs != nil, "upgrade configs is set, but filter chains are found in listener"},
		{vs.Spec.AccessLog != nil, "access log is set, but filter chains are found in listener"},
		{vs.Spec.AccessLogConfig != nil, "access log config is set, but filter chains are found in listener"},
		{len(vs.Spec.AccessLogs) > 0, "access logs are set, but filter chains are found in listener"},
		{len(vs.Spec.AccessLogConfigs) > 0, "access log configs are set, but filter chains are found in listener"},
	}

	for _, conflict := range conflicts {
		if conflict.condition {
			return fmt.Errorf("conflict: %s", conflict.message)
		}
	}

	return nil
}

// extractClustersFromFilterChains extracts clusters from filter chains
func extractClustersFromFilterChains(filterChains []*listenerv3.FilterChain, store *store.Store) ([]*cluster.Cluster, error) {
	// Get a slice from the pool
	clustersPtr := utils.GetClusterSlice()
	defer utils.PutClusterSlice(clustersPtr)

	for _, fc := range filterChains {
		for _, filter := range fc.Filters {
			if tc := filter.GetTypedConfig(); tc != nil {
				if tc.TypeUrl != "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy" {
					return nil, fmt.Errorf("unexpected filter type: %s", tc.TypeUrl)
				}

				var tcpProxy tcpProxyv3.TcpProxy
				if err := tc.UnmarshalTo(&tcpProxy); err != nil {
					return nil, err
				}

				clusterName := tcpProxy.GetCluster()
				cl := store.GetSpecCluster(clusterName)
				if cl == nil {
					return nil, fmt.Errorf("cluster %s not found", clusterName)
				}

				xdsCluster, err := cl.UnmarshalV3AndValidate()
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", clusterName, err)
				}

				*clustersPtr = append(*clustersPtr, xdsCluster)
			}
		}
	}

	// Create a new slice to return to the caller - we can't return the pooled slice directly
	result := make([]*cluster.Cluster, len(*clustersPtr))
	copy(result, *clustersPtr)

	return result, nil
}

// buildResourcesFromVirtualService builds resources from virtual service configuration
func (rb *ResourceBuilder) buildResourcesFromVirtualService(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, listenerNN helpers.NamespacedName, nn helpers.NamespacedName) (*Resources, error) {
	listenerIsTLS := utils.IsTLSListener(xdsListener)

	// Build virtual host and route configuration
	virtualHost, routeConfiguration, err := buildRouteConfiguration(vs, xdsListener, nn, rb.store)
	if err != nil {
		return nil, err
	}

	// Build HTTP filters using modular builder
	httpFilters, err := rb.filtersBuilder.BuildHTTPFilters(vs)
	if err != nil {
		return nil, err
	}

	// Build filter chain parameters
	filterChainParams, err := buildFilterChainParams(vs, nn, httpFilters, listenerIsTLS, virtualHost, rb.store)
	if err != nil {
		return nil, err
	}

	// Build filter chains
	fcs, err := buildFilterChains(filterChainParams)
	if err != nil {
		return nil, err
	}

	// Update listener with filter chains
	xdsListener.FilterChains = fcs
	if err := xdsListener.ValidateAll(); err != nil {
		return nil, err
	}

	// Build clusters using modular builder
	clusters, err := rb.clustersBuilder.BuildClusters(vs, virtualHost, httpFilters)
	if err != nil {
		return nil, err
	}

	// Build secrets using modular builder
	secrets, usedSecrets, err := rb.secretsBuilder.BuildSecrets(vs, filterChainParams.SecretNameToDomains)
	if err != nil {
		return nil, fmt.Errorf("failed to build secrets: %w", err)
	}

	return &Resources{
		Listener:    listenerNN,
		FilterChain: fcs,
		RouteConfig: routeConfiguration,
		Clusters:    clusters,
		Secrets:     secrets,
		UsedSecrets: usedSecrets,
		Domains:     virtualHost.Domains,
	}, nil
}

// buildRouteConfiguration builds the route configuration from the virtual service
func buildRouteConfiguration(
	vs *v1alpha1.VirtualService,
	xdsListener *listenerv3.Listener,
	nn helpers.NamespacedName,
	store *store.Store,
) (*routev3.VirtualHost, *routev3.RouteConfiguration, error) {
	virtualHost, err := buildVirtualHost(vs, nn, store)
	if err != nil {
		return nil, nil, err
	}

	routeConfiguration := &routev3.RouteConfiguration{
		Name:         nn.String(),
		VirtualHosts: []*routev3.VirtualHost{virtualHost},
	}

	// Add fallback route for TLS listeners
	// https://github.com/envoyproxy/envoy/issues/37810
	listenerIsTLS := utils.IsTLSListener(xdsListener)
	if listenerIsTLS && !(len(virtualHost.Domains) == 1 && virtualHost.Domains[0] == "*") && utils.ListenerHasPort443(xdsListener) {
		routeConfiguration.VirtualHosts = append(routeConfiguration.VirtualHosts, &routev3.VirtualHost{
			Name:    "421vh",
			Domains: []string{"*"},
			Routes: []*routev3.Route{
				{
					Match: &routev3.RouteMatch{PathSpecifier: &routev3.RouteMatch_Prefix{Prefix: "/"}},
					Action: &routev3.Route_DirectResponse{
						DirectResponse: &routev3.DirectResponseAction{
							Status: 421,
						},
					},
				},
			},
		})
	}

	if err = routeConfiguration.ValidateAll(); err != nil {
		return nil, nil, err
	}

	return virtualHost, routeConfiguration, nil
}

// buildFilterChainParams builds the filter chain parameters
func buildFilterChainParams(vs *v1alpha1.VirtualService, nn helpers.NamespacedName, httpFilters []*hcmv3.HttpFilter, listenerIsTLS bool, virtualHost *routev3.VirtualHost, store *store.Store) (*FilterChainsParams, error) {
	filterChainParams := &FilterChainsParams{
		VSName:            nn.String(),
		UseRemoteAddress:  helpers.BoolFromPtr(vs.Spec.UseRemoteAddress),
		RouteConfigName:   nn.String(),
		StatPrefix:        strings.ReplaceAll(nn.String(), ".", "-"),
		HTTPFilters:       httpFilters,
		IsTLS:             listenerIsTLS,
		XFFNumTrustedHops: vs.Spec.XFFNumTrustedHops,
	}
	// Tracing config: enforce XOR between inline and ref; priority inline > ref
	if vs.Spec.Tracing != nil && vs.Spec.TracingRef != nil {
		return nil, fmt.Errorf("only one of spec.tracing or spec.tracingRef may be set")
	}
	if vs.Spec.Tracing != nil {
		tracing := &hcmv3.HttpConnectionManager_Tracing{}
		if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.Tracing.Raw, tracing); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracing: %w", err)
		}
		if err := tracing.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate tracing: %w", err)
		}
		filterChainParams.Tracing = tracing
	} else if vs.Spec.TracingRef != nil {
		tracingRefNs := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
		tr := store.GetTracing(helpers.NamespacedName{Namespace: tracingRefNs, Name: vs.Spec.TracingRef.Name})
		if tr == nil {
			return nil, fmt.Errorf("tracing %s/%s not found", tracingRefNs, vs.Spec.TracingRef.Name)
		}
		trv3, err := tr.UnmarshalV3AndValidate()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracing: %w", err)
		}
		filterChainParams.Tracing = trv3
	}

	// Build upgrade configs
	upgradeConfigs, err := buildUpgradeConfigs(vs.Spec.UpgradeConfigs)
	if err != nil {
		return nil, err
	}
	filterChainParams.UpgradeConfigs = upgradeConfigs

	// Build access log config
	accessLogs, err := buildAccessLogConfigs(vs, store)
	if err != nil {
		return nil, err
	}
	filterChainParams.AccessLogs = accessLogs

	// Check TLS configuration
	if listenerIsTLS && vs.Spec.TlsConfig == nil {
		return nil, fmt.Errorf("tls listener not configured, virtual service has not tls config")
	}
	if !listenerIsTLS && vs.Spec.TlsConfig != nil {
		return nil, fmt.Errorf("listener is not tls, virtual service has tls config")
	}

	// Process TLS configuration
	if vs.Spec.TlsConfig != nil {
		tlsType, err := getTLSType(vs.Spec.TlsConfig)
		if err != nil {
			return nil, err
		}

		switch tlsType {
		case utils.SecretRefType:
			filterChainParams.SecretNameToDomains = getSecretNameToDomainsViaSecretRef(vs.Spec.TlsConfig.SecretRef, vs.Namespace, virtualHost.Domains)
		case utils.AutoDiscoveryType:
			filterChainParams.SecretNameToDomains, err = getSecretNameToDomainsViaAutoDiscovery(virtualHost.Domains, store.MapDomainSecrets())
			if err != nil {
				return nil, err
			}
		}
	}

	return filterChainParams, nil
}

func (rb *ResourceBuilder) buildListener(listenerNN helpers.NamespacedName) (*listenerv3.Listener, error) {
	listener := rb.store.GetListener(listenerNN)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", listenerNN.String())
	}
	xdsListener, err := listener.UnmarshalV3()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal listener %s: %w", listenerNN.String(), err)
	}
	xdsListener.Name = listenerNN.String()
	return xdsListener, nil
}

func buildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName, store *store.Store) (*routev3.VirtualHost, error) {
	if vs.Spec.VirtualHost == nil {
		return nil, fmt.Errorf("virtual host is empty")
	}

	virtualHost := &routev3.VirtualHost{}
	if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return nil, fmt.Errorf("failed to unmarshal virtual host: %w", err)
	}
	virtualHost.Name = nn.String()

	for _, routeRef := range vs.Spec.AdditionalRoutes {
		routeRefNs := helpers.GetNamespace(routeRef.Namespace, vs.Namespace)
		route := store.GetRoute(helpers.NamespacedName{Namespace: routeRefNs, Name: routeRef.Name})
		if route == nil {
			return nil, fmt.Errorf("route %s/%s not found", routeRefNs, routeRef.Name)
		}
		for idx, rt := range route.Spec {
			var r routev3.Route
			if err := protoutil.Unmarshaler.Unmarshal(rt.Raw, &r); err != nil {
				return nil, fmt.Errorf("failed to unmarshal route %s/%s (%d): %w", routeRefNs, routeRef.Name, idx, err)
			}
			if rr := r.GetRoute(); rr != nil {
				if clName := rr.GetCluster(); clName != "" {
					cl := store.GetSpecCluster(clName)
					if cl == nil {
						return nil, fmt.Errorf("cluster %s not found", clName)
					}
				}
			}
			virtualHost.Routes = append(virtualHost.Routes, &r)
		}
	}

	rootMatchIndexes := make([]int, 0, 1)
	// reorder routes, root must be in the end
	for index, route := range virtualHost.Routes {
		if route.Match != nil && (route.Match.GetPrefix() == "/" || route.Match.GetPath() == "/") {
			rootMatchIndexes = append(rootMatchIndexes, index)
		}
	}

	switch {
	case len(rootMatchIndexes) > 1:
		return nil, fmt.Errorf("multiple root routes found")
	case len(rootMatchIndexes) == 1 && rootMatchIndexes[0] != len(virtualHost.Routes)-1:
		index := rootMatchIndexes[0]
		route := virtualHost.Routes[index]
		virtualHost.Routes = append(virtualHost.Routes[:index], virtualHost.Routes[index+1:]...)
		virtualHost.Routes = append(virtualHost.Routes, route)
	}

	if err := virtualHost.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate virtual host: %w", err)
	}

	if err := utils.CheckAllDomainsUnique(virtualHost.Domains); err != nil {
		return nil, err
	}

	return virtualHost, nil
}

func buildHTTPFilters(vs *v1alpha1.VirtualService, store *store.Store) ([]*hcmv3.HttpFilter, error) {
	// Check cache first
	cacheKey := generateHTTPFiltersCacheKey(vs, store)
	if cached, exists := httpFiltersGlobalCache.get(cacheKey); exists {
		return cached, nil
	}

	httpFilters := make([]*hcmv3.HttpFilter, 0, len(vs.Spec.HTTPFilters)+len(vs.Spec.AdditionalHttpFilters))

	rbacF, err := buildRBACFilter(vs, store)
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
			hf := store.GetHTTPFilter(helpers.NamespacedName{Namespace: httpFilterRefNs, Name: httpFilterRef.Name})
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
	httpFiltersGlobalCache.set(cacheKey, httpFilters)

	return httpFilters, nil
}

func buildRBACFilter(vs *v1alpha1.VirtualService, store *store.Store) (*rbacFilter.RBAC, error) {
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
		policy := store.GetPolicy(helpers.NamespacedName{Namespace: ns, Name: policyRef.Name})
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

func buildFilterChains(params *FilterChainsParams) ([]*listenerv3.FilterChain, error) {
	var filterChains []*listenerv3.FilterChain

	if len(params.SecretNameToDomains) > 0 {
		for secretName, domains := range params.SecretNameToDomains {
			params.Domains = domains
			params.DownstreamTLSContext = &tlsv3.DownstreamTlsContext{
				CommonTlsContext: &tlsv3.CommonTlsContext{
					TlsCertificateSdsSecretConfigs: []*tlsv3.SdsSecretConfig{{
						Name: secretName.String(),
						SdsConfig: &corev3.ConfigSource{
							ConfigSourceSpecifier: &corev3.ConfigSource_Ads{
								Ads: &corev3.AggregatedConfigSource{},
							},
							ResourceApiVersion: corev3.ApiVersion_V3,
						},
					}},
					AlpnProtocols: []string{"h2", "http/1.1"},
				},
			}
			fc, err := buildFilterChain(params)
			if err != nil {
				return nil, err
			}
			filterChains = append(filterChains, fc)
		}
		return filterChains, nil
	}

	fc, err := buildFilterChain(params)
	if err != nil {
		return nil, err
	}
	filterChains = append(filterChains, fc)
	return filterChains, nil
}

func buildFilterChain(params *FilterChainsParams) (*listenerv3.FilterChain, error) {
	httpConnectionManager := &hcmv3.HttpConnectionManager{
		CodecType:  hcmv3.HttpConnectionManager_AUTO,
		StatPrefix: params.StatPrefix,
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				ConfigSource: &corev3.ConfigSource{
					ResourceApiVersion:    corev3.ApiVersion_V3,
					ConfigSourceSpecifier: &corev3.ConfigSource_Ads{},
				},
				RouteConfigName: params.RouteConfigName,
			},
		},
		UseRemoteAddress: &wrapperspb.BoolValue{Value: params.UseRemoteAddress},
		UpgradeConfigs:   params.UpgradeConfigs,
		HttpFilters:      params.HTTPFilters,
	}
	if params.Tracing != nil {
		httpConnectionManager.Tracing = params.Tracing
	}
	if params.XFFNumTrustedHops != nil {
		httpConnectionManager.XffNumTrustedHops = *params.XFFNumTrustedHops
	}
	if len(params.AccessLogs) > 0 {
		httpConnectionManager.AccessLog = append(httpConnectionManager.AccessLog, params.AccessLogs...)
	}

	if err := httpConnectionManager.ValidateAll(); err != nil {
		return nil, err
	}

	pbst, err := anypb.New(httpConnectionManager)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal httpConnectionManager to anypb: %w", err)
	}

	fc := &listenerv3.FilterChain{}
	fc.Filters = []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}
	if params.IsTLS && len(params.Domains) > 0 && !slices.Contains(params.Domains, "*") {
		fc.FilterChainMatch = &listenerv3.FilterChainMatch{
			ServerNames: params.Domains,
		}
	}
	if params.DownstreamTLSContext != nil {
		scfg, err := anypb.New(params.DownstreamTLSContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal downstreamTlsContext to anypb, %w", err)
		}
		fc.TransportSocket = &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: scfg,
			},
		}
	}
	fc.Name = params.VSName

	if err := fc.ValidateAll(); err != nil {
		return nil, err
	}

	return fc, nil
}

func buildUpgradeConfigs(rawUpgradeConfigs []*runtime.RawExtension) ([]*hcmv3.HttpConnectionManager_UpgradeConfig, error) {
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

func buildAccessLogConfigs(vs *v1alpha1.VirtualService, store *store.Store) ([]*accesslogv3.AccessLog, error) {
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
		accessLogConfig := store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: vs.Spec.AccessLogConfig.Name})
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
		accessLog := store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: accessLogConfig.Name})
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

func getTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error) {
	if vsTLSConfig == nil {
		return "", fmt.Errorf("TLS configuration is missing: please provide TLS parameters")
	}
	if vsTLSConfig.SecretRef != nil {
		if vsTLSConfig.AutoDiscovery != nil && *vsTLSConfig.AutoDiscovery {
			return "", fmt.Errorf("TLS configuration conflict: cannot use both secretRef and autoDiscovery simultaneously")
		}
		return utils.SecretRefType, nil
	}
	if vsTLSConfig.AutoDiscovery != nil {
		if !*vsTLSConfig.AutoDiscovery {
			return "", fmt.Errorf("invalid TLS configuration: cannot use autoDiscovery=false without specifying secretRef")
		}
		return utils.AutoDiscoveryType, nil
	}
	return "", fmt.Errorf("empty TLS configuration: either secretRef or autoDiscovery must be specified")
}

func getSecretNameToDomainsViaSecretRef(secretRef *v1alpha1.ResourceRef, vsNamespace string, domains []string) map[helpers.NamespacedName][]string {
	m := make(map[helpers.NamespacedName][]string)

	var secretNamespace string

	if secretRef.Namespace != nil {
		secretNamespace = *secretRef.Namespace
	} else {
		secretNamespace = vsNamespace
	}

	m[helpers.NamespacedName{Namespace: secretNamespace, Name: secretRef.Name}] = domains
	return m
}

func getSecretNameToDomainsViaAutoDiscovery(domains []string, domainToSecretMap map[string]v1.Secret) (map[helpers.NamespacedName][]string, error) {
	m := make(map[helpers.NamespacedName][]string)

	for _, domain := range domains {
		var secret v1.Secret
		secret, ok := domainToSecretMap[domain]
		if !ok {
			secret, ok = domainToSecretMap[getWildcardDomain(domain)]
			if !ok {
				return nil, fmt.Errorf("can't find secret for domain %s", domain)
			}
		}

		domainsFromMap, ok := m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}]
		if ok {
			m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = append(domainsFromMap, domain)
		} else {
			m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = []string{domain}
		}
	}

	return m, nil
}

func getWildcardDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	parts[0] = "*"
	return strings.Join(parts, ".")
}

func findClusterNames(data interface{}, fieldName string) []string {
	var results []string

	switch value := data.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k == fieldName {
				results = append(results, fmt.Sprintf("%v", v))
			}
			results = append(results, findClusterNames(v, fieldName)...)
		}
	case []interface{}:
		for _, item := range value {
			results = append(results, findClusterNames(item, fieldName)...)
		}
	}

	return results
}

func findSDSNames(data interface{}, fieldName string) []string {
	var results []string

	switch value := data.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k == fieldName {
				results = append(results, fmt.Sprintf("%v", value["name"]))
			}
			results = append(results, findSDSNames(v, fieldName)...)
		}
	case []interface{}:
		for _, item := range value {
			results = append(results, findSDSNames(item, fieldName)...)
		}
	}

	return results
}

func isTLSListener(xdsListener *listenerv3.Listener) bool {
	if xdsListener == nil {
		return false
	}
	if len(xdsListener.ListenerFilters) == 0 {
		return false
	}
	for _, lFilter := range xdsListener.ListenerFilters {
		if tc := lFilter.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector" {
				return true
			}
		}
	}
	return false
}

func checkAllDomainsUnique(domains []string) error {
	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		if _, exists := seen[domain]; exists {
			return fmt.Errorf("duplicate domain found: %s", domain)
		}
		seen[domain] = struct{}{}
	}
	return nil
}
