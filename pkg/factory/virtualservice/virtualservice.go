package virtualservice

import (
	"context"
	"strings"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/factory/virtualservice/tls"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/filterchain"
	"google.golang.org/protobuf/types/known/anypb"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualService struct {
	Name                    string
	NodeIDs                 []string
	VirtualHost             *routev3.VirtualHost
	AccessLog               *accesslogv3.AccessLog
	HttpFilters             []*hcmv3.HttpFilter
	RouteConfig             *routev3.RouteConfiguration
	CertificatesWithDomains map[string][]string
	UseRemoteAddress        *wrapperspb.BoolValue
	UpgradeConfigs          []*hcmv3.HttpConnectionManager_UpgradeConfig
}

type VirtualServiceFactory struct {
	*v1alpha1.VirtualService
	client     client.Client
	listener   *v1alpha1.Listener
	tlsFactory *tls.TlsFactory
}

func NewVirtualServiceFactory(
	client client.Client,
	vs *v1alpha1.VirtualService,
	listener *v1alpha1.Listener,
	tlsFactory tls.TlsFactory,
) *VirtualServiceFactory {
	return &VirtualServiceFactory{
		VirtualService: vs,
		client:         client,
		listener:       listener,
		tlsFactory:     &tlsFactory,
	}
}

func FilterChains(vs *VirtualService) ([]*listenerv3.FilterChain, error) {
	var chains []*listenerv3.FilterChain

	b := filterchain.NewBuilder()

	statPrefix := strings.ReplaceAll(vs.Name, ".", "-")

	if vs.CertificatesWithDomains != nil {
		for certName, domains := range vs.CertificatesWithDomains {
			vs.VirtualHost.Domains = domains
			f, err := b.WithDownstreamTlsContext(certName).
				WithFilterChainMatch(domains).
				WithHttpConnectionManager(vs.AccessLog,
					vs.HttpFilters,
					vs.Name,
					statPrefix,
					vs.UseRemoteAddress,
					vs.UpgradeConfigs,
				).
				Build(vs.Name)
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
		vs.UseRemoteAddress,
		vs.UpgradeConfigs,
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

	httpFilters, err := f.HttpFilters(ctx)
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create HTTP Filters for Virtual Service")
	}

	routeConfig, err := f.RouteConfiguration(name, virtualHost)
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create Route Configs for Virtual Service")
	}

	upgradeConfigs, err := f.UpgradeConfigs()
	if err != nil {
		return VirtualService{}, errors.Wrap(err, "cannot create Upgrade Configs for Virtual Service")
	}

	virtualService := VirtualService{
		Name:             k8s.ResourceName(f.VirtualService.Namespace, f.VirtualService.Name),
		NodeIDs:          nodeIDs,
		VirtualHost:      virtualHost,
		AccessLog:        accesslog,
		HttpFilters:      httpFilters,
		RouteConfig:      routeConfig,
		UseRemoteAddress: f.UseRemoteAddress(),
		UpgradeConfigs:   upgradeConfigs,
	}

	if f.tlsFactory.TlsConfig != nil {
		certificatesWithDomains, err := f.tlsFactory.Provide(ctx, virtualHost.Domains)
		if err != nil {
			return VirtualService{}, errors.Wrap(err, "TLS provider error")
		}

		if len(certificatesWithDomains) == 0 {
			return VirtualService{}, errors.NewUKS("No certificates found")
		}

		virtualService.CertificatesWithDomains = certificatesWithDomains
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

	if err := options.Unmarshaler.Unmarshal(data, &accessLog); err != nil {
		return nil, errors.WrapUKS(err, errors.UnmarshalMessage)
	}

	if err := accessLog.ValidateAll(); err != nil {
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	return &accessLog, nil
}

func (f *VirtualServiceFactory) VirtualHost(ctx context.Context) (*routev3.VirtualHost, error) {
	virtualHost := &routev3.VirtualHost{}
	if err := options.Unmarshaler.Unmarshal(f.Spec.VirtualHost.Raw, virtualHost); err != nil {
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
				if err := options.Unmarshaler.Unmarshal(rt.Raw, routes); err != nil {
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

func (f *VirtualServiceFactory) HttpFilters(ctx context.Context) ([]*hcmv3.HttpFilter, error) {

	httpFilters := []*hcmv3.HttpFilter{}

	rbacFilter, err := v1alpha1.VirtualServiceRBACFilter(ctx, f.client, f.VirtualService)
	if err != nil {
		return nil, err
	}
	if rbacFilter != nil {
		configType := &hcmv3.HttpFilter_TypedConfig{
			TypedConfig: &anypb.Any{},
		}
		if err := configType.TypedConfig.MarshalFrom(rbacFilter); err != nil {
			return nil, err
		}
		httpFilters = append(httpFilters, &hcmv3.HttpFilter{
			Name:       "exc.filters.http.rbac",
			ConfigType: configType,
		})
	}

	for _, httpFilter := range f.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := v1alpha1.UnmarshalAndValidateHTTPFilter(httpFilter.Raw, hf); err != nil {
			return nil, err
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
				if err := v1alpha1.UnmarshalAndValidateHTTPFilter(httpFilter.Raw, hf); err != nil {
					return nil, err
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

func (f *VirtualServiceFactory) UseRemoteAddress() *wrapperspb.BoolValue {
	ura := wrapperspb.BoolValue{
		Value: false,
	}

	if f.Spec.UseRemoteAddress != nil {
		ura = wrapperspb.BoolValue{
			Value: *f.Spec.UseRemoteAddress,
		}
	}

	return &ura
}

func (f *VirtualServiceFactory) UpgradeConfigs() ([]*hcmv3.HttpConnectionManager_UpgradeConfig, error) {
	upgradeConfigs := []*hcmv3.HttpConnectionManager_UpgradeConfig{}
	if f.Spec.UpgradeConfigs != nil {
		for _, upgradeConfig := range f.Spec.UpgradeConfigs {
			uc := &hcmv3.HttpConnectionManager_UpgradeConfig{}
			if err := options.Unmarshaler.Unmarshal(upgradeConfig.Raw, uc); err != nil {
				return upgradeConfigs, errors.WrapUKS(err, errors.UnmarshalMessage)
			}
			if err := uc.ValidateAll(); err != nil {
				return upgradeConfigs, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
			}

			upgradeConfigs = append(upgradeConfigs, uc)
		}
	}

	return upgradeConfigs, nil
}
