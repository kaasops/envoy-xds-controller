package adapters

import (
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/routes"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
)

// RoutingAdapter adapts the routes.Builder to implement the RoutingBuilder interface
type RoutingAdapter struct {
	builder *routes.Builder
}

// NewRoutingAdapter creates a new adapter for the routes.Builder
func NewRoutingAdapter(builder *routes.Builder) interfaces.RoutingBuilder {
	return &RoutingAdapter{
		builder: builder,
	}
}

// BuildRouteConfiguration implements the RoutingBuilder interface by providing the
// correct method signature that includes the xdsListener parameter
func (a *RoutingAdapter) BuildRouteConfiguration(
	vs *v1alpha1.VirtualService,
	xdsListener *listenerv3.Listener,
	nn helpers.NamespacedName,
) (*routev3.VirtualHost, *routev3.RouteConfiguration, error) {
	// First build the virtual host
	virtualHost, err := a.BuildVirtualHost(vs, nn)
	if err != nil {
		return nil, nil, err
	}

	// Then build the route configuration
	routeConfig, err := a.builder.BuildRouteConfiguration(vs, nn)
	if err != nil {
		return nil, nil, err
	}

	// For TLS listeners, add a fallback virtual host if needed
	listenerIsTLS := utils.IsTLSListener(xdsListener)
	if listenerIsTLS && !(len(virtualHost.Domains) == 1 && virtualHost.Domains[0] == "*") && utils.ListenerHasPort443(xdsListener) {
		// Build a fallback virtual host
		fallbackVH := &routev3.VirtualHost{
			Name:    "421vh",
			Domains: []string{"*"},
			Routes: []*routev3.Route{
				{
					Match: &routev3.RouteMatch{
						PathSpecifier: &routev3.RouteMatch_Prefix{
							Prefix: "/",
						},
					},
					Action: &routev3.Route_DirectResponse{
						DirectResponse: &routev3.DirectResponseAction{
							Status: 421,
						},
					},
				},
			},
		}

		// Add the fallback virtual host to the route configuration
		routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, fallbackVH)
	}

	return virtualHost, routeConfig, nil
}

// BuildVirtualHost delegates to the wrapped builder's BuildVirtualHost method
func (a *RoutingAdapter) BuildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName) (*routev3.VirtualHost, error) {
	return a.builder.BuildVirtualHost(vs, nn)
}