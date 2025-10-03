package store

import (
	"context"
	"strings"
	"sync"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OptimizedStore is a truly optimized implementation with real performance improvements
type OptimizedStore struct {
	mu sync.RWMutex

	// Unified resource storage with string interning
	stringPool *StringPool

	// Core storage - unified approach with better cache locality
	virtualServices map[helpers.NamespacedName]*v1alpha1.VirtualService
	listeners       map[helpers.NamespacedName]*v1alpha1.Listener
	clusters        map[helpers.NamespacedName]*v1alpha1.Cluster

	// Fast lookup indices - maintained incrementally, not rebuilt
	uidToVS       map[string]*v1alpha1.VirtualService
	uidToListener map[string]*v1alpha1.Listener
	uidToCluster  map[string]*v1alpha1.Cluster

	// Template index for efficient VirtualService lookups
	templateToVS map[helpers.NamespacedName][]*v1alpha1.VirtualService

	// Other resources - complete implementation
	virtualServiceTemplates      map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate
	virtualServiceTemplatesByUID map[string]*v1alpha1.VirtualServiceTemplate

	routes      map[helpers.NamespacedName]*v1alpha1.Route
	routesByUID map[string]*v1alpha1.Route

	httpFilters      map[helpers.NamespacedName]*v1alpha1.HttpFilter
	httpFiltersByUID map[string]*v1alpha1.HttpFilter

	accessLogs      map[helpers.NamespacedName]*v1alpha1.AccessLogConfig
	accessLogsByUID map[string]*v1alpha1.AccessLogConfig

	secrets map[helpers.NamespacedName]*corev1.Secret

	policies      map[helpers.NamespacedName]*v1alpha1.Policy
	policiesByUID map[string]*v1alpha1.Policy

	tracings      map[helpers.NamespacedName]*v1alpha1.Tracing
	tracingsByUID map[string]*v1alpha1.Tracing

	// Additional indices
	specClusters  map[string]*v1alpha1.Cluster
	domainSecrets map[string]corev1.Secret
	nodeDomains   map[string]map[string]struct{}
}

// NewOptimizedStore creates a new truly optimized store
func NewOptimizedStore() Store {
	return &OptimizedStore{
		stringPool: NewStringPool(),

		// Pre-allocate maps with realistic capacity to avoid resizing
		virtualServices: make(map[helpers.NamespacedName]*v1alpha1.VirtualService, 1000),
		listeners:       make(map[helpers.NamespacedName]*v1alpha1.Listener, 500),
		clusters:        make(map[helpers.NamespacedName]*v1alpha1.Cluster, 500),

		// UID indices
		uidToVS:       make(map[string]*v1alpha1.VirtualService, 1000),
		uidToListener: make(map[string]*v1alpha1.Listener, 500),
		uidToCluster:  make(map[string]*v1alpha1.Cluster, 500),

		// Template index
		templateToVS: make(map[helpers.NamespacedName][]*v1alpha1.VirtualService, 100),

		// All resource types
		virtualServiceTemplates:      make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate, 100),
		virtualServiceTemplatesByUID: make(map[string]*v1alpha1.VirtualServiceTemplate, 100),

		routes:      make(map[helpers.NamespacedName]*v1alpha1.Route, 500),
		routesByUID: make(map[string]*v1alpha1.Route, 500),

		httpFilters:      make(map[helpers.NamespacedName]*v1alpha1.HttpFilter, 200),
		httpFiltersByUID: make(map[string]*v1alpha1.HttpFilter, 200),

		accessLogs:      make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig, 100),
		accessLogsByUID: make(map[string]*v1alpha1.AccessLogConfig, 100),

		secrets: make(map[helpers.NamespacedName]*corev1.Secret, 200),

		policies:      make(map[helpers.NamespacedName]*v1alpha1.Policy, 100),
		policiesByUID: make(map[string]*v1alpha1.Policy, 100),

		tracings:      make(map[helpers.NamespacedName]*v1alpha1.Tracing, 50),
		tracingsByUID: make(map[string]*v1alpha1.Tracing, 50),

		// Additional indices
		specClusters:  make(map[string]*v1alpha1.Cluster, 500),
		domainSecrets: make(map[string]corev1.Secret, 200),
		nodeDomains:   make(map[string]map[string]struct{}, 100),
	}
}

// VirtualService operations with optimizations
// WARNING: Set* methods mutate input parameters for string interning optimization.
// This is safe in controller-runtime reconcile loops where objects are already owned by the store.

// SetVirtualService adds or updates a VirtualService in the store
// WARNING: This method mutates the input VirtualService (name/namespace interning)
func (s *OptimizedStore) SetVirtualService(vs *v1alpha1.VirtualService) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Intern strings to reduce memory usage
	vs.Name = s.stringPool.Intern(vs.Name)
	vs.Namespace = s.stringPool.InternNamespace(vs.Namespace)
	uid := s.stringPool.InternUID(string(vs.UID))

	key := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}

	// Remove old entry from UID index if exists
	if old := s.virtualServices[key]; old != nil {
		delete(s.uidToVS, string(old.UID))
		s.removeFromTemplateIndex(old)
	}

	// Set new entry
	s.virtualServices[key] = vs
	s.uidToVS[uid] = vs
	s.updateTemplateIndex(vs)
}

func (s *OptimizedStore) GetVirtualService(name helpers.NamespacedName) *v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vs := s.virtualServices[name]
	return vs
}

func (s *OptimizedStore) GetVirtualServiceByUID(uid string) *v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vs := s.uidToVS[uid]
	return vs
}

func (s *OptimizedStore) DeleteVirtualService(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if vs := s.virtualServices[name]; vs != nil {
		delete(s.virtualServices, name)
		delete(s.uidToVS, string(vs.UID))
		s.removeFromTemplateIndex(vs)
	}
}

// Template index management
func (s *OptimizedStore) updateTemplateIndex(vs *v1alpha1.VirtualService) {
	if vs.Spec.Template == nil {
		return
	}

	namespace := vs.Namespace // default to VirtualService's namespace
	if vs.Spec.Template.Namespace != nil {
		namespace = *vs.Spec.Template.Namespace
	}

	templateKey := helpers.NamespacedName{
		Namespace: namespace,
		Name:      vs.Spec.Template.Name,
	}

	// Check if already exists to prevent duplicates
	vsList := s.templateToVS[templateKey]
	for _, existing := range vsList {
		if existing.UID == vs.UID {
			return // Already in index
		}
	}

	s.templateToVS[templateKey] = append(vsList, vs)
}

func (s *OptimizedStore) removeFromTemplateIndex(vs *v1alpha1.VirtualService) {
	if vs.Spec.Template == nil {
		return
	}

	namespace := vs.Namespace // default to VirtualService's namespace
	if vs.Spec.Template.Namespace != nil {
		namespace = *vs.Spec.Template.Namespace
	}

	templateKey := helpers.NamespacedName{
		Namespace: namespace,
		Name:      vs.Spec.Template.Name,
	}

	vsList := s.templateToVS[templateKey]
	for i, existing := range vsList {
		if existing.UID == vs.UID {
			s.templateToVS[templateKey] = append(vsList[:i], vsList[i+1:]...)
			break
		}
	}
}

// Copy creates a shallow copy of the store with independent maps
// This is safe with the immutable pattern where buildSnapshots returns statuses
// instead of mutating VirtualServices directly
func (s *OptimizedStore) Copy() Store {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create new store instance
	newStore := &OptimizedStore{
		stringPool: s.stringPool, // Share string pool (it's immutable-friendly)

		// Create new maps with appropriate capacity
		virtualServices: make(map[helpers.NamespacedName]*v1alpha1.VirtualService, len(s.virtualServices)),
		listeners:       make(map[helpers.NamespacedName]*v1alpha1.Listener, len(s.listeners)),
		clusters:        make(map[helpers.NamespacedName]*v1alpha1.Cluster, len(s.clusters)),

		uidToVS:       make(map[string]*v1alpha1.VirtualService, len(s.uidToVS)),
		uidToListener: make(map[string]*v1alpha1.Listener, len(s.uidToListener)),
		uidToCluster:  make(map[string]*v1alpha1.Cluster, len(s.uidToCluster)),

		templateToVS: make(map[helpers.NamespacedName][]*v1alpha1.VirtualService, len(s.templateToVS)),

		// All resource types
		virtualServiceTemplates:      make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate, len(s.virtualServiceTemplates)),
		virtualServiceTemplatesByUID: make(map[string]*v1alpha1.VirtualServiceTemplate, len(s.virtualServiceTemplatesByUID)),

		routes:      make(map[helpers.NamespacedName]*v1alpha1.Route, len(s.routes)),
		routesByUID: make(map[string]*v1alpha1.Route, len(s.routesByUID)),

		httpFilters:      make(map[helpers.NamespacedName]*v1alpha1.HttpFilter, len(s.httpFilters)),
		httpFiltersByUID: make(map[string]*v1alpha1.HttpFilter, len(s.httpFiltersByUID)),

		accessLogs:      make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig, len(s.accessLogs)),
		accessLogsByUID: make(map[string]*v1alpha1.AccessLogConfig, len(s.accessLogsByUID)),

		secrets: make(map[helpers.NamespacedName]*corev1.Secret, len(s.secrets)),

		policies:      make(map[helpers.NamespacedName]*v1alpha1.Policy, len(s.policies)),
		policiesByUID: make(map[string]*v1alpha1.Policy, len(s.policiesByUID)),

		tracings:      make(map[helpers.NamespacedName]*v1alpha1.Tracing, len(s.tracings)),
		tracingsByUID: make(map[string]*v1alpha1.Tracing, len(s.tracingsByUID)),

		// Additional indices
		specClusters:  make(map[string]*v1alpha1.Cluster, len(s.specClusters)),
		domainSecrets: make(map[string]corev1.Secret, len(s.domainSecrets)),
		nodeDomains:   make(map[string]map[string]struct{}, len(s.nodeDomains)),
	}

	// Copy VirtualServices - shallow copy is now safe with immutable pattern
	// buildSnapshots no longer mutates VirtualServices directly, statuses are applied separately
	for k, v := range s.virtualServices {
		newStore.virtualServices[k] = v
	}
	for k, v := range s.uidToVS {
		newStore.uidToVS[k] = v
	}

	// Copy template index
	for k, vsList := range s.templateToVS {
		newStore.templateToVS[k] = append([]*v1alpha1.VirtualService(nil), vsList...)
	}

	// Copy other resources (shallow copy for performance)
	for k, v := range s.listeners {
		newStore.listeners[k] = v
	}
	for k, v := range s.uidToListener {
		newStore.uidToListener[k] = v
	}

	for k, v := range s.clusters {
		newStore.clusters[k] = v
	}
	for k, v := range s.uidToCluster {
		newStore.uidToCluster[k] = v
	}

	for k, v := range s.secrets {
		newStore.secrets[k] = v
	}

	for k, v := range s.policies {
		newStore.policies[k] = v
	}
	for k, v := range s.policiesByUID {
		newStore.policiesByUID[k] = v
	}

	// Copy VirtualServiceTemplates - shallow copy is now safe
	for k, v := range s.virtualServiceTemplates {
		newStore.virtualServiceTemplates[k] = v
	}
	for k, v := range s.virtualServiceTemplatesByUID {
		newStore.virtualServiceTemplatesByUID[k] = v
	}

	// Copy Routes
	for k, v := range s.routes {
		newStore.routes[k] = v
	}
	for k, v := range s.routesByUID {
		newStore.routesByUID[k] = v
	}

	// Copy HTTPFilters
	for k, v := range s.httpFilters {
		newStore.httpFilters[k] = v
	}
	for k, v := range s.httpFiltersByUID {
		newStore.httpFiltersByUID[k] = v
	}

	// Copy AccessLogs
	for k, v := range s.accessLogs {
		newStore.accessLogs[k] = v
	}
	for k, v := range s.accessLogsByUID {
		newStore.accessLogsByUID[k] = v
	}

	// Copy Tracings
	for k, v := range s.tracings {
		newStore.tracings[k] = v
	}
	for k, v := range s.tracingsByUID {
		newStore.tracingsByUID[k] = v
	}

	// Copy additional indices
	for k, v := range s.specClusters {
		newStore.specClusters[k] = v
	}

	for k, v := range s.domainSecrets {
		newStore.domainSecrets[k] = v
	}

	for nodeID, domains := range s.nodeDomains {
		newDomains := make(map[string]struct{}, len(domains))
		for domain := range domains {
			newDomains[domain] = struct{}{}
		}
		newStore.nodeDomains[nodeID] = newDomains
	}

	return newStore
}

// Listener operations (similar pattern)
func (s *OptimizedStore) SetListener(listener *v1alpha1.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Intern strings
	listener.Name = s.stringPool.Intern(listener.Name)
	listener.Namespace = s.stringPool.InternNamespace(listener.Namespace)
	uid := s.stringPool.InternUID(string(listener.UID))

	key := helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}

	// Remove old entry from UID index if exists
	if old := s.listeners[key]; old != nil {
		delete(s.uidToListener, string(old.UID))
	}

	s.listeners[key] = listener
	s.uidToListener[uid] = listener
}

func (s *OptimizedStore) GetListener(name helpers.NamespacedName) *v1alpha1.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()

	listener := s.listeners[name]
	return listener
}

// Cluster operations
func (s *OptimizedStore) SetCluster(cluster *v1alpha1.Cluster) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Intern strings
	cluster.Name = s.stringPool.Intern(cluster.Name)
	cluster.Namespace = s.stringPool.InternNamespace(cluster.Namespace)
	uid := s.stringPool.InternUID(string(cluster.UID))

	key := helpers.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}

	// Remove old entry from indices if exists
	if old := s.clusters[key]; old != nil {
		delete(s.uidToCluster, string(old.UID))
		// Use UnmarshalV3().Name to find the correct specClusters key
		if oldV3, err := old.UnmarshalV3(); err == nil {
			delete(s.specClusters, oldV3.Name)
		}
	}

	s.clusters[key] = cluster
	s.uidToCluster[uid] = cluster
	// Use UnmarshalV3().Name for specClusters index (like LegacyStore)
	if clusterV3, err := cluster.UnmarshalV3(); err == nil {
		s.specClusters[clusterV3.Name] = cluster
	}
}

func (s *OptimizedStore) GetCluster(name helpers.NamespacedName) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cluster := s.clusters[name]
	return cluster
}

// resourceResult holds the results of concurrent resource loading
type resourceResult struct {
	accessLogs  []v1alpha1.AccessLogConfig
	clusters    []v1alpha1.Cluster
	listeners   []v1alpha1.Listener
	routes      []v1alpha1.Route
	vs          []v1alpha1.VirtualService
	vst         []v1alpha1.VirtualServiceTemplate
	httpFilters []v1alpha1.HttpFilter
	policies    []v1alpha1.Policy
	tracings    []v1alpha1.Tracing
	secrets     []corev1.Secret
	err         error
}

// loadResourcesConcurrently loads all resources from Kubernetes in parallel
func (s *OptimizedStore) loadResourcesConcurrently(ctx context.Context, cl client.Client) (*resourceResult, error) {
	resultChan := make(chan resourceResult, 10)

	// Start 10 concurrent loaders
	go s.loadVirtualServices(ctx, cl, resultChan)
	go s.loadListeners(ctx, cl, resultChan)
	go s.loadClusters(ctx, cl, resultChan)
	go s.loadAccessLogs(ctx, cl, resultChan)
	go s.loadRoutes(ctx, cl, resultChan)
	go s.loadVirtualServiceTemplates(ctx, cl, resultChan)
	go s.loadHttpFilters(ctx, cl, resultChan)
	go s.loadPolicies(ctx, cl, resultChan)
	go s.loadTracings(ctx, cl, resultChan)
	go s.loadSecrets(ctx, cl, resultChan)

	// Collect results
	aggregated := &resourceResult{}
	for i := 0; i < 10; i++ {
		result := <-resultChan
		if result.err != nil {
			return nil, result.err
		}
		if len(result.accessLogs) > 0 {
			aggregated.accessLogs = result.accessLogs
		}
		if len(result.vs) > 0 {
			aggregated.vs = result.vs
		}
		if len(result.listeners) > 0 {
			aggregated.listeners = result.listeners
		}
		if len(result.clusters) > 0 {
			aggregated.clusters = result.clusters
		}
		if len(result.routes) > 0 {
			aggregated.routes = result.routes
		}
		if len(result.vst) > 0 {
			aggregated.vst = result.vst
		}
		if len(result.httpFilters) > 0 {
			aggregated.httpFilters = result.httpFilters
		}
		if len(result.policies) > 0 {
			aggregated.policies = result.policies
		}
		if len(result.tracings) > 0 {
			aggregated.tracings = result.tracings
		}
		if len(result.secrets) > 0 {
			aggregated.secrets = result.secrets
		}
	}
	return aggregated, nil
}

// Resource loader helper functions
func (s *OptimizedStore) loadVirtualServices(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.VirtualServiceList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{vs: list.Items}
}

func (s *OptimizedStore) loadListeners(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.ListenerList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{listeners: list.Items}
}

func (s *OptimizedStore) loadClusters(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.ClusterList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{clusters: list.Items}
}

func (s *OptimizedStore) loadAccessLogs(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.AccessLogConfigList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{accessLogs: list.Items}
}

func (s *OptimizedStore) loadRoutes(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.RouteList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{routes: list.Items}
}

func (s *OptimizedStore) loadVirtualServiceTemplates(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.VirtualServiceTemplateList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{vst: list.Items}
}

func (s *OptimizedStore) loadHttpFilters(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.HttpFilterList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{httpFilters: list.Items}
}

func (s *OptimizedStore) loadPolicies(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.PolicyList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{policies: list.Items}
}

func (s *OptimizedStore) loadTracings(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list v1alpha1.TracingList
	if err := cl.List(ctx, &list); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{tracings: list.Items}
}

func (s *OptimizedStore) loadSecrets(ctx context.Context, cl client.Client, ch chan<- resourceResult) {
	var list corev1.SecretList
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"envoy.kaasops.io/secret-type": "sds-cached",
		},
	}
	selector, _ := metav1.LabelSelectorAsSelector(&labelSelector)
	if err := cl.List(ctx, &list, &client.ListOptions{LabelSelector: selector}); err != nil {
		ch <- resourceResult{err: err}
		return
	}
	ch <- resourceResult{secrets: list.Items}
}

// FillFromKubernetes with optimized batch loading
func (s *OptimizedStore) FillFromKubernetes(ctx context.Context, cl client.Client) error {
	// Handle nil client for testing
	if cl == nil {
		return nil
	}

	// Load all resources concurrently
	aggregated, err := s.loadResourcesConcurrently(ctx, cl)
	if err != nil {
		return err
	}

	// Now write to store under lock
	s.mu.Lock()
	defer s.mu.Unlock()

	// Process AccessLogConfigs
	for i := range aggregated.accessLogs {
		accessLog := &aggregated.accessLogs[i]
		accessLog.Name = s.stringPool.Intern(accessLog.Name)
		accessLog.Namespace = s.stringPool.InternNamespace(accessLog.Namespace)
		uid := s.stringPool.InternUID(string(accessLog.UID))

		key := helpers.NamespacedName{Namespace: accessLog.Namespace, Name: accessLog.Name}
		s.accessLogs[key] = accessLog
		s.accessLogsByUID[uid] = accessLog
	}

	// Process VirtualServices
	for i := range aggregated.vs {
		vs := &aggregated.vs[i]
		vs.NormalizeSpec() // Call NormalizeSpec like LegacyStore
		vs.Name = s.stringPool.Intern(vs.Name)
		vs.Namespace = s.stringPool.InternNamespace(vs.Namespace)
		uid := s.stringPool.InternUID(string(vs.UID))

		key := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}
		s.virtualServices[key] = vs
		s.uidToVS[uid] = vs
		s.updateTemplateIndex(vs)
	}

	// Process Listeners
	for i := range aggregated.listeners {
		listener := &aggregated.listeners[i]
		listener.Name = s.stringPool.Intern(listener.Name)
		listener.Namespace = s.stringPool.InternNamespace(listener.Namespace)
		uid := s.stringPool.InternUID(string(listener.UID))

		key := helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}
		s.listeners[key] = listener
		s.uidToListener[uid] = listener
	}

	// Process Clusters
	for i := range aggregated.clusters {
		cluster := &aggregated.clusters[i]
		cluster.Name = s.stringPool.Intern(cluster.Name)
		cluster.Namespace = s.stringPool.InternNamespace(cluster.Namespace)
		uid := s.stringPool.InternUID(string(cluster.UID))

		key := helpers.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}
		s.clusters[key] = cluster
		s.uidToCluster[uid] = cluster

		// Use UnmarshalV3().Name for specClusters index (like LegacyStore)
		if clusterV3, err := cluster.UnmarshalV3(); err == nil {
			s.specClusters[clusterV3.Name] = cluster
		}
	}

	// Process Routes
	for i := range aggregated.routes {
		route := &aggregated.routes[i]
		route.Name = s.stringPool.Intern(route.Name)
		route.Namespace = s.stringPool.InternNamespace(route.Namespace)
		uid := s.stringPool.InternUID(string(route.UID))

		key := helpers.NamespacedName{Namespace: route.Namespace, Name: route.Name}
		s.routes[key] = route
		s.routesByUID[uid] = route
	}

	// Process VirtualServiceTemplates
	for i := range aggregated.vst {
		vst := &aggregated.vst[i]
		vst.NormalizeSpec() // Call NormalizeSpec like LegacyStore
		vst.Name = s.stringPool.Intern(vst.Name)
		vst.Namespace = s.stringPool.InternNamespace(vst.Namespace)
		uid := s.stringPool.InternUID(string(vst.UID))

		key := helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}
		s.virtualServiceTemplates[key] = vst
		s.virtualServiceTemplatesByUID[uid] = vst
	}

	// Process HttpFilters
	for i := range aggregated.httpFilters {
		httpFilter := &aggregated.httpFilters[i]
		httpFilter.Name = s.stringPool.Intern(httpFilter.Name)
		httpFilter.Namespace = s.stringPool.InternNamespace(httpFilter.Namespace)
		uid := s.stringPool.InternUID(string(httpFilter.UID))

		key := helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name}
		s.httpFilters[key] = httpFilter
		s.httpFiltersByUID[uid] = httpFilter
	}

	// Process Policies
	for i := range aggregated.policies {
		policy := &aggregated.policies[i]
		policy.Name = s.stringPool.Intern(policy.Name)
		policy.Namespace = s.stringPool.InternNamespace(policy.Namespace)
		uid := s.stringPool.InternUID(string(policy.UID))

		key := helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}
		s.policies[key] = policy
		s.policiesByUID[uid] = policy
	}

	// Process Tracings
	for i := range aggregated.tracings {
		tracing := &aggregated.tracings[i]
		tracing.Name = s.stringPool.Intern(tracing.Name)
		tracing.Namespace = s.stringPool.InternNamespace(tracing.Namespace)
		uid := s.stringPool.InternUID(string(tracing.UID))

		key := helpers.NamespacedName{Namespace: tracing.Namespace, Name: tracing.Name}
		s.tracings[key] = tracing
		s.tracingsByUID[uid] = tracing
	}

	// Process Secrets
	for i := range aggregated.secrets {
		secret := &aggregated.secrets[i]
		secret.Name = s.stringPool.Intern(secret.Name)
		secret.Namespace = s.stringPool.InternNamespace(secret.Namespace)

		key := helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}
		s.secrets[key] = secret
	}

	// Update domain secrets map after all secrets are loaded
	s.updateDomainSecretsMap()

	return nil
}

// Performance monitoring
func (s *OptimizedStore) GetStringPoolStats() StringPoolStats {
	return s.stringPool.GetStats()
}

// Implement remaining interface methods
func (s *OptimizedStore) DeleteListener(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if listener := s.listeners[name]; listener != nil {
		delete(s.listeners, name)
		delete(s.uidToListener, string(listener.UID))
	}
}

func (s *OptimizedStore) DeleteCluster(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cluster := s.clusters[name]; cluster != nil {
		delete(s.clusters, name)
		delete(s.uidToCluster, string(cluster.UID))
		// Use UnmarshalV3().Name to find the correct specClusters key
		if clusterV3, err := cluster.UnmarshalV3(); err == nil {
			delete(s.specClusters, clusterV3.Name)
		}
	}
}
func (s *OptimizedStore) MapVirtualServices() map[helpers.NamespacedName]*v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[helpers.NamespacedName]*v1alpha1.VirtualService, len(s.virtualServices))
	for k, v := range s.virtualServices {
		result[k] = v
	}
	return result
}

// VirtualServiceTemplate operations
func (s *OptimizedStore) SetVirtualServiceTemplate(vst *v1alpha1.VirtualServiceTemplate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vst.Name = s.stringPool.Intern(vst.Name)
	vst.Namespace = s.stringPool.InternNamespace(vst.Namespace)
	uid := s.stringPool.InternUID(string(vst.UID))

	key := helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}

	if old := s.virtualServiceTemplates[key]; old != nil {
		delete(s.virtualServiceTemplatesByUID, string(old.UID))
	}

	s.virtualServiceTemplates[key] = vst
	s.virtualServiceTemplatesByUID[uid] = vst
}

func (s *OptimizedStore) GetVirtualServiceTemplate(name helpers.NamespacedName) *v1alpha1.VirtualServiceTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vst := s.virtualServiceTemplates[name]
	return vst
}

func (s *OptimizedStore) DeleteVirtualServiceTemplate(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if vst := s.virtualServiceTemplates[name]; vst != nil {
		delete(s.virtualServiceTemplates, name)
		delete(s.virtualServiceTemplatesByUID, string(vst.UID))
	}
}

// Route operations
func (s *OptimizedStore) SetRoute(route *v1alpha1.Route) {
	s.mu.Lock()
	defer s.mu.Unlock()

	route.Name = s.stringPool.Intern(route.Name)
	route.Namespace = s.stringPool.InternNamespace(route.Namespace)
	uid := s.stringPool.InternUID(string(route.UID))

	key := helpers.NamespacedName{Namespace: route.Namespace, Name: route.Name}

	if old := s.routes[key]; old != nil {
		delete(s.routesByUID, string(old.UID))
	}

	s.routes[key] = route
	s.routesByUID[uid] = route
}

func (s *OptimizedStore) GetRoute(name helpers.NamespacedName) *v1alpha1.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()

	route := s.routes[name]
	return route
}

func (s *OptimizedStore) DeleteRoute(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if route := s.routes[name]; route != nil {
		delete(s.routes, name)
		delete(s.routesByUID, string(route.UID))
	}
}

// HTTPFilter operations
func (s *OptimizedStore) SetHTTPFilter(filter *v1alpha1.HttpFilter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filter.Name = s.stringPool.Intern(filter.Name)
	filter.Namespace = s.stringPool.InternNamespace(filter.Namespace)
	uid := s.stringPool.InternUID(string(filter.UID))

	key := helpers.NamespacedName{Namespace: filter.Namespace, Name: filter.Name}

	if old := s.httpFilters[key]; old != nil {
		delete(s.httpFiltersByUID, string(old.UID))
	}

	s.httpFilters[key] = filter
	s.httpFiltersByUID[uid] = filter
}

func (s *OptimizedStore) GetHTTPFilter(name helpers.NamespacedName) *v1alpha1.HttpFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter := s.httpFilters[name]
	return filter
}

func (s *OptimizedStore) DeleteHTTPFilter(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if filter := s.httpFilters[name]; filter != nil {
		delete(s.httpFilters, name)
		delete(s.httpFiltersByUID, string(filter.UID))
	}
}

// AccessLog operations
func (s *OptimizedStore) SetAccessLog(accessLog *v1alpha1.AccessLogConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accessLog.Name = s.stringPool.Intern(accessLog.Name)
	accessLog.Namespace = s.stringPool.InternNamespace(accessLog.Namespace)
	uid := s.stringPool.InternUID(string(accessLog.UID))

	key := helpers.NamespacedName{Namespace: accessLog.Namespace, Name: accessLog.Name}

	if old := s.accessLogs[key]; old != nil {
		delete(s.accessLogsByUID, string(old.UID))
	}

	s.accessLogs[key] = accessLog
	s.accessLogsByUID[uid] = accessLog
}

func (s *OptimizedStore) GetAccessLog(name helpers.NamespacedName) *v1alpha1.AccessLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accessLog := s.accessLogs[name]
	return accessLog
}

func (s *OptimizedStore) DeleteAccessLog(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if accessLog := s.accessLogs[name]; accessLog != nil {
		delete(s.accessLogs, name)
		delete(s.accessLogsByUID, string(accessLog.UID))
	}
}

// Policy operations
func (s *OptimizedStore) SetPolicy(policy *v1alpha1.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy.Name = s.stringPool.Intern(policy.Name)
	policy.Namespace = s.stringPool.InternNamespace(policy.Namespace)
	uid := s.stringPool.InternUID(string(policy.UID))

	key := helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}

	if old := s.policies[key]; old != nil {
		delete(s.policiesByUID, string(old.UID))
	}

	s.policies[key] = policy
	s.policiesByUID[uid] = policy
}

func (s *OptimizedStore) GetPolicy(name helpers.NamespacedName) *v1alpha1.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy := s.policies[name]
	return policy
}

func (s *OptimizedStore) DeletePolicy(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if policy := s.policies[name]; policy != nil {
		delete(s.policies, name)
		delete(s.policiesByUID, string(policy.UID))
	}
}

// SetSecret adds or updates a secret in the store
// WARNING: This method mutates the input secret (name/namespace interning)
func (s *OptimizedStore) SetSecret(secret *corev1.Secret) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret.Name = s.stringPool.Intern(secret.Name)
	secret.Namespace = s.stringPool.InternNamespace(secret.Namespace)

	key := helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}

	// Remove old secret's domains from index if exists
	if oldSecret := s.secrets[key]; oldSecret != nil {
		s.removeDomainSecretsForSecret(oldSecret)
	}

	s.secrets[key] = secret

	// Add new secret's domains to index
	s.addDomainSecretsForSecret(secret)
}

func (s *OptimizedStore) GetSecret(name helpers.NamespacedName) *corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret := s.secrets[name]
	return secret
}

func (s *OptimizedStore) DeleteSecret(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if secret := s.secrets[name]; secret != nil {
		s.removeDomainSecretsForSecret(secret)
		delete(s.secrets, name)
	}
}

// Tracing operations
func (s *OptimizedStore) SetTracing(tracing *v1alpha1.Tracing) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracing.Name = s.stringPool.Intern(tracing.Name)
	tracing.Namespace = s.stringPool.InternNamespace(tracing.Namespace)
	uid := s.stringPool.InternUID(string(tracing.UID))

	key := helpers.NamespacedName{Namespace: tracing.Namespace, Name: tracing.Name}

	if old := s.tracings[key]; old != nil {
		delete(s.tracingsByUID, string(old.UID))
	}

	s.tracings[key] = tracing
	s.tracingsByUID[uid] = tracing
}

func (s *OptimizedStore) GetTracing(name helpers.NamespacedName) *v1alpha1.Tracing {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tracing := s.tracings[name]
	return tracing
}

func (s *OptimizedStore) DeleteTracing(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tracing := s.tracings[name]; tracing != nil {
		delete(s.tracings, name)
		delete(s.tracingsByUID, string(tracing.UID))
	}
}

// IsExisting methods
func (s *OptimizedStore) IsExistingVirtualService(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.virtualServices[name]
	return exists
}

func (s *OptimizedStore) IsExistingVirtualServiceTemplate(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.virtualServiceTemplates[name]
	return exists
}

func (s *OptimizedStore) IsExistingListener(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.listeners[name]
	return exists
}

func (s *OptimizedStore) IsExistingRoute(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.routes[name]
	return exists
}

func (s *OptimizedStore) IsExistingCluster(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.clusters[name]
	return exists
}

func (s *OptimizedStore) IsExistingHTTPFilter(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.httpFilters[name]
	return exists
}

func (s *OptimizedStore) IsExistingAccessLog(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.accessLogs[name]
	return exists
}

func (s *OptimizedStore) IsExistingPolicy(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.policies[name]
	return exists
}

func (s *OptimizedStore) IsExistingSecret(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.secrets[name]
	return exists
}

func (s *OptimizedStore) IsExistingTracing(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.tracings[name]
	return exists
}

// Map methods
func (s *OptimizedStore) MapVirtualServiceTemplates() map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate, len(s.virtualServiceTemplates))
	for k, v := range s.virtualServiceTemplates {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapListeners() map[helpers.NamespacedName]*v1alpha1.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.Listener, len(s.listeners))
	for k, v := range s.listeners {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapRoutes() map[helpers.NamespacedName]*v1alpha1.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.Route, len(s.routes))
	for k, v := range s.routes {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapClusters() map[helpers.NamespacedName]*v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.Cluster, len(s.clusters))
	for k, v := range s.clusters {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapHTTPFilters() map[helpers.NamespacedName]*v1alpha1.HttpFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.HttpFilter, len(s.httpFilters))
	for k, v := range s.httpFilters {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapAccessLogs() map[helpers.NamespacedName]*v1alpha1.AccessLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig, len(s.accessLogs))
	for k, v := range s.accessLogs {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapPolicies() map[helpers.NamespacedName]*v1alpha1.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.Policy, len(s.policies))
	for k, v := range s.policies {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapSecrets() map[helpers.NamespacedName]*corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*corev1.Secret, len(s.secrets))
	for k, v := range s.secrets {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapTracings() map[helpers.NamespacedName]*v1alpha1.Tracing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[helpers.NamespacedName]*v1alpha1.Tracing, len(s.tracings))
	for k, v := range s.tracings {
		result[k] = v
	}
	return result
}

// ByUID methods
func (s *OptimizedStore) GetVirtualServiceTemplateByUID(uid string) *v1alpha1.VirtualServiceTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.virtualServiceTemplatesByUID[uid]
}

func (s *OptimizedStore) GetListenerByUID(uid string) *v1alpha1.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.uidToListener[uid]
}

func (s *OptimizedStore) GetRouteByUID(uid string) *v1alpha1.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routesByUID[uid]
}

func (s *OptimizedStore) GetClusterByUID(uid string) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.uidToCluster[uid]
}

func (s *OptimizedStore) GetHTTPFilterByUID(uid string) *v1alpha1.HttpFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.httpFiltersByUID[uid]
}

func (s *OptimizedStore) GetAccessLogByUID(uid string) *v1alpha1.AccessLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessLogsByUID[uid]
}

// GetDomainSecret returns the secret for a given domain
// Returns zero value if not found (check with secret.Name != "")
func (s *OptimizedStore) GetDomainSecret(domain string) corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.domainSecrets[domain]
}

func (s *OptimizedStore) GetSpecCluster(name string) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.specClusters[name]
}

// GetVirtualServicesByTemplate returns a copy of VirtualServices using the given template
// Returns nil if no VirtualServices use this template
func (s *OptimizedStore) GetVirtualServicesByTemplate(templateName helpers.NamespacedName) []*v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vsList := s.templateToVS[templateName]
	if vsList == nil {
		return nil
	}

	// Return a copy to prevent external modifications
	return append([]*v1alpha1.VirtualService(nil), vsList...)
}

// Additional required methods for Store interface
func (s *OptimizedStore) MapSpecClusters() map[string]*v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*v1alpha1.Cluster, len(s.specClusters))
	for k, v := range s.specClusters {
		result[k] = v
	}
	return result
}

func (s *OptimizedStore) MapDomainSecrets() map[string]*corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*corev1.Secret, len(s.domainSecrets))
	for k, v := range s.domainSecrets {
		secret := v // Create explicit copy for pointer safety
		result[k] = &secret
	}
	return result
}

// Node domains index methods
func (s *OptimizedStore) ReplaceNodeDomainsIndex(idx map[string]map[string]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy the provided index
	s.nodeDomains = make(map[string]map[string]struct{})
	for nodeID, domains := range idx {
		s.nodeDomains[nodeID] = make(map[string]struct{})
		for domain := range domains {
			s.nodeDomains[nodeID][domain] = struct{}{}
		}
	}
}

func (s *OptimizedStore) GetNodeDomainsIndex() map[string]map[string]struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy to prevent external modifications
	result := make(map[string]map[string]struct{})
	for nodeID, domains := range s.nodeDomains {
		result[nodeID] = make(map[string]struct{})
		for domain := range domains {
			result[nodeID][domain] = struct{}{}
		}
	}
	return result
}

func (s *OptimizedStore) GetNodeDomainsForNodes(nodeIDs []string) (map[string]map[string]struct{}, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	foundDomains := make(map[string]map[string]struct{})
	var missingNodes []string

	for _, nodeID := range nodeIDs {
		if domains, exists := s.nodeDomains[nodeID]; exists {
			// Deep copy the domains
			foundDomains[nodeID] = make(map[string]struct{})
			for domain := range domains {
				foundDomains[nodeID][domain] = struct{}{}
			}
		} else {
			missingNodes = append(missingNodes, nodeID)
		}
	}

	return foundDomains, missingNodes
}

func (s *OptimizedStore) GetVirtualServicesByTemplateNN(nn helpers.NamespacedName) []*v1alpha1.VirtualService {
	return s.GetVirtualServicesByTemplate(nn)
}

// addDomainSecretsForSecret adds domains from a secret to the domain index
func (s *OptimizedStore) addDomainSecretsForSecret(secret *corev1.Secret) {
	if secret.Annotations == nil {
		return
	}
	domainsAnnotation, exists := secret.Annotations[v1alpha1.AnnotationSecretDomains]
	if !exists {
		return
	}

	for _, domain := range strings.Split(domainsAnnotation, ",") {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if _, exists := s.domainSecrets[domain]; exists {
			// TODO: domain already exists in another secret! Need to create error case
			continue
		}
		s.domainSecrets[domain] = *secret
	}
}

// removeDomainSecretsForSecret removes domains from a secret from the domain index
func (s *OptimizedStore) removeDomainSecretsForSecret(secret *corev1.Secret) {
	if secret.Annotations == nil {
		return
	}
	domainsAnnotation, exists := secret.Annotations[v1alpha1.AnnotationSecretDomains]
	if !exists {
		return
	}

	for _, domain := range strings.Split(domainsAnnotation, ",") {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		delete(s.domainSecrets, domain)
	}
}

// updateDomainSecretsMap rebuilds the entire domain to secret mapping from all secrets
// This is used only during FillFromKubernetes for initial population
func (s *OptimizedStore) updateDomainSecretsMap() {
	m := make(map[string]corev1.Secret)

	for _, secret := range s.secrets {
		if secret.Annotations == nil {
			continue
		}
		domainsAnnotation, exists := secret.Annotations[v1alpha1.AnnotationSecretDomains]
		if !exists {
			continue
		}

		for _, domain := range strings.Split(domainsAnnotation, ",") {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			if _, ok := m[domain]; ok {
				// TODO: domain already exists in another secret! Need to create error case
				continue
			}
			m[domain] = *secret
		}
	}
	s.domainSecrets = m
}
