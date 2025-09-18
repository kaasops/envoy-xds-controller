package routing

import (
	"fmt"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

// Builder handles the construction of route configurations
type Builder struct {
	store *store.Store
}

// NewBuilder creates a new route builder
func NewBuilder(store *store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// BuildRouteConfiguration builds a complete route configuration from VirtualService
// Implements the RoutingBuilder interface
func (b *Builder) BuildRouteConfiguration(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, nn helpers.NamespacedName) (*routev3.VirtualHost, *routev3.RouteConfiguration, error) {
	virtualHost, err := b.BuildVirtualHost(vs, nn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build virtual host: %w", err)
	}

	routeConfiguration := &routev3.RouteConfiguration{
		Name:         nn.String(),
		VirtualHosts: []*routev3.VirtualHost{virtualHost},
	}

	// Add fallback route for TLS listeners
	// https://github.com/envoyproxy/envoy/issues/37810
	listenerIsTLS := utils.IsTLSListener(xdsListener)
	if listenerIsTLS && !(len(virtualHost.Domains) == 1 && virtualHost.Domains[0] == "*") && utils.ListenerHasPort443(xdsListener) {
		fallbackVH := b.BuildFallbackVirtualHost()
		routeConfiguration.VirtualHosts = append(routeConfiguration.VirtualHosts, fallbackVH)
	}

	if err := routeConfiguration.ValidateAll(); err != nil {
		return nil, nil, fmt.Errorf("failed to validate route configuration: %w", err)
	}

	return virtualHost, routeConfiguration, nil
}

// BuildVirtualHost builds a VirtualHost from VirtualService specification
func (b *Builder) BuildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName) (*routev3.VirtualHost, error) {
	if vs.Spec.VirtualHost == nil {
		return nil, fmt.Errorf("virtual host is empty")
	}

	// Unmarshal base virtual host configuration
	virtualHost := &routev3.VirtualHost{}
	if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return nil, fmt.Errorf("failed to unmarshal virtual host: %w", err)
	}
	virtualHost.Name = nn.String()

	// Process additional routes if specified
	if len(vs.Spec.AdditionalRoutes) > 0 {
		additionalRoutes, err := b.buildAdditionalRoutes(vs.Spec.AdditionalRoutes, vs.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to build additional routes: %w", err)
		}
		virtualHost.Routes = append(virtualHost.Routes, additionalRoutes...)
	}

	// Reorder routes to ensure root routes are at the end
	if err := b.reorderRoutes(virtualHost); err != nil {
		return nil, fmt.Errorf("failed to reorder routes: %w", err)
	}

	// Validate the complete virtual host
	if err := virtualHost.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate virtual host: %w", err)
	}

	// Check domain uniqueness
	if err := utils.CheckAllDomainsUnique(virtualHost.Domains); err != nil {
		return nil, err
	}

	return virtualHost, nil
}

// buildAdditionalRoutes builds routes from references
func (b *Builder) buildAdditionalRoutes(routeRefs []*v1alpha1.ResourceRef, vsNamespace string) ([]*routev3.Route, error) {
	var allRoutes []*routev3.Route

	for _, routeRef := range routeRefs {
		routeRefNs := helpers.GetNamespace(routeRef.Namespace, vsNamespace)
		route := b.store.GetRoute(helpers.NamespacedName{Namespace: routeRefNs, Name: routeRef.Name})
		if route == nil {
			return nil, fmt.Errorf("route %s/%s not found", routeRefNs, routeRef.Name)
		}

		// Process each route specification in the referenced route
		routes, err := b.buildRoutesFromSpec(route.Spec, routeRefNs, routeRef.Name)
		if err != nil {
			return nil, err
		}
		allRoutes = append(allRoutes, routes...)
	}

	return allRoutes, nil
}

// buildRoutesFromSpec builds route objects from route specifications
func (b *Builder) buildRoutesFromSpec(routeSpecs []*runtime.RawExtension, namespace, routeName string) ([]*routev3.Route, error) {
	routes := make([]*routev3.Route, 0, len(routeSpecs))

	for idx, routeSpec := range routeSpecs {
		var r routev3.Route
		if err := protoutil.Unmarshaler.Unmarshal(routeSpec.Raw, &r); err != nil {
			return nil, fmt.Errorf("failed to unmarshal route %s/%s[%d]: %w", namespace, routeName, idx, err)
		}

		// Validate cluster reference if present
		if err := b.validateRouteClusterReferences(&r); err != nil {
			return nil, fmt.Errorf("invalid cluster reference in route %s/%s[%d]: %w", namespace, routeName, idx, err)
		}

		routes = append(routes, &r)
	}

	return routes, nil
}

// validateRouteClusterReferences validates that referenced clusters exist
func (b *Builder) validateRouteClusterReferences(route *routev3.Route) error {
	if routeAction := route.GetRoute(); routeAction != nil {
		switch clusterSpec := routeAction.ClusterSpecifier.(type) {
		case *routev3.RouteAction_Cluster:
			if clusterSpec.Cluster != "" {
				if cl := b.store.GetSpecCluster(clusterSpec.Cluster); cl == nil {
					return fmt.Errorf("cluster %s not found", clusterSpec.Cluster)
				}
			}
		case *routev3.RouteAction_WeightedClusters:
			if clusterSpec.WeightedClusters != nil {
				for _, wc := range clusterSpec.WeightedClusters.Clusters {
					if wc.Name != "" {
						if cl := b.store.GetSpecCluster(wc.Name); cl == nil {
							return fmt.Errorf("weighted cluster %s not found", wc.Name)
						}
					}
				}
			}
		}
	}
	return nil
}

// reorderRoutes ensures that root routes (prefix="/", path="/") are positioned at the end
func (b *Builder) reorderRoutes(virtualHost *routev3.VirtualHost) error {
	if len(virtualHost.Routes) <= 1 {
		return nil // No reordering needed for 0 or 1 routes
	}

	rootMatchIndexes := b.findRootRouteIndexes(virtualHost.Routes)

	switch {
	case len(rootMatchIndexes) > 1:
		return fmt.Errorf("multiple root routes found")
	case len(rootMatchIndexes) == 1:
		rootIndex := rootMatchIndexes[0]
		// Only move if not already at the end
		if rootIndex != len(virtualHost.Routes)-1 {
			b.moveRouteToEnd(virtualHost, rootIndex)
		}
	}

	return nil
}

// findRootRouteIndexes finds indexes of routes that match root paths
func (b *Builder) findRootRouteIndexes(routes []*routev3.Route) []int {
	var rootIndexes []int

	for index, route := range routes {
		if b.isRootRoute(route) {
			rootIndexes = append(rootIndexes, index)
		}
	}

	return rootIndexes
}

// isRootRoute checks if a route matches root path patterns
func (b *Builder) isRootRoute(route *routev3.Route) bool {
	if route.Match == nil {
		return false
	}

	switch pathSpec := route.Match.PathSpecifier.(type) {
	case *routev3.RouteMatch_Prefix:
		return pathSpec.Prefix == "/"
	case *routev3.RouteMatch_Path:
		return pathSpec.Path == "/"
	default:
		return false
	}
}

// moveRouteToEnd moves a route from the given index to the end of the routes slice
func (b *Builder) moveRouteToEnd(virtualHost *routev3.VirtualHost, index int) {
	route := virtualHost.Routes[index]
	// Remove route from current position
	virtualHost.Routes = append(virtualHost.Routes[:index], virtualHost.Routes[index+1:]...)
	// Add route to the end
	virtualHost.Routes = append(virtualHost.Routes, route)
}

// BuildFallbackVirtualHost creates a fallback virtual host for TLS listeners
// This addresses https://github.com/envoyproxy/envoy/issues/37810
func (b *Builder) BuildFallbackVirtualHost() *routev3.VirtualHost {
	return &routev3.VirtualHost{
		Name:    "421vh",
		Domains: []string{"*"},
		Routes: []*routev3.Route{
			{
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Prefix{Prefix: "/"},
				},
				Action: &routev3.Route_DirectResponse{
					DirectResponse: &routev3.DirectResponseAction{
						Status: 421,
					},
				},
			},
		},
	}
}

// ValidateRouteConfiguration performs additional validation on route configuration
func (b *Builder) ValidateRouteConfiguration(routeConfig *routev3.RouteConfiguration) error {
	if routeConfig == nil {
		return fmt.Errorf("route configuration is nil")
	}

	if routeConfig.Name == "" {
		return fmt.Errorf("route configuration name is empty")
	}

	if len(routeConfig.VirtualHosts) == 0 {
		return fmt.Errorf("route configuration has no virtual hosts")
	}

	// Validate each virtual host
	for i, vh := range routeConfig.VirtualHosts {
		if vh.Name == "" {
			return fmt.Errorf("virtual host[%d] name is empty", i)
		}
		if len(vh.Domains) == 0 {
			return fmt.Errorf("virtual host[%d] has no domains", i)
		}
		if len(vh.Routes) == 0 {
			return fmt.Errorf("virtual host[%d] has no routes", i)
		}
	}

	return nil
}
