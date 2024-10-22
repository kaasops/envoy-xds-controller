package filterchain

import (
	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/utils/strings/slices"

	wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

type Builder interface {
	WithDownstreamTlsContext(secret string) Builder
	WithHttpConnectionManager(
		accessLog *accesslogv3.AccessLog,
		httpFilters []*hcm.HttpFilter,
		routeConfigName string,
		statPrefix string,
		useRemoteAddress *wrappers.BoolValue,
		upgradeConfigs []*hcm.HttpConnectionManager_UpgradeConfig,
	) Builder
	WithFilterChainMatch(domains []string) Builder
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
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
	}

	b.downstreamTlsContext = sdsTls

	return b
}

func (b *builder) WithHttpConnectionManager(
	accessLog *accesslogv3.AccessLog,
	httpFilters []*hcm.HttpFilter,
	routeConfigName string, statPrefix string,
	useRemoteAddress *wrappers.BoolValue,
	upgradeConfigs []*hcm.HttpConnectionManager_UpgradeConfig,
) Builder {
	hfs := []*hcm.HttpFilter{}
	if len(httpFilters) > 0 {
		hfs = append(hfs, httpFilters...)
	}

	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: &corev3.ConfigSource{
					ResourceApiVersion:    corev3.ApiVersion_V3,
					ConfigSourceSpecifier: &corev3.ConfigSource_Ads{},
				},
				RouteConfigName: routeConfigName,
			},
		},
		UseRemoteAddress: useRemoteAddress,
		UpgradeConfigs:   upgradeConfigs,
		HttpFilters:      hfs,
	}

	if accessLog != nil {
		manager.AccessLog = append(manager.AccessLog, accessLog)
	}

	b.httpConnectionManager = manager

	return b
}

func (b *builder) WithFilterChainMatch(domains []string) Builder {
	if slices.Contains(domains, "*") {
		return b
	}

	filterChainMatch := &listenerv3.FilterChainMatch{
		ServerNames: domains,
	}
	b.filterChainMatch = filterChainMatch
	return b
}

func (b *builder) Build(name string) (*listenerv3.FilterChain, error) {
	// I'm get name from prefix. Not good idea
	filterchain := &listenerv3.FilterChain{
		Name: name,
	}

	if err := b.httpConnectionManager.ValidateAll(); err != nil {
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	pbst, err := anypb.New(b.httpConnectionManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal httpConnectionManager to anypb")
	}

	filters := []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}

	filterchain.Filters = filters

	filterchain.FilterChainMatch = b.filterChainMatch

	if err := b.downstreamTlsContext.ValidateAll(); err != nil {
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	if b.downstreamTlsContext != nil {
		scfg, err := anypb.New(b.downstreamTlsContext)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal downstreamTlsContext to anypb")
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

	if err := filterchain.ValidateAll(); err != nil {
		return nil, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
	}

	return filterchain, nil
}
