package tls

import (
	"context"
	"errors"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ErrInvalidSecretType = errors.New("invalid secret type")
)

type KeyPair struct {
	Certificate []byte
	PrivateKey  []byte
}

type CertificateGetter interface {
	GetKeyPair() (KeyPair, error)
}

type SecretCertificateGetter struct {
	ctx       context.Context
	Client    client.Client
	SecretRef v1alpha1.ResourceRef
}

func NewSecretCertificateGetter(ctx context.Context, client client.Client, secret v1alpha1.ResourceRef) CertificateGetter {
	return &SecretCertificateGetter{
		ctx:       ctx,
		Client:    client,
		SecretRef: secret,
	}
}

func (s *SecretCertificateGetter) GetKeyPair() (KeyPair, error) {
	log := log.FromContext(s.ctx).WithName("TLS")
	secret := corev1.Secret{}
	err := s.Client.Get(s.ctx, s.SecretRef.NamespacedName(), &secret)
	if err != nil {
		if api_errors.IsNotFound(err) {
			log.Error(err, "TLS secret not found")
			return KeyPair{}, err
		}
		log.Error(err, "Failed to get TLS secret")
		return KeyPair{}, err
	}
	if secret.Type != corev1.SecretTypeTLS {
		log.Error(ErrInvalidSecretType, "TLS type expected")
		return KeyPair{}, err
	}

	return KeyPair{
		Certificate: secret.Data["tls.crt"],
		PrivateKey:  secret.Data["tls.key"],
	}, nil
}
