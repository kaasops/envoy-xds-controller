package adapters

import (
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/interfaces"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/routes"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/utils"
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
	virtualHost, err := a.BuildVirtualHost(vs, nn)
	if err != nil {
		return nil, nil, err
	}

	routeConfig, err := a.builder.BuildRouteConfiguration(vs, nn)
	if err != nil {
		return nil, nil, err
	}

	// Add fallback virtual host for TLS listeners if needed
	listenerIsTLS := utils.IsTLSListener(xdsListener)
	hasPort443 := utils.ListenerHasPort443(xdsListener)
	a.builder.AddFallbackVirtualHostIfNeeded(routeConfig, virtualHost, listenerIsTLS, hasPort443)

	return virtualHost, routeConfig, nil
}

// BuildVirtualHost delegates to the wrapped builder's BuildVirtualHost method
func (a *RoutingAdapter) BuildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName) (*routev3.VirtualHost, error) {
	return a.builder.BuildVirtualHost(vs, nn)
}
