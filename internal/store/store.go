package store

import (
	"context"
	"sync"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Store struct {
	mu                          sync.RWMutex
	virtualServices             map[helpers.NamespacedName]*v1alpha1.VirtualService
	virtualServiceByUID         map[string]*v1alpha1.VirtualService
	virtualServiceTemplates     map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate
	virtualServiceTemplateByUID map[string]*v1alpha1.VirtualServiceTemplate
	routes                      map[helpers.NamespacedName]*v1alpha1.Route
	routeByUID                  map[string]*v1alpha1.Route
	clusters                    map[helpers.NamespacedName]*v1alpha1.Cluster
	clusterByUID                map[string]*v1alpha1.Cluster
	specClusters                map[string]*v1alpha1.Cluster
	httpFilters                 map[helpers.NamespacedName]*v1alpha1.HttpFilter
	httpFilterByUID             map[string]*v1alpha1.HttpFilter
	listeners                   map[helpers.NamespacedName]*v1alpha1.Listener
	listenerByUID               map[string]*v1alpha1.Listener
	accessLogs                  map[helpers.NamespacedName]*v1alpha1.AccessLogConfig
	accessLogByUID              map[string]*v1alpha1.AccessLogConfig
	policies                    map[helpers.NamespacedName]*v1alpha1.Policy
	secrets                     map[helpers.NamespacedName]*v1.Secret
	tracings                    map[helpers.NamespacedName]*v1alpha1.Tracing
	domainToSecretMap           map[string]v1.Secret

	// Indices (optional, used by light validators when enabled)
	listenerAddrIndex map[string]string              // host:port -> listener namespaced name string
	listenerAddrDup   *listenerAddrDup               // first detected duplicate, if any
	nodeDomainsIndex  map[string]map[string]struct{} // nodeID -> set(domains)
}

func New() *Store {
	store := &Store{
		accessLogs:                  make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig),
		accessLogByUID:              make(map[string]*v1alpha1.AccessLogConfig),
		virtualServices:             make(map[helpers.NamespacedName]*v1alpha1.VirtualService),
		virtualServiceTemplates:     make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate),
		virtualServiceTemplateByUID: make(map[string]*v1alpha1.VirtualServiceTemplate),
		routes:                      make(map[helpers.NamespacedName]*v1alpha1.Route),
		routeByUID:                  make(map[string]*v1alpha1.Route),
		clusters:                    make(map[helpers.NamespacedName]*v1alpha1.Cluster),
		clusterByUID:                make(map[string]*v1alpha1.Cluster),
		httpFilters:                 make(map[helpers.NamespacedName]*v1alpha1.HttpFilter),
		httpFilterByUID:             make(map[string]*v1alpha1.HttpFilter),
		listeners:                   make(map[helpers.NamespacedName]*v1alpha1.Listener),
		listenerByUID:               make(map[string]*v1alpha1.Listener),
		policies:                    make(map[helpers.NamespacedName]*v1alpha1.Policy),
		secrets:                     make(map[helpers.NamespacedName]*v1.Secret),
		domainToSecretMap:           make(map[string]v1.Secret),
		specClusters:                make(map[string]*v1alpha1.Cluster),
		tracings:                    make(map[helpers.NamespacedName]*v1alpha1.Tracing),
	}
	return store
}

// Copy creates copy of the Store
func (s *Store) Copy() *Store {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &Store{
		accessLogs:                  make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig, len(s.accessLogs)),
		accessLogByUID:              make(map[string]*v1alpha1.AccessLogConfig, len(s.accessLogByUID)),
		virtualServices:             make(map[helpers.NamespacedName]*v1alpha1.VirtualService, len(s.virtualServices)),
		virtualServiceTemplates:     make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate, len(s.virtualServiceTemplates)),
		virtualServiceTemplateByUID: make(map[string]*v1alpha1.VirtualServiceTemplate, len(s.virtualServiceTemplateByUID)),
		routes:                      make(map[helpers.NamespacedName]*v1alpha1.Route, len(s.routes)),
		routeByUID:                  make(map[string]*v1alpha1.Route, len(s.routeByUID)),
		clusters:                    make(map[helpers.NamespacedName]*v1alpha1.Cluster, len(s.clusters)),
		clusterByUID:                make(map[string]*v1alpha1.Cluster, len(s.clusterByUID)),
		httpFilters:                 make(map[helpers.NamespacedName]*v1alpha1.HttpFilter, len(s.httpFilters)),
		httpFilterByUID:             make(map[string]*v1alpha1.HttpFilter, len(s.httpFilterByUID)),
		listeners:                   make(map[helpers.NamespacedName]*v1alpha1.Listener, len(s.listeners)),
		listenerByUID:               make(map[string]*v1alpha1.Listener, len(s.listenerByUID)),
		policies:                    make(map[helpers.NamespacedName]*v1alpha1.Policy, len(s.policies)),
		secrets:                     make(map[helpers.NamespacedName]*v1.Secret, len(s.secrets)),
		domainToSecretMap:           make(map[string]v1.Secret, len(s.domainToSecretMap)),
		specClusters:                make(map[string]*v1alpha1.Cluster, len(s.specClusters)),
		tracings:                    make(map[helpers.NamespacedName]*v1alpha1.Tracing, len(s.tracings)),
	}

	for k, v := range s.virtualServices {
		clone.virtualServices[k] = v.DeepCopy()
	}
	for k, v := range s.virtualServiceTemplates {
		clone.virtualServiceTemplates[k] = v.DeepCopy()
	}
	for k, v := range s.routes {
		clone.routes[k] = v
	}
	for k, v := range s.clusters {
		clone.clusters[k] = v
	}
	for k, v := range s.httpFilters {
		clone.httpFilters[k] = v
	}
	for k, v := range s.listeners {
		clone.listeners[k] = v
	}
	for k, v := range s.accessLogs {
		clone.accessLogs[k] = v
	}
	for k, v := range s.policies {
		clone.policies[k] = v
	}
	for k, v := range s.secrets {
		clone.secrets[k] = v
	}
	for k, v := range s.tracings {
		clone.tracings[k] = v
	}

	// Deep copy nodeDomainsIndex (if present)
	if s.nodeDomainsIndex != nil {
		clone.nodeDomainsIndex = make(map[string]map[string]struct{}, len(s.nodeDomainsIndex))
		for node, set := range s.nodeDomainsIndex {
			inner := make(map[string]struct{}, len(set))
			for d := range set {
				inner[d] = struct{}{}
			}
			clone.nodeDomainsIndex[node] = inner
		}
	}

	clone.updateListenerByUIDMap()
	clone.updateVirtualServiceByUIDMap()
	clone.updateVirtualServiceTemplateByUIDMap()
	clone.updateRouteByUIDMap()
	clone.updateClusterByUIDMap()
	clone.updateAccessLogByUIDMap()
	clone.updateHTTPFilterByUIDMap()
	clone.updateDomainSecretsMap()
	clone.updateSpecClusters()
	clone.updateListenerAddressIndex()

	return clone
}

func (s *Store) FillFromKubernetes(ctx context.Context, cl client.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var accessLogConfigs v1alpha1.AccessLogConfigList
	if err := cl.List(ctx, &accessLogConfigs); err != nil {
		return err
	}
	var clusters v1alpha1.ClusterList
	if err := cl.List(ctx, &clusters); err != nil {
		return err
	}
	var listeners v1alpha1.ListenerList
	if err := cl.List(ctx, &listeners); err != nil {
		return err
	}
	var routes v1alpha1.RouteList
	if err := cl.List(ctx, &routes); err != nil {
		return err
	}
	var virtualServices v1alpha1.VirtualServiceList
	if err := cl.List(ctx, &virtualServices); err != nil {
		return err
	}
	var virtualServiceTemplates v1alpha1.VirtualServiceTemplateList
	if err := cl.List(ctx, &virtualServiceTemplates); err != nil {
		return err
	}
	var httpFilters v1alpha1.HttpFilterList
	if err := cl.List(ctx, &httpFilters); err != nil {
		return err
	}
	var policies v1alpha1.PolicyList
	if err := cl.List(ctx, &policies); err != nil {
		return err
	}
	var tracings v1alpha1.TracingList
	if err := cl.List(ctx, &tracings); err != nil {
		return err
	}

	var secrets v1.SecretList
	requirement, err := labels.NewRequirement("envoy.kaasops.io/secret-type", "==", []string{"sds-cached"})
	if err != nil {
		return err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	if err := cl.List(ctx, &secrets, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		return err
	}

	s.virtualServices = make(map[helpers.NamespacedName]*v1alpha1.VirtualService, len(virtualServices.Items))
	s.virtualServiceTemplates = make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate, len(virtualServiceTemplates.Items))
	s.routes = make(map[helpers.NamespacedName]*v1alpha1.Route, len(routes.Items))
	s.clusters = make(map[helpers.NamespacedName]*v1alpha1.Cluster, len(clusters.Items))
	s.httpFilters = make(map[helpers.NamespacedName]*v1alpha1.HttpFilter, len(httpFilters.Items))
	s.listeners = make(map[helpers.NamespacedName]*v1alpha1.Listener, len(listeners.Items))
	s.accessLogs = make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig, len(accessLogConfigs.Items))
	s.policies = make(map[helpers.NamespacedName]*v1alpha1.Policy, len(policies.Items))
	s.secrets = make(map[helpers.NamespacedName]*v1.Secret, len(secrets.Items))
	s.domainToSecretMap = make(map[string]v1.Secret, len(secrets.Items))
	s.specClusters = make(map[string]*v1alpha1.Cluster, len(clusters.Items))
	s.tracings = make(map[helpers.NamespacedName]*v1alpha1.Tracing, len(tracings.Items))

	for _, vs := range virtualServices.Items {
		vs.NormalizeSpec()
		s.virtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}] = &vs
	}
	for _, vst := range virtualServiceTemplates.Items {
		vst.NormalizeSpec()
		s.virtualServiceTemplates[helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}] = &vst
	}
	for _, route := range routes.Items {
		s.routes[helpers.NamespacedName{Namespace: route.Namespace, Name: route.Name}] = &route
	}
	for _, cluster := range clusters.Items {
		s.clusters[helpers.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}] = &cluster
	}
	for _, httpFilter := range httpFilters.Items {
		s.httpFilters[helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name}] = &httpFilter
	}
	for _, listener := range listeners.Items {
		s.listeners[helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}] = &listener
	}
	for _, accessLogConfig := range accessLogConfigs.Items {
		s.accessLogs[helpers.NamespacedName{Namespace: accessLogConfig.Namespace, Name: accessLogConfig.Name}] = &accessLogConfig
	}
	for _, policy := range policies.Items {
		s.policies[helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = &policy
	}
	for _, secret := range secrets.Items {
		s.secrets[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = &secret
	}
	for _, tracing := range tracings.Items {
		s.tracings[helpers.NamespacedName{Namespace: tracing.Namespace, Name: tracing.Name}] = &tracing
	}
	s.updateListenerByUIDMap()
	s.updateVirtualServiceByUIDMap()
	s.updateVirtualServiceTemplateByUIDMap()
	s.updateRouteByUIDMap()
	s.updateClusterByUIDMap()
	s.updateAccessLogByUIDMap()
	s.updateHTTPFilterByUIDMap()
	s.updateDomainSecretsMap()
	s.updateSpecClusters()
	s.updateListenerAddressIndex()
	return nil
}
