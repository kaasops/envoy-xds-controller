package main_builder

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
)

// Builder is responsible for coordinating the building of all resources for a VirtualService
type Builder struct {
	store            *store.Store
	httpFilterBuilder resbuilder_v2.HTTPFilterBuilder
	filterChainBuilder resbuilder_v2.FilterChainBuilder
	routingBuilder   resbuilder_v2.RoutingBuilder
	accessLogBuilder resbuilder_v2.AccessLogBuilder
	tlsBuilder       resbuilder_v2.TLSBuilder
	clusterExtractor resbuilder_v2.ClusterExtractor
}

// NewBuilder creates a new Builder with the provided dependencies
func NewBuilder(
	store *store.Store,
	httpFilterBuilder resbuilder_v2.HTTPFilterBuilder,
	filterChainBuilder resbuilder_v2.FilterChainBuilder,
	routingBuilder resbuilder_v2.RoutingBuilder,
	accessLogBuilder resbuilder_v2.AccessLogBuilder,
	tlsBuilder resbuilder_v2.TLSBuilder,
	clusterExtractor resbuilder_v2.ClusterExtractor,
) *Builder {
	return &Builder{
		store:            store,
		httpFilterBuilder: httpFilterBuilder,
		filterChainBuilder: filterChainBuilder,
		routingBuilder:   routingBuilder,
		accessLogBuilder: accessLogBuilder,
		tlsBuilder:       tlsBuilder,
		clusterExtractor: clusterExtractor,
	}
}

// BuildResources is the main entry point for building Envoy resources for a VirtualService
// It returns an interface{} that is actually a *Resources to match the MainBuilder interface
func (b *Builder) BuildResources(vs *v1alpha1.VirtualService) (interface{}, error) {
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

	if len(xdsListener.FilterChains) > 1 {
		return nil, fmt.Errorf("multiple filter chains found in listener %s", listenerNN.String())
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

	// 8. Extract clusters from virtual host and HTTP filters
	virtualHostClusters, err := b.clusterExtractor.ExtractClustersFromVirtualHost(virtualHost)
	if err != nil {
		return nil, fmt.Errorf("failed to extract clusters from virtual host: %w", err)
	}
	resources.Clusters = append(resources.Clusters, virtualHostClusters...)

	httpFilterClusters, err := b.clusterExtractor.ExtractClustersFromHTTPFilters(httpFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to extract clusters from HTTP filters: %w", err)
	}
	resources.Clusters = append(resources.Clusters, httpFilterClusters...)

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