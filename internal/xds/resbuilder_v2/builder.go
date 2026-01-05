package resbuilder_v2

import (
	"fmt"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/clusters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/main_builder"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/routes"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/secrets"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
)

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
	store           store.Store
	clustersBuilder *clusters.Builder
	filtersBuilder  *filters.Builder
	routesBuilder   *routes.Builder
	secretsBuilder  *secrets.Builder
	mainBuilder     interfaces.MainBuilder
}

// NewResourceBuilder creates a new ResourceBuilder with all modular components
func NewResourceBuilder(store store.Store) *ResourceBuilder {
	rb := &ResourceBuilder{
		store:           store,
		clustersBuilder: clusters.NewBuilder(store),
		filtersBuilder:  filters.NewBuilder(store),
		routesBuilder:   routes.NewBuilder(store),
		secretsBuilder:  secrets.NewBuilder(store),
	}

	// Initialize MainBuilder
	UpdateResourceBuilder(rb)

	return rb
}

// EnableMainBuilder is kept for backward compatibility but is now a no-op
// MainBuilder is always used
func (rb *ResourceBuilder) EnableMainBuilder(enable bool) {
	// No-op: MainBuilder is always enabled
}

// BuildResources builds all Envoy resources using MainBuilder
func (rb *ResourceBuilder) BuildResources(vs *v1alpha1.VirtualService) (*Resources, error) {
	return rb.buildResourcesWithMainBuilder(vs)
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
func BuildResources(vs *v1alpha1.VirtualService, store store.Store) (*Resources, error) {
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
