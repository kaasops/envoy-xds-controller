package virtualservice

import (
	"context"
	"errors"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/util/k8s"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrMultipleAccessLogConfig = errors.New("only one access log config is allowed")
	ErrNodeIDsMismatch         = errors.New("NodeIDs mismatch")
)

type VirtualService struct {
	NodeIDs     []string
	VirtualHost *routev3.VirtualHost
	AccessLog   *accesslogv3.AccessLog
	HttpFilters []*hcmv3.HttpFilter
	RouteConfig *routev3.RouteConfiguration
}

type VirtualServiceFactory struct {
	*v1alpha1.VirtualService
	client      client.Client
	unmarshaler protojson.UnmarshalOptions
	listener    *v1alpha1.Listener
}

func NewVirtualServiceFactory(client client.Client, unmarshaler protojson.UnmarshalOptions, vs *v1alpha1.VirtualService, listener *v1alpha1.Listener) *VirtualServiceFactory {
	return &VirtualServiceFactory{
		VirtualService: vs,
		client:         client,
		unmarshaler:    unmarshaler,
		listener:       listener,
	}
}

func (f *VirtualServiceFactory) Create(ctx context.Context, name string) (VirtualService, error) {

	// If VirtualService nodeIDs is empty use listener nodeIds
	nodeIDs := k8s.NodeIDs(f.VirtualService)
	if len(nodeIDs) == 0 {
		nodeIDs = k8s.NodeIDs(f.listener)
	}

	accesslog, err := f.AccessLog(ctx)
	if err != nil {
		return VirtualService{}, err
	}

	virtualHost, err := f.VirtualHost(ctx)
	if err != nil {
		return VirtualService{}, err
	}

	httpFilters, err := f.HttpFilters(ctx, name)

	if err != nil {
		return VirtualService{}, err
	}

	routeConfig, err := f.RouteConfiguration(name, virtualHost)

	if err != nil {
		return VirtualService{}, err
	}

	return VirtualService{
		NodeIDs:     nodeIDs,
		VirtualHost: virtualHost,
		AccessLog:   accesslog,
		HttpFilters: httpFilters,
		RouteConfig: routeConfig,
	}, nil
}

func (f *VirtualServiceFactory) AccessLog(ctx context.Context) (*accesslogv3.AccessLog, error) {
	var data []byte
	accessLog := accesslogv3.AccessLog{}

	if f.Spec.AccessLog == nil && f.Spec.AccessLogConfig == nil {
		return nil, nil
	}

	if f.Spec.AccessLog != nil {
		data = f.Spec.AccessLog.Raw
	}

	if f.Spec.AccessLogConfig != nil {
		if f.Spec.AccessLog != nil {
			return nil, ErrMultipleAccessLogConfig
		}
		accessLogConfig := &v1alpha1.AccessLogConfig{}
		err := f.client.Get(ctx, f.Spec.AccessLogConfig.NamespacedName(f.Namespace), accessLogConfig)
		if err != nil {
			return nil, err
		}
		data = accessLogConfig.Spec.Raw
	}

	if err := f.unmarshaler.Unmarshal(data, &accessLog); err != nil {
		return nil, err
	}

	if err := accessLog.ValidateAll(); err != nil {
		return nil, err
	}

	return &accessLog, nil
}

func (f *VirtualServiceFactory) VirtualHost(ctx context.Context) (*routev3.VirtualHost, error) {
	virtualHost := &routev3.VirtualHost{}
	if err := f.unmarshaler.Unmarshal(f.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return nil, err
	}

	// TODO: Dont get routes from cluster all the time
	if len(f.Spec.AdditionalRoutes) != 0 {
		for _, rts := range f.Spec.AdditionalRoutes {
			routesSpec := &v1alpha1.Route{}
			err := f.client.Get(ctx, rts.NamespacedName(f.Namespace), routesSpec)
			if err != nil {
				return nil, err
			}
			for _, rt := range routesSpec.Spec {
				routes := &routev3.Route{}
				if err := f.unmarshaler.Unmarshal(rt.Raw, routes); err != nil {
					return nil, err
				}
				virtualHost.Routes = append(virtualHost.Routes, routes)
			}
		}
	}

	if err := virtualHost.ValidateAll(); err != nil {
		return nil, err
	}

	return virtualHost, nil
}

func (f *VirtualServiceFactory) HttpFilters(ctx context.Context, name string) ([]*hcmv3.HttpFilter, error) {
	httpFilters := []*hcmv3.HttpFilter{}
	for _, httpFilter := range f.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := f.unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, err
		}

		if err := hf.ValidateAll(); err != nil {
			return nil, err
		}

		httpFilters = append(httpFilters, hf)
	}

	return httpFilters, nil
}

func (f *VirtualServiceFactory) RouteConfiguration(name string, vh *routev3.VirtualHost) (*routev3.RouteConfiguration, error) {

	routeConfig := &routev3.RouteConfiguration{
		Name: name,
		VirtualHosts: []*routev3.VirtualHost{{
			Name: name,
			// Clean domain list that tls package can split to multiply filterchains
			Domains:             []string{"*"},
			Routes:              vh.Routes,
			RequestHeadersToAdd: vh.RequestHeadersToAdd,
		}},
	}

	if err := routeConfig.ValidateAll(); err != nil {
		return nil, err
	}

	return routeConfig, nil
}
