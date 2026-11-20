package filter_chains

import (
	"testing"

	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/stretchr/testify/require"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

// A service whose domains take several certificates gets a filter chain per certificate. The chains
// come from a map, so without an order of their own they come out shuffled on every build of the same
// objects, and Envoy is sent a changed listener each time.
func TestFilterChainsFollowTheCertificateNames(t *testing.T) {
	params := &FilterChainsParams{
		StatPrefix:      "https",
		RouteConfigName: "https",
		IsTLS:           true,
		SecretNameToDomains: map[helpers.NamespacedName][]string{
			{Namespace: "certs", Name: "e"}: {"e.example.com"},
			{Namespace: "certs", Name: "a"}: {"a.example.com"},
			{Namespace: "apps", Name: "z"}:  {"z.example.com"},
			{Namespace: "certs", Name: "c"}: {"c.example.com", "cc.example.com"},
			{Namespace: "certs", Name: "b"}: {"b.example.com"},
			{Namespace: "apps", Name: "y"}:  {"y.example.com"},
		},
	}
	want := []string{"apps/y", "apps/z", "certs/a", "certs/b", "certs/c", "certs/e"}

	for range 20 {
		chains, err := NewBuilder(nil).BuildFilterChains(params)
		require.NoError(t, err)
		got := make([]string, 0, len(chains))
		for _, fc := range chains {
			tls := &tlsv3.DownstreamTlsContext{}
			require.NoError(t, fc.GetTransportSocket().GetTypedConfig().UnmarshalTo(tls))
			got = append(got, tls.GetCommonTlsContext().GetTlsCertificateSdsSecretConfigs()[0].GetName())
		}
		require.Equal(t, want, got)
	}
}
