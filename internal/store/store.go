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
	domainToSecretMap           map[string]v1.Secret
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
	}
	return store
}

func (s *Store) Fill(ctx context.Context, cl client.Client) error {
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

	for _, vs := range virtualServices.Items {
		s.virtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}] = &vs
	}
	for _, vst := range virtualServiceTemplates.Items {
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
	s.updateListenerByUIDMap()
	s.updateVirtualServiceByUIDMap()
	s.updateVirtualServiceTemplateByUIDMap()
	s.updateRouteByUIDMap()
	s.updateClusterByUIDMap()
	s.updateAccessLogByUIDMap()
	s.updateHTTPFilterByUIDMap()
	s.updateDomainSecretsMap()
	s.updateSpecClusters()
	return nil
}
