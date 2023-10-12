package virtualservice

import (
	"context"
	"errors"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrMultipleAccessLogConfig = errors.New("only one access log config is allowed")
)

type virtualService struct {
	v1alpha1.VirtualService
	unmarshaler *protojson.UnmarshalOptions
}

func NewVirtualService() *virtualService {
	return &virtualService{}
}

func (vs *virtualService) AccessLog(ctx context.Context, client client.Client) (*accesslogv3.AccessLog, error) {
	var data []byte
	var accessLog *accesslogv3.AccessLog

	if vs.Spec.AccessLog == nil && vs.Spec.AccessLogConfig == nil {
		return nil, nil
	}

	if vs.Spec.AccessLog != nil {
		data = vs.Spec.AccessLog.Raw
	}

	if vs.Spec.AccessLogConfig != nil {
		if vs.Spec.AccessLog != nil {
			return nil, ErrMultipleAccessLogConfig
		}
		accessLogConfig := &v1alpha1.AccessLogConfig{}
		err := client.Get(ctx, vs.Spec.AccessLogConfig.NamespacedName(vs.Namespace), accessLogConfig)
		if err != nil {
			return nil, err
		}
		data = accessLogConfig.Spec.Raw
	}

	if err := vs.unmarshaler.Unmarshal(data, accessLog); err != nil {
		return nil, err
	}

	return accessLog, nil
}

func (vs *virtualService) VirtualHost(ctx context.Context, client client.Client) (*routev3.VirtualHost, error) {
	virtualHost := &routev3.VirtualHost{}
	if err := vs.unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return nil, err
	}
	return virtualHost, nil
}

func (vs *virtualService) RouteConfiguration(ctx context.Context, client client.Client, name string) (*routev3.RouteConfiguration, error) {

	vh, err := vs.VirtualHost(ctx, client)
	if err != nil {
		return nil, err
	}

	if len(vs.Spec.AdditionalRoutes) != 0 {
		for _, rts := range vs.Spec.AdditionalRoutes {
			routesSpec := &v1alpha1.Route{}
			err := client.Get(ctx, rts.NamespacedName(vs.Namespace), routesSpec)
			if err != nil {
				return nil, err
			}
			for _, rt := range routesSpec.Spec {
				routes := &routev3.Route{}
				if err := vs.unmarshaler.Unmarshal(rt.Raw, routes); err != nil {
					return nil, err
				}
				vh.Routes = append(vh.Routes, routes)
			}
		}
	}

	routeConfig := &routev3.RouteConfiguration{
		Name: name,
		VirtualHosts: []*routev3.VirtualHost{{
			Name:                name,
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

func (vs *virtualService) HttpFilter(ctx context.Context, client client.Client, name string) ([]*hcmv3.HttpFilter, error) {
	httpFilters := []*hcmv3.HttpFilter{}
	for _, httpFilter := range vs.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := vs.unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, err
		}
		httpFilters = append(httpFilters, hf)
	}
	return httpFilters, nil
}
