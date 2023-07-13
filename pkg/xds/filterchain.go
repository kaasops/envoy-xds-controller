package xds

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
)

type FilterChainBuilder interface {
	Build() (*listenerv3.FilterChain, error)
}

type virutalServiceFilterChainBuilder struct {
	virtualHost *routev3.VirtualHost
	secretName  string
}

func NewVirutalServiceFilterChainBuilder(vs *routev3.VirtualHost, secret string) *virutalServiceFilterChainBuilder {
	return &virutalServiceFilterChainBuilder{
		virtualHost: vs,
		secretName:  secret,
	}
}

func (b *virutalServiceFilterChainBuilder) Build() (*listenerv3.FilterChain, error) {

	transportSocket, err := b.BuildTlsTransportSocket()

	if err != nil {
		return nil, err
	}

	filters, err := b.BuildFilters()

	if err != nil {
		return nil, err
	}

	return &listenerv3.FilterChain{
		Filters:         filters,
		TransportSocket: transportSocket,
	}, nil
}

func (b *virutalServiceFilterChainBuilder) BuildTlsTransportSocket() (*corev3.TransportSocket, error) {

	if b.secretName == "" {
		return nil, nil
	}

	sdsTls := &tlsv3.DownstreamTlsContext{
		CommonTlsContext: &tlsv3.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*tlsv3.SdsSecretConfig{{
				Name: b.secretName,
				SdsConfig: &corev3.ConfigSource{
					ConfigSourceSpecifier: &corev3.ConfigSource_Ads{
						Ads: &corev3.AggregatedConfigSource{},
					},
					ResourceApiVersion: corev3.ApiVersion_V3,
				},
			}},
		},
	}

	scfg, err := anypb.New(sdsTls)

	if err != nil {
		return nil, err
	}

	return &corev3.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &corev3.TransportSocket_TypedConfig{
			TypedConfig: scfg,
		},
	}, nil
}

func (b *virutalServiceFilterChainBuilder) BuildFilters() ([]*listenerv3.Filter, error) {
	rte := &routev3.RouteConfiguration{
		Name: b.virtualHost.Name,
		VirtualHosts: []*routev3.VirtualHost{{
			Name:    b.virtualHost.Name,
			Domains: b.virtualHost.Domains,
			Routes:  b.virtualHost.Routes,
		}},
	}
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: b.virtualHost.Name,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: rte,
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}

	pbst, err := anypb.New(manager)
	filters := []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}
	if err != nil {
		return nil, err
	}

	return filters, nil
}
