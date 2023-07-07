package xds

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

type ListenerBuilder interface {
	SetSocketAddress(address string, port string) ListenerBuilder
	SetFilterChains() ListenerBuilder
	SetVirtualService(v1alpha1.VirtualService) ListenerBuilder
	Build() listenerv3.Listener
}

type listenerBuilder struct {
	Address      *corev3.Address
	baseListener listenerv3.Listener
	virtualHost  routev3.VirtualHost
}

func (b *listenerBuilder) Build() listenerv3.Listener {
	return listenerv3.Listener{}
}

func (b *listenerBuilder) SetSocketAddress(address string, port uint32) ListenerBuilder {
	b.Address = &corev3.Address{
		Address: &corev3.Address_SocketAddress{
			SocketAddress: &corev3.SocketAddress{
				Protocol: corev3.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &corev3.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
	return nil
}
