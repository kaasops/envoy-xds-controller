package resbuilder

import (
	"fmt"
	"runtime/debug"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/clusters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/filters"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/main_builder"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/routes"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/secrets"
)

// Note: FilterChainsParams is defined in filter_chains/params.go
// This type alias is removed to avoid duplication (DUP-005)

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

	UpdateResourceBuilder(rb)

	return rb
}

// BuildResources builds all Envoy resources
func (rb *ResourceBuilder) BuildResources(vs *v1alpha1.VirtualService) (*Resources, error) {
	// Input validation
	if vs == nil {
		return nil, fmt.Errorf("virtual service cannot be nil")
	}

	// Ensure builder is initialized
	if rb.mainBuilder == nil {
		UpdateResourceBuilder(rb)
		if rb.mainBuilder == nil {
			return nil, fmt.Errorf("failed to initialize builder")
		}
	}

	// Build resources with panic recovery
	var result interface{}
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				// Include stack trace for debugging (BUG-005 fix)
				err = fmt.Errorf("panic in BuildResources: %v\nstack trace:\n%s", r, debug.Stack())
			}
		}()
		result, err = rb.mainBuilder.BuildResources(vs)
	}()

	if err != nil {
		return nil, fmt.Errorf("BuildResources failed: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("BuildResources returned nil result")
	}

	mainResources, ok := result.(*main_builder.Resources)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Validate required fields
	if mainResources.Listener.Name == "" {
		return nil, fmt.Errorf("invalid result: Listener name is empty")
	}
	if len(mainResources.FilterChain) == 0 {
		return nil, fmt.Errorf("invalid result: FilterChain is empty")
	}

	// Convert to Resources
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
