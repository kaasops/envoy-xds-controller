package xds

import (
	"log"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
)

type filterChain struct {
	*listenerv3.FilterChain
}

func NewFilterChain() *filterChain {
	f := &listenerv3.FilterChain{}
	return &filterChain{f}
}

func (c *filterChain) WithTLS(secret string) *filterChain {
	t, err := transportSocket(secret)
	if err != nil {
		log.Fatal(err)
	}
	c.TransportSocket = t
	return c
}

func (c *filterChain) WithFilters(v *routev3.VirtualHost) *filterChain {
	f, err := filters(v)
	if err != nil {
		log.Fatal(err)
	}
	c.Filters = f
	return c
}

func transportSocket(secret string) (*corev3.TransportSocket, error) {

	if secret == "" {
		return nil, nil
	}

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

func filters(v *routev3.VirtualHost) ([]*listenerv3.Filter, error) {
	rte := &routev3.RouteConfiguration{
		Name: v.Name,
		VirtualHosts: []*routev3.VirtualHost{{
			Name:    v.Name,
			Domains: v.Domains,
			Routes:  v.Routes,
		}},
	}
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: v.Name,
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
