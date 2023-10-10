package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

type CA struct {
	certificate *x509.Certificate
	key         *rsa.PrivateKey
}

func (c CA) CACertificatePem() (b *bytes.Buffer, err error) {
	var crtBytes []byte
	crtBytes, err = x509.CreateCertificate(rand.Reader, c.certificate, c.certificate, &c.key.PublicKey, c.key)

	if err != nil {
		return
	}

	b = new(bytes.Buffer)
	err = pem.Encode(b, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crtBytes,
	})

	return b, err
}

func GenerateCertificateAuthority() (s *CA, err error) {
	s = &CA{
		certificate: &x509.Certificate{
			SerialNumber: big.NewInt(2019),
			Subject: pkix.Name{
				Organization:  []string{"Kaasops"},
				Country:       []string{"CY"},
				Province:      []string{""},
				Locality:      []string{"Limassol"},
				StreetAddress: []string{"666, Best Street"},
				PostalCode:    []string{"6666"},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			IsCA:                  true,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			BasicConstraintsValid: true,
		},
	}

	s.key, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	return
}

func GetCertificateWithPrivateKeyFromBytes(certBytes, keyBytes []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := GetCertificateFromBytes(certBytes)
	if err != nil {
		return nil, nil, err
	}

	key, err := GetPrivateKeyFromBytes(keyBytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

func GetCertificateFromBytes(certBytes []byte) (*x509.Certificate, error) {
	var b *pem.Block

	b, _ = pem.Decode(certBytes)

	return x509.ParseCertificate(b.Bytes)
}

func GetPrivateKeyFromBytes(keyBytes []byte) (*rsa.PrivateKey, error) {
	var b *pem.Block

	b, _ = pem.Decode(keyBytes)

	return x509.ParsePKCS1PrivateKey(b.Bytes)
}

func ValidateCertificate(cert *x509.Certificate, key *rsa.PrivateKey, expirationThreshold time.Duration) error {
	if !key.PublicKey.Equal(cert.PublicKey) {
		return errors.New("certificate signed by wrong public key")
	}

	now := time.Now()

	if now.Before(cert.NotBefore) {
		return errors.New("certificate is not valid yet")
	}

	if now.After(cert.NotAfter.Add(-expirationThreshold)) {
		return errors.New("certificate expired or going to expire soon")
	}

	return nil
}

func (c *CA) GenerateCertificate(opts CertificateOptions) (certificatePem *bytes.Buffer, certificateKey *bytes.Buffer, err error) {
	var certPrivKey *rsa.PrivateKey
	certPrivKey, err = rsa.GenerateKey(rand.Reader, 4096)

	if err != nil {
		return nil, nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"Kaasops"},
			Country:       []string{"CY"},
			Province:      []string{""},
			Locality:      []string{"Limassol"},
			StreetAddress: []string{"666, Best Street"},
			PostalCode:    []string{"6666"},
		},
		DNSNames:     opts.DNSNames(),
		NotBefore:    time.Now().AddDate(0, 0, -1),
		NotAfter:     opts.ExpirationDate(),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	var certBytes []byte
	certBytes, err = x509.CreateCertificate(rand.Reader, cert, c.certificate, &certPrivKey.PublicKey, c.key)

	if err != nil {
		return nil, nil, err
	}

	certificatePem = new(bytes.Buffer)
	err = pem.Encode(certificatePem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	if err != nil {
		return
	}

	certificateKey = new(bytes.Buffer)

	err = pem.Encode(certificateKey, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return
	}

	return
}
