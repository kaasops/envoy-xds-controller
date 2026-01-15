/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateTestCertificate creates a self-signed test certificate with the given expiration time.
// This is intended for use in unit tests only.
// Panics if certificate generation fails (indicates a fundamental problem in tests).
func GenerateTestCertificate(notAfter time.Time) []byte {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("testutil: failed to generate RSA key: " + err.Error())
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic("testutil: failed to create certificate: " + err.Error())
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM
}

// GenerateValidCertificate creates a certificate valid for 24 hours
func GenerateValidCertificate() []byte {
	return GenerateTestCertificate(time.Now().Add(24 * time.Hour))
}

// GenerateExpiredCertificate creates a certificate that expired 24 hours ago
func GenerateExpiredCertificate() []byte {
	return GenerateTestCertificate(time.Now().Add(-24 * time.Hour))
}

// NewTLSSecret creates a TLS secret with the given parameters for testing
func NewTLSSecret(namespace, name string, domains string, certData []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"envoy.kaasops.io/domains": domains,
			},
		},
		Data: map[string][]byte{
			"tls.crt": certData,
			"tls.key": []byte("key"),
		},
	}
}
