package virtualservice

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/factory/virtualservice/tls"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/filterchain"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualService struct {
	Name        string
	NodeIDs     []string
	VirtualHost *routev3.VirtualHost
	AccessLog   *accesslogv3.AccessLog
	HttpFilters []*hcmv3.HttpFilter
	RouteConfig *routev3.RouteConfiguration
	Tls         *tls.Tls
}

type VirtualServiceFactory struct {
	*v1alpha1.VirtualService
	client      client.Client
	unmarshaler protojson.UnmarshalOptions
	listener    *v1alpha1.Listener
	tlsFactory  *tls.TlsFactory
}

func NewVirtualServiceFactory(client client.Client, unmarshaler protojson.UnmarshalOptions, vs *v1alpha1.VirtualService, listener *v1alpha1.Listener, tlsFactory tls.TlsFactory) *VirtualServiceFactory {
	return &VirtualServiceFactory{
		VirtualService: vs,
		client:         client,
		unmarshaler:    unmarshaler,
		listener:       listener,
		tlsFactory:     &tlsFactory,
	}
}

func FilterChains(vs *VirtualService) ([]*listenerv3.FilterChain, error) {
	var chains []*listenerv3.FilterChain

	b := filterchain.NewBuilder()

	statPrefix := strings.ReplaceAll(vs.Name, ".", "-")

	if vs.Tls != nil {
		for certName, domains := range vs.Tls.CertificatesWithDomains {
			vs.VirtualHost.Domains = domains
			f, err := b.WithDownstreamTlsContext(certName).
				WithFilterChainMatch(domains).
				WithHttpConnectionManager(vs.AccessLog,
					vs.HttpFilters,
					vs.Name,
					statPrefix,
				).
				Build(fmt.Sprintf("%s-%s", vs.Name, certName))
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate Filter Chain")
			}
			chains = append(chains, f)
		}
		return chains, nil
	}

	f, err := b.WithHttpConnectionManager(
		vs.AccessLog,
		vs.HttpFilters,
		vs.Name,
		statPrefix,
	).
		WithFilterChainMatch(vs.VirtualHost.Domains).
		Build(vs.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate Filter Chain")
	}
	chains = append(chains, f)

	return chains, nil
}

func (f *VirtualServiceFactory) Create(ctx context.Context, name string) (VirtualService, error) {

	// If VirtualService nodeIDs is empty use listener nodeIds
	nodeIDs := k8s.NodeIDs(f.VirtualService)
	if len(nodeIDs) == 0 {
		nodeIDs = k8s.NodeIDs(f.listener)
	}

	accesslog, err := f.AccessLog(ctx)
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create Access Log for Virtual Service")
	}

	virtualHost, err := f.VirtualHost(ctx)
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create Virtual Host for Virtual Service")
	}

	httpFilters, err := f.HttpFilters(ctx, name)
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create HTTP Filters for Virtual Service")
	}

	routeConfig, err := f.RouteConfiguration(name, virtualHost)
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create Route Configs for Virtual Service")
	}

	virtualService := VirtualService{
		Name:        k8s.ResourceName(f.VirtualService.Namespace, f.VirtualService.Name),
		NodeIDs:     nodeIDs,
		VirtualHost: virtualHost,
		AccessLog:   accesslog,
		HttpFilters: httpFilters,
		RouteConfig: routeConfig,
	}

	if f.tlsFactory.TlsConfig != nil {
		tls, err := f.tlsFactory.Provide(ctx, virtualHost.Domains)

		if err != nil {
			return VirtualService{}, errors.Wrap(err, "TLS provider error")
		}

		if len(tls.CertificatesWithDomains) == 0 {
			return VirtualService{}, errors.NewUKS("No certificates found")
		}

		virtualService.Tls = tls
	}

	return virtualService, nil
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
			return nil, errors.New(errors.MultipleAccessLogConfigMessage)
		}
		accessLogConfig := &v1alpha1.AccessLogConfig{}
		err := f.client.Get(ctx, f.Spec.AccessLogConfig.NamespacedName(f.Namespace), accessLogConfig)
		if err != nil {
			return nil, errors.Wrap(err, errors.GetFromKubernetesMessage)
		}
		data = accessLogConfig.Spec.Raw
	}

	if err := f.unmarshaler.Unmarshal(data, &accessLog); err != nil {
		return nil, errors.WrapUKS(err, errors.UnmarshalMessage)
	}

	if err := accessLog.ValidateAll(); err != nil {
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	return &accessLog, nil
}

func (f *VirtualServiceFactory) VirtualHost(ctx context.Context) (*routev3.VirtualHost, error) {
	virtualHost := &routev3.VirtualHost{}
	if err := f.unmarshaler.Unmarshal(f.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return nil, errors.WrapUKS(err, errors.UnmarshalMessage)
	}

	// TODO: Dont get routes from cluster all the time
	if len(f.Spec.AdditionalRoutes) != 0 {
		for _, rts := range f.Spec.AdditionalRoutes {
			routesSpec := &v1alpha1.Route{}
			err := f.client.Get(ctx, rts.NamespacedName(f.Namespace), routesSpec)
			if err != nil {
				return nil, errors.Wrap(err, errors.GetFromKubernetesMessage)
			}
			for _, rt := range routesSpec.Spec {
				routes := &routev3.Route{}
				if err := f.unmarshaler.Unmarshal(rt.Raw, routes); err != nil {
					return nil, errors.WrapUKS(err, errors.UnmarshalMessage)
				}
				virtualHost.Routes = append(virtualHost.Routes, routes)
			}
		}
	}

	if err := virtualHost.ValidateAll(); err != nil {
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	return virtualHost, nil
}

func (f *VirtualServiceFactory) HttpFilters(ctx context.Context, name string) ([]*hcmv3.HttpFilter, error) {
	httpFilters := []*hcmv3.HttpFilter{}
	for _, httpFilter := range f.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := f.unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, errors.WrapUKS(err, errors.UnmarshalMessage)
		}

		if err := hf.ValidateAll(); err != nil {
			return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
		}

		httpFilters = append(httpFilters, hf)
	}

	// TODO: Dont get httpFilters from cluster all the time
	if len(f.Spec.AdditionalHttpFilters) != 0 {
		for _, hfs := range f.Spec.AdditionalHttpFilters {
			hfSpec := &v1alpha1.HttpFilter{}
			err := f.client.Get(ctx, hfs.NamespacedName(f.Namespace), hfSpec)
			if err != nil {
				return nil, errors.Wrap(err, errors.GetFromKubernetesMessage)
			}
			for _, httpFilter := range hfSpec.Spec {
				hf := &hcmv3.HttpFilter{}
				if err := f.unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
					return nil, errors.WrapUKS(err, errors.UnmarshalMessage)
				}
				httpFilters = append(httpFilters, hf)
			}
		}
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
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	return routeConfig, nil
}
