package filterchain

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
)

type Builder interface {
	WithDownstreamTlsContext(secret string) Builder
	WithHttpConnectionManager(virtualHost *routev3.VirtualHost) Builder
	WithFilterChainMatch(virtualHost *routev3.VirtualHost) Builder
	Build(name string) (*listenerv3.FilterChain, error)
}

type builder struct {
	filterchain           *listenerv3.FilterChain
	downstreamTlsContext  *tlsv3.DownstreamTlsContext
	httpConnectionManager *hcm.HttpConnectionManager
	filterChainMatch      *listenerv3.FilterChainMatch
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) WithDownstreamTlsContext(secret string) Builder {
	sdsTls := &tlsv3.DownstreamTlsContext{
		CommonTlsContext: &tlsv3.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*tlsv3.SdsSecretConfig{{
				Name: secret,
				SdsConfig: &corev3.ConfigSource{
					ConfigSourceSpecifier: &corev3.ConfigSource_Ads{
						Ads: &corev3.AggregatedConfigSource{},
					},
					ResourceApiVersion: corev3.ApiVersion_V3,
				},
			}},
		},
	}

	b.downstreamTlsContext = sdsTls

	return b
}

func (b *builder) WithHttpConnectionManager(v *routev3.VirtualHost) Builder {
	rte := &routev3.RouteConfiguration{
		Name: v.Name,
		VirtualHosts: []*routev3.VirtualHost{{
			Name:    v.Name,
			Domains: v.Domains,
			Routes:  v.Routes,
		}},
	}
	routerConfig, _ := anypb.New(&router.Router{})

	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: v.Name,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: rte,
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: routerConfig,
			},
		}},
	}

	b.httpConnectionManager = manager
	return b
}

func (b *builder) WithFilterChainMatch(virtualHost *routev3.VirtualHost) Builder {
	filterChainMatch := &listenerv3.FilterChainMatch{
		ServerNames: virtualHost.Domains,
	}
	b.filterChainMatch = filterChainMatch
	return b
}

func (b *builder) Build(name string) (*listenerv3.FilterChain, error) {

	filterchain := &listenerv3.FilterChain{
		Name: name,
	}

	pbst, err := anypb.New(b.httpConnectionManager)

	if err != nil {
		return nil, err
	}

	filters := []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}

	filterchain.Filters = filters

	filterchain.FilterChainMatch = b.filterChainMatch

	if b.downstreamTlsContext != nil {
		scfg, err := anypb.New(b.downstreamTlsContext)
		if err != nil {
			return nil, err
		}

		transportSocker := &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: scfg,
			},
		}
		filterchain.TransportSocket = transportSocker
	}

	b.filterchain = filterchain

	return filterchain, nil
}
