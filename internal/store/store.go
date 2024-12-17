package store

import (
	"context"
	"strings"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Store struct {
	VirtualServices         map[helpers.NamespacedName]*v1alpha1.VirtualService
	VirtualServiceTemplates map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate
	Routes                  map[helpers.NamespacedName]*v1alpha1.Route
	Clusters                map[helpers.NamespacedName]*v1alpha1.Cluster
	SpecClusters            map[string]*v1alpha1.Cluster
	HTTPFilters             map[helpers.NamespacedName]*v1alpha1.HttpFilter
	Listeners               map[helpers.NamespacedName]*v1alpha1.Listener
	AccessLogs              map[helpers.NamespacedName]*v1alpha1.AccessLogConfig
	Policies                map[helpers.NamespacedName]*v1alpha1.Policy
	DomainToSecretMap       map[string]v1.Secret
	Secrets                 map[helpers.NamespacedName]*v1.Secret
}

func New() *Store {
	store := &Store{
		AccessLogs:              make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig),
		VirtualServices:         make(map[helpers.NamespacedName]*v1alpha1.VirtualService),
		VirtualServiceTemplates: make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate),
		Routes:                  make(map[helpers.NamespacedName]*v1alpha1.Route),
		Clusters:                make(map[helpers.NamespacedName]*v1alpha1.Cluster),
		HTTPFilters:             make(map[helpers.NamespacedName]*v1alpha1.HttpFilter),
		Listeners:               make(map[helpers.NamespacedName]*v1alpha1.Listener),
		Policies:                make(map[helpers.NamespacedName]*v1alpha1.Policy),
		Secrets:                 make(map[helpers.NamespacedName]*v1.Secret),
	}
	store.UpdateDomainSecretsMap()
	store.UpdateSpecClusters()
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

	s.VirtualServices = make(map[helpers.NamespacedName]*v1alpha1.VirtualService, len(virtualServices.Items))
	s.VirtualServiceTemplates = make(map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate, len(virtualServiceTemplates.Items))
	s.Routes = make(map[helpers.NamespacedName]*v1alpha1.Route, len(routes.Items))
	s.Clusters = make(map[helpers.NamespacedName]*v1alpha1.Cluster, len(clusters.Items))
	s.HTTPFilters = make(map[helpers.NamespacedName]*v1alpha1.HttpFilter, len(httpFilters.Items))
	s.Listeners = make(map[helpers.NamespacedName]*v1alpha1.Listener, len(listeners.Items))
	s.AccessLogs = make(map[helpers.NamespacedName]*v1alpha1.AccessLogConfig, len(accessLogConfigs.Items))
	s.Policies = make(map[helpers.NamespacedName]*v1alpha1.Policy, len(policies.Items))
	s.Secrets = make(map[helpers.NamespacedName]*v1.Secret, len(secrets.Items))
	s.DomainToSecretMap = make(map[string]v1.Secret, len(secrets.Items))
	s.SpecClusters = make(map[string]*v1alpha1.Cluster, len(clusters.Items))

	for _, vs := range virtualServices.Items {
		s.VirtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}] = &vs
	}
	for _, vst := range virtualServiceTemplates.Items {
		s.VirtualServiceTemplates[helpers.NamespacedName{Namespace: vst.Namespace, Name: vst.Name}] = &vst
	}
	for _, route := range routes.Items {
		s.Routes[helpers.NamespacedName{Namespace: route.Namespace, Name: route.Name}] = &route
	}
	for _, cluster := range clusters.Items {
		s.Clusters[helpers.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}] = &cluster
	}
	s.UpdateSpecClusters()
	for _, httpFilter := range httpFilters.Items {
		s.HTTPFilters[helpers.NamespacedName{Namespace: httpFilter.Namespace, Name: httpFilter.Name}] = &httpFilter
	}
	for _, listener := range listeners.Items {
		s.Listeners[helpers.NamespacedName{Namespace: listener.Namespace, Name: listener.Name}] = &listener
	}
	for _, accessLogConfig := range accessLogConfigs.Items {
		s.AccessLogs[helpers.NamespacedName{Namespace: accessLogConfig.Namespace, Name: accessLogConfig.Name}] = &accessLogConfig
	}
	for _, policy := range policies.Items {
		s.Policies[helpers.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = &policy
	}
	for _, secret := range secrets.Items {
		s.Secrets[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = &secret
	}
	s.UpdateDomainSecretsMap()
	return err
}

func (s *Store) UpdateDomainSecretsMap() {
	m := make(map[string]v1.Secret)

	for _, secret := range s.Secrets {
		for _, domain := range strings.Split(secret.Annotations[v1alpha1.AnnotationSecretDomains], ",") {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			if _, ok := m[domain]; ok {
				// TODO domain already exist in another secret! Need create error case
				continue
			}
			m[domain] = *secret
		}
	}
	s.DomainToSecretMap = m
}

func (s *Store) UpdateSpecClusters() {
	m := make(map[string]*v1alpha1.Cluster)

	for _, cluster := range s.Clusters {
		clusterV3, _ := cluster.UnmarshalV3()
		m[clusterV3.Name] = cluster
	}

	s.SpecClusters = m
}
