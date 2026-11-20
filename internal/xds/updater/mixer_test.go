package updater

import (
	"fmt"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

// chainsOf returns the filter chains of a service with a certificate per domain: all of them carry the
// name of the service, and they differ only by their server names.
func chainsOf(service string, n int) []*listenerv3.FilterChain {
	chains := make([]*listenerv3.FilterChain, 0, n)
	for i := range n {
		chains = append(chains, &listenerv3.FilterChain{
			Name:             service,
			FilterChainMatch: &listenerv3.FilterChainMatch{ServerNames: []string{fmt.Sprintf("d%02d.%s", i, service)}},
		})
	}
	return chains
}

// The services common to all nodes come in no particular order. Whatever the order of the services, the
// chains of one service keep their own order in the listener: they share the name the listener is
// sorted by.
func TestListenerKeepsTheOrderOfTheChainsOfAService(t *testing.T) {
	s := store.New()
	s.SetListener(makeTestListenerCR("https", &listenerv3.Listener{
		Name: "https",
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{
			Address: "0.0.0.0", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 443},
		}}},
	}))
	listener := helpers.NamespacedName{Namespace: "default", Name: "https"}
	a, b, c := chainsOf("default/a", 15), chainsOf("default/b", 15), chainsOf("default/c", 15)

	order := func(services ...[]*listenerv3.FilterChain) []string {
		m := NewMixer()
		for _, chains := range services {
			m.AddListenerParams(listener, chains, "node")
		}
		mixed, err := m.Mix(s)
		require.NoError(t, err)
		l := mixed["node"][resource.ListenerType][0].(*listenerv3.Listener)
		names := make([]string, 0, len(l.GetFilterChains()))
		for _, fc := range l.GetFilterChains() {
			names = append(names, fc.GetFilterChainMatch().GetServerNames()[0])
		}
		return names
	}

	want := order(a, b, c)
	require.Equal(t, want, order(c, a, b))
	require.Equal(t, want, order(b, c, a))
	require.Equal(t, want, order(c, b, a))
}
