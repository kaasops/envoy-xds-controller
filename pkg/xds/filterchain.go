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
	WithTlsTransportSocket(secretName string) FilterChainBuilder
	WithFilters(virtualHost routev3.VirtualHost) FilterChainBuilder
	Build() (*listenerv3.FilterChain, error)
}

type filterChainBuilder struct {
	TlsContext *tlsv3.DownstreamTlsContext
	Manager    *hcm.HttpConnectionManager
}

func NewFilterChainBuilder() *filterChainBuilder {
	return &filterChainBuilder{}
}

func (b *filterChainBuilder) WithTlsTransportSocket(secretName string) FilterChainBuilder {
	sdsTls := &tlsv3.DownstreamTlsContext{
		CommonTlsContext: &tlsv3.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*tlsv3.SdsSecretConfig{{
				Name: secretName,
				SdsConfig: &corev3.ConfigSource{
					ConfigSourceSpecifier: &corev3.ConfigSource_Ads{
						Ads: &corev3.AggregatedConfigSource{},
					},
					ResourceApiVersion: corev3.ApiVersion_V3,
				},
			}},
		},
	}

	b.TlsContext = sdsTls

	return b
}

func (b *filterChainBuilder) WithFilters(virtualHost routev3.VirtualHost) FilterChainBuilder {
	rte := &routev3.RouteConfiguration{
		Name: virtualHost.Name,
		VirtualHosts: []*routev3.VirtualHost{{
			Name:    virtualHost.Name,
			Domains: virtualHost.Domains,
			Routes:  virtualHost.Routes,
		}},
	}
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: virtualHost.Name,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: rte,
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}

	b.Manager = manager

	return b
}

func (b *filterChainBuilder) Build() (*listenerv3.FilterChain, error) {
	scfg, err := anypb.New(b.TlsContext)
	if err != nil {
		return nil, err
	}
	transportSocket := &corev3.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &corev3.TransportSocket_TypedConfig{
			TypedConfig: scfg,
		},
	}

	pbst, err := anypb.New(b.Manager)
	filters := []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}
	if err != nil {
		return nil, err
	}

	return &listenerv3.FilterChain{
		Filters:         filters,
		TransportSocket: transportSocket,
	}, nil
}
