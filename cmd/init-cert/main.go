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

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kaasops/cert"
	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

const (
	// certificateValidity is the validity of the certificate
	certificateValidity = 365 * 24 * time.Hour
)

type Config struct {
	InstallationNamespace string `default:"envoy-xds-controller" envconfig:"INSTALLATION_NAMESPACE"`
	Webhook               struct {
		TLSSecretName  string `default:"envoy-xds-controller-webhook-cert"           envconfig:"WEBHOOK_TLS_SECRET_NAME"`
		WebhookCfgName string `default:"envoy-xds-controller-validating-webhook-configuration" envconfig:"WEBHOOK_CFG_NAME"`
		ServiceName    string `default:"envoy-xds-controller-webhook-service"        envconfig:"SERVICE_NAME"`
	}
}

func main() {
	// Initialize logger
	log.SetLogger(zap.New())
	logger := log.Log.WithName("init-cert")

	// Parse configuration
	var cfg Config
	err := envconfig.Process("APP", &cfg)
	if err != nil {
		logger.Error(err, "unable to process env var")
		os.Exit(1)
	}

	logger.Info("Starting certificate initialization",
		"namespace", cfg.InstallationNamespace,
		"secretName", cfg.Webhook.TLSSecretName,
		"webhookName", cfg.Webhook.WebhookCfgName,
		"serviceName", cfg.Webhook.ServiceName)

	// Create Kubernetes client
	k8sClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		logger.Error(err, "unable to create Kubernetes client")
		os.Exit(1)
	}

	// Create certificate secret
	certSecret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      cfg.Webhook.TLSSecretName,
			Namespace: cfg.InstallationNamespace,
		},
	}

	// Check if certificate exists
	ctx := context.Background()
	err = k8sClient.Get(ctx, types.NamespacedName{
		Namespace: certSecret.Namespace,
		Name:      certSecret.Name,
	}, certSecret)

	if err != nil {
		logger.Info("Certificate secret not found, creating new one")
		// Create new certificate
		// nolint: lll
		if err := createCertificate(ctx, k8sClient, certSecret, cfg.InstallationNamespace, cfg.Webhook.ServiceName); err != nil {
			logger.Error(err, "failed to create certificate")
			os.Exit(1)
		}
	} else {
		logger.Info("Certificate secret found, checking if it needs to be updated")
		// Check if certificate needs to be updated
		if shouldUpdateCertificate(certSecret) {
			logger.Info("Certificate needs to be updated, creating new one")
			// nolint: lll
			if err := createCertificate(ctx, k8sClient, certSecret, cfg.InstallationNamespace, cfg.Webhook.ServiceName); err != nil {
				logger.Error(err, "failed to update certificate")
				os.Exit(1)
			}
		} else {
			logger.Info("Certificate is valid, no update needed")
		}
	}

	// Update webhook configuration with CA bundle
	caBundle, ok := certSecret.Data[corev1.ServiceAccountRootCAKey]
	if !ok {
		// nolint: lll
		logger.Error(fmt.Errorf("missing %s field in %s secret", corev1.ServiceAccountRootCAKey, cfg.Webhook.TLSSecretName), "invalid certificate secret")
		os.Exit(1)
	}

	if err := updateValidatingWebhookConfiguration(ctx, k8sClient, cfg.Webhook.WebhookCfgName, caBundle); err != nil {
		logger.Error(err, "failed to update ValidatingWebhookConfiguration")
		os.Exit(1)
	}

	logger.Info("Certificate initialization completed successfully")
}

// createCertificate creates a new certificate and updates the secret
func createCertificate(
	ctx context.Context,
	k8sClient client.Client,
	certSecret *corev1.Secret,
	namespace, serviceName string,
) error {
	ca, err := cert.GenerateCertificateAuthority()
	if err != nil {
		return fmt.Errorf("failed to generate certificate authority: %w", err)
	}

	opts := cert.NewCertOpts(time.Now().Add(certificateValidity), fmt.Sprintf("%s.%s.svc", serviceName, namespace))

	crt, key, err := ca.GenerateCertificate(opts)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	caCrt, _ := ca.CACertificatePem()

	certSecret.Data = map[string][]byte{
		corev1.TLSCertKey:              crt.Bytes(),
		corev1.TLSPrivateKeyKey:        key.Bytes(),
		corev1.ServiceAccountRootCAKey: caCrt.Bytes(),
	}

	t := &corev1.Secret{ObjectMeta: certSecret.ObjectMeta}

	_, err = controllerutil.CreateOrUpdate(ctx, k8sClient, t, func() error {
		t.Data = certSecret.Data
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// shouldUpdateCertificate checks whether it is necessary to update or create a certificate
func shouldUpdateCertificate(secret *corev1.Secret) bool {
	if _, ok := secret.Data[corev1.ServiceAccountRootCAKey]; !ok {
		return true
	}

	// nolint: lll
	certificate, key, err := cert.GetCertificateWithPrivateKeyFromBytes(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
	if err != nil {
		return true
	}

	if certificate == nil || key == nil {
		return true
	}

	// Check if the certificate is valid for at least 30 days
	minValidityDuration := 30 * 24 * time.Hour

	return time.Until(certificate.NotAfter) < minValidityDuration
}

// updateValidatingWebhookConfiguration updates the ValidatingWebhookConfiguration with the CA bundle
func updateValidatingWebhookConfiguration(
	ctx context.Context,
	k8sClient client.Client,
	webhookName string,
	caBundle []byte,
) error {
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: webhookName}, webhook)
	if err != nil {
		return fmt.Errorf("failed to get ValidatingWebhookConfiguration: %w", err)
	}

	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	if err := k8sClient.Update(ctx, webhook); err != nil {
		return fmt.Errorf("failed to update ValidatingWebhookConfiguration: %w", err)
	}

	return nil
}
