package tls

import (
	"errors"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

var (
	ErrInvalidSecretType = errors.New("invalid secret type")
)

type CertificateGetter interface {
	GetCerts() (map[string][]string, error)
}

type virtualServiceCertificateGetter struct {
	virtualHost *routev3.VirtualHost
	tlsConfig   *v1alpha1.TlsConfigSpec
}

func NewVirtualServiceCertificateGetter(vh *routev3.VirtualHost, tlsconfig *v1alpha1.TlsConfigSpec) CertificateGetter {
	return &virtualServiceCertificateGetter{
		virtualHost: vh,
		tlsConfig:   tlsconfig,
	}
}

func (s *virtualServiceCertificateGetter) GetCerts() (map[string][]string, error) {
	return map[string][]string{"secret1-cert": ([]string{"example.com"})}, nil
}
