package main_builder

import (
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/utils"
)

// Cache for BuildResources results to avoid expensive re-computation
// Uses container/list for O(1) LRU operations (OPT-002)
type resourcesCache struct {
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
	ll       *list.List               // doubly linked list for LRU ordering
	elements map[string]*list.Element // map for O(1) lookups
}

// cacheEntry represents a cached item with metadata
type cacheEntry struct {
	key       string
	resource  *Resources
	createdAt time.Time
}

func newResourcesCache() *resourcesCache {
	return &resourcesCache{
		maxSize:  100,             // Limit cache size
		ttl:      5 * time.Minute, // Default TTL
		ll:       list.New(),
		elements: make(map[string]*list.Element),
	}
}

// get retrieves a resource from the cache if it exists and is not expired
func (c *resourcesCache) get(key string) (*Resources, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.elements[key]
	if !exists {
		utils.RecordCacheMiss("main_builder")
		return nil, false
	}

	entry := element.Value.(*cacheEntry)

	// Check if the entry has expired
	if time.Since(entry.createdAt) > c.ttl {
		// Remove expired entry - O(1) with container/list
		c.ll.Remove(element)
		delete(c.elements, key)
		utils.RecordCacheEviction("main_builder", "ttl_expired")
		utils.RecordCacheMiss("main_builder")
		utils.UpdateCacheSize("main_builder", c.ll.Len())
		return nil, false
	}

	// Move to front (most recently used) - O(1)
	c.ll.MoveToFront(element)

	// Record cache hit and item age
	utils.RecordCacheHit("main_builder")
	ageSeconds := time.Since(entry.createdAt).Seconds()
	utils.CacheItemAge.WithLabelValues("main_builder", "hit").Observe(ageSeconds)

	return entry.resource, true
}

// set adds a resource to the cache
func (c *resourcesCache) set(key string, resource *Resources) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key already exists, update it and move to front - O(1)
	if element, exists := c.elements[key]; exists {
		c.ll.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		entry.resource = resource
		entry.createdAt = time.Now()
		return
	}

	// LRU eviction: if cache is full, remove the oldest entry - O(1)
	if c.ll.Len() >= c.maxSize {
		oldest := c.ll.Back()
		if oldest != nil {
			oldEntry := oldest.Value.(*cacheEntry)
			c.ll.Remove(oldest)
			delete(c.elements, oldEntry.key)
			utils.RecordCacheEviction("main_builder", "lru_eviction")
		}
	}

	// Create new entry and add to front - O(1)
	entry := &cacheEntry{
		key:       key,
		resource:  resource,
		createdAt: time.Now(),
	}
	element := c.ll.PushFront(entry)
	c.elements[key] = element

	// Update cache size metric
	utils.UpdateCacheSize("main_builder", c.ll.Len())
}

// generateCacheKey creates a hash-based cache key for a VirtualService
func generateCacheKey(vs *v1alpha1.VirtualService) string {
	hasher := sha256.New()

	// Include name and namespace
	hasher.Write([]byte(vs.Name))
	hasher.Write([]byte(vs.Namespace))

	// Include generation number which changes on updates
	hasher.Write([]byte(fmt.Sprintf("%d", vs.Generation)))

	// Include spec data
	if specData, err := json.Marshal(vs.Spec); err == nil {
		hasher.Write(specData)
	}

	// Return hex-encoded hash as the cache key
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Builder is responsible for coordinating the building of all resources for a VirtualService
type Builder struct {
	store              store.Store
	httpFilterBuilder  interfaces.HTTPFilterBuilder
	filterChainBuilder interfaces.FilterChainBuilder
	routingBuilder     interfaces.RoutingBuilder
	tlsBuilder         interfaces.TLSBuilder
	clusterExtractor   interfaces.ClusterExtractor
	cache              *resourcesCache
}

// BuildResources is the main entry point for building Envoy resources for a VirtualService
// It returns an interface{} that is actually a *Resources to match the MainBuilder interface
func (b *Builder) BuildResources(vs *v1alpha1.VirtualService) (interface{}, error) {
	// Start measuring build time
	startTime := time.Now()

	// Check if result is in cache
	cacheKey := generateCacheKey(vs)
	if cachedResources, found := b.cache.get(cacheKey); found {
		// Record build duration (cache hit is very fast)
		buildDuration := time.Since(startTime).Seconds()
		utils.RecordBuildDuration("virtual_service", "cache_hit", buildDuration)

		// Return cached result
		return cachedResources, nil
	}

	var err error
	nn := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}

	// Store original VS pointer to update status if needed
	vsPtr := vs

	// Apply template if specified
	vs, err = b.applyVirtualServiceTemplate(vs)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template: %w", err)
	}

	// Build listener
	listenerNN, err := vs.GetListenerNamespacedName()
	if err != nil {
		return nil, fmt.Errorf("failed to get listener namespaced name: %w", err)
	}

	xdsListener, err := b.buildListener(listenerNN)
	if err != nil {
		return nil, fmt.Errorf("failed to build listener: %w", err)
	}

	// If the listener already has filter chains, use them
	if len(xdsListener.FilterChains) > 0 {
		return b.buildResourcesFromExistingFilterChains(vs, xdsListener, listenerNN)
	}

	// Otherwise, build resources from virtual service configuration
	resources, err := b.buildResourcesFromVirtualService(vs, xdsListener, listenerNN, nn)
	if err != nil {
		return nil, fmt.Errorf("failed to build resources from virtual service: %w", err)
	}

	// Update status if needed
	if vs.Status.Message != "" {
		vsPtr.UpdateStatus(vs.Status.Invalid, vs.Status.Message)
	}

	// Record resource creation
	utils.RecordResourceCreation("virtual_service")

	// Record build duration
	buildDuration := time.Since(startTime).Seconds()
	utils.RecordBuildDuration("virtual_service", "build", buildDuration)

	// Store result in cache
	b.cache.set(cacheKey, resources)

	return resources, nil
}

// applyVirtualServiceTemplate applies a template to the virtual service if specified
func (b *Builder) applyVirtualServiceTemplate(vs *v1alpha1.VirtualService) (*v1alpha1.VirtualService, error) {
	if vs.Spec.Template == nil {
		return vs, nil
	}

	templateNamespace := helpers.GetNamespace(vs.Spec.Template.Namespace, vs.Namespace)
	templateName := vs.Spec.Template.Name
	templateNN := helpers.NamespacedName{Namespace: templateNamespace, Name: templateName}

	vst := b.store.GetVirtualServiceTemplate(templateNN)
	if vst == nil {
		return nil, fmt.Errorf("virtual service template %s/%s not found", templateNamespace, templateName)
	}

	vsCopy := vs.DeepCopy()
	if err := vsCopy.FillFromTemplate(vst, vs.Spec.TemplateOptions...); err != nil {
		return nil, fmt.Errorf("failed to fill from template: %w", err)
	}

	return vsCopy, nil
}

// buildListener builds a listener from a namespaced name
func (b *Builder) buildListener(listenerNN helpers.NamespacedName) (*listenerv3.Listener, error) {
	listener := b.store.GetListener(listenerNN)
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

// buildResourcesFromExistingFilterChains builds resources using existing filter chains from the listener
func (b *Builder) buildResourcesFromExistingFilterChains(
	vs *v1alpha1.VirtualService,
	xdsListener *listenerv3.Listener,
	listenerNN helpers.NamespacedName,
) (*Resources, error) {
	// Check for conflicts with virtual service configuration
	if err := b.filterChainBuilder.CheckFilterChainsConflicts(vs); err != nil {
		return nil, fmt.Errorf("filter chain conflicts: %w", err)
	}

	// Extract clusters from filter chains
	clusters, err := b.clusterExtractor.ExtractClustersFromFilterChains(xdsListener.FilterChains)
	if err != nil {
		return nil, fmt.Errorf("failed to extract clusters from filter chains: %w", err)
	}

	return &Resources{
		Listener:    listenerNN,
		FilterChain: xdsListener.FilterChains,
		Clusters:    clusters,
	}, nil
}

// buildResourcesFromVirtualService builds resources from a virtual service configuration
func (b *Builder) buildResourcesFromVirtualService(
	vs *v1alpha1.VirtualService,
	xdsListener *listenerv3.Listener,
	listenerNN helpers.NamespacedName,
	nn helpers.NamespacedName,
) (*Resources, error) {
	// 1. Build HTTP filters
	httpFilters, err := b.httpFilterBuilder.BuildHTTPFilters(vs)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP filters: %w", err)
	}

	// 2. Build route configuration
	virtualHost, routeConfig, err := b.routingBuilder.BuildRouteConfiguration(vs, xdsListener, nn)
	if err != nil {
		return nil, fmt.Errorf("failed to build route configuration: %w", err)
	}

	// 3. Check if listener is TLS
	listenerIsTLS := utils.IsTLSListener(xdsListener)

	// 4. Build filter chain parameters
	params, err := b.filterChainBuilder.BuildFilterChainParams(vs, nn, httpFilters, listenerIsTLS, virtualHost)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter chain parameters: %w", err)
	}

	// 5. Build filter chains
	filterChains, err := b.filterChainBuilder.BuildFilterChains(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter chains: %w", err)
	}

	// 6. Extract domains from virtual host
	domains := virtualHost.Domains

	// 7. Create initial resources structure
	resources := &Resources{
		Listener:    listenerNN,
		FilterChain: filterChains,
		RouteConfig: routeConfig,
		Domains:     domains,
	}

	// 8. Extract clusters from various sources
	// 8.1 Extract clusters from virtual host routes
	virtualHostClusters, err := b.clusterExtractor.ExtractClustersFromVirtualHost(virtualHost)
	if err != nil {
		return nil, fmt.Errorf("failed to extract clusters from virtual host: %w", err)
	}
	resources.Clusters = append(resources.Clusters, virtualHostClusters...)

	// 8.2 Extract clusters from HTTP filters
	httpFilterClusters, err := b.clusterExtractor.ExtractClustersFromHTTPFilters(httpFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to extract clusters from HTTP filters: %w", err)
	}
	resources.Clusters = append(resources.Clusters, httpFilterClusters...)

	// 8.3 Extract clusters from inline tracing configuration
	if vs.Spec.Tracing != nil {
		tracingClusters, err := b.clusterExtractor.ExtractClustersFromTracingRaw(vs.Spec.Tracing)
		if err != nil {
			return nil, fmt.Errorf("failed to extract clusters from inline tracing: %w", err)
		}
		resources.Clusters = append(resources.Clusters, tracingClusters...)
	}

	// 8.4 Extract clusters from tracing reference
	if vs.Spec.TracingRef != nil {
		tracingRefClusters, err := b.clusterExtractor.ExtractClustersFromTracingRef(vs)
		if err != nil {
			return nil, fmt.Errorf("failed to extract clusters from tracing ref: %w", err)
		}
		resources.Clusters = append(resources.Clusters, tracingRefClusters...)
	}

	// 9. Build TLS configuration if needed
	if listenerIsTLS && vs.Spec.TlsConfig != nil {
		secretNameToDomains := params.SecretNameToDomains
		if len(secretNameToDomains) > 0 {
			var secrets []*tlsv3.Secret
			var usedSecrets []helpers.NamespacedName

			// For each secret, build a TLS secret
			for secretName := range secretNameToDomains {
				secret, err := b.buildSecret(secretName)
				if err != nil {
					return nil, fmt.Errorf("failed to build secret %s: %w", secretName.String(), err)
				}
				secrets = append(secrets, secret)
				usedSecrets = append(usedSecrets, secretName)
			}

			resources.Secrets = secrets
			resources.UsedSecrets = usedSecrets
		}
	}

	return resources, nil
}

// buildSecret builds a TLS secret from a namespaced name
func (b *Builder) buildSecret(secretName helpers.NamespacedName) (*tlsv3.Secret, error) {
	k8sSecret := b.store.GetSecret(secretName)
	if k8sSecret == nil {
		return nil, fmt.Errorf("Kubernetes secret %s not found", secretName.String())
	}

	// Validate and extract certificate data
	certData, exists := k8sSecret.Data["tls.crt"]
	if !exists || len(certData) == 0 {
		return nil, fmt.Errorf("certificate data not found in secret %s", secretName.String())
	}

	keyData, exists := k8sSecret.Data["tls.key"]
	if !exists || len(keyData) == 0 {
		return nil, fmt.Errorf("private key data not found in secret %s", secretName.String())
	}

	// Build TLS certificate configuration
	tlsCert := &tlsv3.TlsCertificate{
		CertificateChain: &corev3.DataSource{
			Specifier: &corev3.DataSource_InlineBytes{
				InlineBytes: certData,
			},
		},
		PrivateKey: &corev3.DataSource{
			Specifier: &corev3.DataSource_InlineBytes{
				InlineBytes: keyData,
			},
		},
	}

	// Create Envoy TLS secret
	secret := &tlsv3.Secret{
		Name: secretName.String(),
		Type: &tlsv3.Secret_TlsCertificate{
			TlsCertificate: tlsCert,
		},
	}

	return secret, nil
}
