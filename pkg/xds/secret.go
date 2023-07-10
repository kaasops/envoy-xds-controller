package xds

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/kaasops/envoy-xds-controller/pkg/tls"
)

func EnvoySecret(name string, keyPair tls.KeyPair) []types.Resource {
	var s = []types.Resource{
		&tlsv3.Secret{
			Name: name,
			Type: &tlsv3.Secret_TlsCertificate{
				TlsCertificate: &tlsv3.TlsCertificate{
					CertificateChain: &corev3.DataSource{
						Specifier: &corev3.DataSource_InlineBytes{
							InlineBytes: keyPair.Certificate,
						},
					},
					PrivateKey: &corev3.DataSource{
						Specifier: &corev3.DataSource_InlineBytes{
							InlineBytes: keyPair.PrivateKey,
						},
					},
				},
			},
		},
	}
	return s
}
