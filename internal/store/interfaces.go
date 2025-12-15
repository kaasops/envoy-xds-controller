package store

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Store is the main interface for the storage system matching LegacyStore methods
type Store interface {
	// Virtual Services
	GetVirtualService(name helpers.NamespacedName) *v1alpha1.VirtualService
	GetVirtualServiceByUID(uid string) *v1alpha1.VirtualService
	SetVirtualService(vs *v1alpha1.VirtualService)
	DeleteVirtualService(name helpers.NamespacedName)
	IsExistingVirtualService(name helpers.NamespacedName) bool
	MapVirtualServices() map[helpers.NamespacedName]*v1alpha1.VirtualService
	GetVirtualServicesByTemplateNN(nn helpers.NamespacedName) []*v1alpha1.VirtualService

	// Virtual Service Templates
	GetVirtualServiceTemplate(name helpers.NamespacedName) *v1alpha1.VirtualServiceTemplate
	GetVirtualServiceTemplateByUID(uid string) *v1alpha1.VirtualServiceTemplate
	SetVirtualServiceTemplate(vst *v1alpha1.VirtualServiceTemplate)
	DeleteVirtualServiceTemplate(name helpers.NamespacedName)
	IsExistingVirtualServiceTemplate(name helpers.NamespacedName) bool
	MapVirtualServiceTemplates() map[helpers.NamespacedName]*v1alpha1.VirtualServiceTemplate

	// Listeners
	GetListener(name helpers.NamespacedName) *v1alpha1.Listener
	GetListenerByUID(uid string) *v1alpha1.Listener
	SetListener(l *v1alpha1.Listener)
	DeleteListener(name helpers.NamespacedName)
	IsExistingListener(name helpers.NamespacedName) bool
	MapListeners() map[helpers.NamespacedName]*v1alpha1.Listener

	// Routes
	GetRoute(name helpers.NamespacedName) *v1alpha1.Route
	GetRouteByUID(uid string) *v1alpha1.Route
	SetRoute(r *v1alpha1.Route)
	DeleteRoute(name helpers.NamespacedName)
	IsExistingRoute(name helpers.NamespacedName) bool
	MapRoutes() map[helpers.NamespacedName]*v1alpha1.Route

	// Clusters
	GetCluster(name helpers.NamespacedName) *v1alpha1.Cluster
	SetCluster(c *v1alpha1.Cluster)
	DeleteCluster(name helpers.NamespacedName)
	IsExistingCluster(name helpers.NamespacedName) bool
	MapClusters() map[helpers.NamespacedName]*v1alpha1.Cluster
	GetSpecCluster(name string) *v1alpha1.Cluster
	MapSpecClusters() map[string]*v1alpha1.Cluster

	// HTTP Filters
	GetHTTPFilter(name helpers.NamespacedName) *v1alpha1.HttpFilter
	GetHTTPFilterByUID(uid string) *v1alpha1.HttpFilter
	SetHTTPFilter(hf *v1alpha1.HttpFilter)
	DeleteHTTPFilter(name helpers.NamespacedName)
	IsExistingHTTPFilter(name helpers.NamespacedName) bool
	MapHTTPFilters() map[helpers.NamespacedName]*v1alpha1.HttpFilter

	// Access Logs
	GetAccessLog(name helpers.NamespacedName) *v1alpha1.AccessLogConfig
	GetAccessLogByUID(uid string) *v1alpha1.AccessLogConfig
	SetAccessLog(a *v1alpha1.AccessLogConfig)
	DeleteAccessLog(name helpers.NamespacedName)
	IsExistingAccessLog(name helpers.NamespacedName) bool
	MapAccessLogs() map[helpers.NamespacedName]*v1alpha1.AccessLogConfig

	// Policies
	GetPolicy(name helpers.NamespacedName) *v1alpha1.Policy
	SetPolicy(p *v1alpha1.Policy)
	DeletePolicy(name helpers.NamespacedName)
	IsExistingPolicy(name helpers.NamespacedName) bool
	MapPolicies() map[helpers.NamespacedName]*v1alpha1.Policy

	// Secrets
	GetSecret(name helpers.NamespacedName) *corev1.Secret
	SetSecret(secret *corev1.Secret)
	DeleteSecret(name helpers.NamespacedName)
	IsExistingSecret(name helpers.NamespacedName) bool
	MapSecrets() map[helpers.NamespacedName]*corev1.Secret
	MapDomainSecrets() map[string]*corev1.Secret
	MapDomainSecretsForNamespace(preferredNamespace string) map[string]*corev1.Secret
	GetDomainSecretForNamespace(domain string, preferredNamespace string) *corev1.Secret

	// Tracing
	GetTracing(name helpers.NamespacedName) *v1alpha1.Tracing
	SetTracing(t *v1alpha1.Tracing)
	DeleteTracing(name helpers.NamespacedName)
	IsExistingTracing(name helpers.NamespacedName) bool
	MapTracings() map[helpers.NamespacedName]*v1alpha1.Tracing

	// Domain indices
	ReplaceNodeDomainsIndex(idx map[string]map[string]struct{})
	GetNodeDomainsIndex() map[string]map[string]struct{}
	GetNodeDomainsForNodes(nodeIDs []string) (map[string]map[string]struct{}, []string)

	// Snapshot operations - returns Store interface instead of concrete type
	Copy() Store

	// Fill from Kubernetes
	FillFromKubernetes(ctx context.Context, cl client.Client) error
}
