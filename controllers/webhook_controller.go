package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/kaasops/cert"

	"github.com/kaasops/envoy-xds-controller/controllers/utils"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	certificateExpirationThreshold = 3 * 24 * time.Hour
	certificateValidity            = 6 * 30 * 24 * time.Hour
)

type WebhookReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Namespace string
	Config    *config.Config
	Log       logr.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add event with TLS secret to Reconcile
	enqueueFn := handler.EnqueueRequestsFromMapFunc(func(context.Context, client.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: r.Namespace,
					Name:      r.Config.GetTLSSecretName(),
				},
			},
		}
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, utils.NamesMatchingPredicate(r.Config.GetTLSSecretName())).
		Watches(&admissionregistrationv1.ValidatingWebhookConfiguration{}, enqueueFn, utils.NamesMatchingPredicate(r.Config.GetValidatingWebhookCfgName())).
		Complete(r)
}

func (r *WebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.Log.WithValues("controller", "webhook")
	r.Log.Info("Reconciling Webhook")

	certSecret := &corev1.Secret{}
	if err := r.Client.Get(ctx, req.NamespacedName, certSecret); err != nil {
		return reconcile.Result{}, errors.Wrap(err, errors.GetFromKubernetesMessage)
	}

	certSecret.Labels = map[string]string{
		options.SecretLabelKey: options.WebhookSecretLabelValue,
	}

	if err := r.ReconcileCertificates(ctx, certSecret); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "cannot reconcile TLS certificate")
	}

	// Check certificate expiried time
	certificate, err := cert.GetCertificateFromBytes(certSecret.Data[corev1.TLSCertKey])
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "cannot get certificate from bytes")
	}

	now := time.Now()
	requeueTime := certificate.NotAfter.Add(-(certificateExpirationThreshold - 1*time.Second))
	rq := requeueTime.Sub(now)

	r.Log.Info("Reconciliation completed, processing back in " + rq.String())

	return reconcile.Result{Requeue: true, RequeueAfter: rq}, nil
}

func (r *WebhookReconciler) ReconcileCertificates(ctx context.Context, certSecret *corev1.Secret) error {

	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: certSecret.Namespace, Name: certSecret.Name}, certSecret); err != nil {
		return errors.Wrap(err, errors.GetFromKubernetesMessage)
	}

	// If need create of update certificate for webhook - do it
	if r.shouldUpdateCertificate(certSecret) {
		r.Log.Info("Generating new TLS Certificate")

		ca, err := cert.GenerateCertificateAuthority()
		if err != nil {
			return errors.Wrap(err, "failed to generate ca certificate")
		}

		opts := cert.NewCertOpts(time.Now().Add(certificateValidity), fmt.Sprintf("envoy-xds-controller-webhook-service.%s.svc", r.Namespace))

		crt, key, err := ca.GenerateCertificate(opts)
		if err != nil {
			return errors.Wrap(err, "failed to generate new TLS certificate")
		}

		caCrt, _ := ca.CACertificatePem()

		certSecret.Data = map[string][]byte{
			corev1.TLSCertKey:              crt.Bytes(),
			corev1.TLSPrivateKeyKey:        key.Bytes(),
			corev1.ServiceAccountRootCAKey: caCrt.Bytes(),
		}

		t := &corev1.Secret{ObjectMeta: certSecret.ObjectMeta}

		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, t, func() error {
			t.Data = certSecret.Data

			return nil
		})
		if err != nil {
			return errors.Wrap(err, errors.UpdateInKubernetesMessage)
		}
	}

	caBundle, ok := certSecret.Data[corev1.ServiceAccountRootCAKey]
	if !ok {
		return errors.New(fmt.Sprintf("missing %s field in %s secret", corev1.ServiceAccountRootCAKey, r.Config.GetTLSSecretName()))
	}

	return r.updateValidatingWebhookConfiguration(ctx, caBundle)
}

// shouldUpdateCertificate checks whether it is necessary to update or create a certificate
func (r *WebhookReconciler) shouldUpdateCertificate(secret *corev1.Secret) bool {
	if _, ok := secret.Data[corev1.ServiceAccountRootCAKey]; !ok {
		return true
	}

	certificate, key, err := cert.GetCertificateWithPrivateKeyFromBytes(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
	if err != nil {
		return true
	}

	if err := cert.ValidateCertificate(certificate, key, certificateExpirationThreshold); err != nil {
		r.Log.V(1).Error(err, "failed to validate certificate, generating new one")

		return true
	}

	r.Log.V(1).Info("Skipping TLS certificate generation as it is still valid")

	return false
}

func (r *WebhookReconciler) updateValidatingWebhookConfiguration(ctx context.Context, caBundle []byte) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		vw := &admissionregistrationv1.ValidatingWebhookConfiguration{}
		err = r.Get(ctx, types.NamespacedName{Name: r.Config.GetValidatingWebhookCfgName()}, vw)
		if err != nil {
			return errors.Wrap(err, "cannot retrieve ValidatingWebhookConfiguration")
		}
		for i, w := range vw.Webhooks {
			// Updating CABundle only in case of an internal service reference
			if w.ClientConfig.Service != nil {
				vw.Webhooks[i].ClientConfig.CABundle = caBundle
			}
		}

		return r.Update(ctx, vw, &client.UpdateOptions{})
	})
}
