/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"

	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecretReconciler reconciles a Secret object
type KubeSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cache  *xdscache.Cache

	log logr.Logger
}

//+kubebuilder:rbac:resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:resources=secrets/status,verbs=get;update;patch
//+kubebuilder:rbac:resources=secrets/finalizers,verbs=update

func (r *KubeSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx).WithValues("Kubernetes TLS Secret", req.NamespacedName)
	r.log.V(1).Info("Reconciling kubernetes tls secrets")

	// Get secret
	kubeSecret := &corev1.Secret{}
	err := r.Get(ctx, req.NamespacedName, kubeSecret)
	if err != nil {
		if api_errors.IsNotFound(err) {
			r.log.Info("Secret not found. Delete object fron xDS cache")
			nodeIDs, err := r.Cache.GetNodeIDsForResource(resourcev3.SecretType, getResourceName(req.Namespace, req.Name))
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, errors.GetNodeIDForResource)
			}
			for _, nodeID := range nodeIDs {
				if err := r.Cache.Delete(nodeID, resourcev3.SecretType, getResourceName(req.Namespace, req.Name)); err != nil {
					return ctrl.Result{}, errors.Wrap(err, errors.CannotDeleteFromCacheMessage)
				}
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, errors.GetFromKubernetesMessage)
	}

	if !r.valid(kubeSecret) {
		return ctrl.Result{}, nil
	}

	nodeIDs := k8s.NodeIDs(kubeSecret)
	if len(nodeIDs) == 0 {
		defaultNodeIDs, err := defaultNodeIDs(ctx, r.Client, req.Namespace)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, errors.GetDefaultNodeIDMessage)
		}
		nodeIDs = append(nodeIDs, defaultNodeIDs...)
	}

	for _, nodeID := range nodeIDs {
		envoySecret, err := r.makeEnvoySecret(kubeSecret)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "cannot generate xDS secret from Kubernetes Secret")
		}
		if err := r.Cache.Update(nodeID, envoySecret); err != nil {
			return ctrl.Result{}, errors.Wrap(err, errors.CannotUpdateCacheMessage)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}

// Check if Kubernetes Secret it TLS secret with ALT names
func (r *KubeSecretReconciler) valid(secret *corev1.Secret) bool {
	v, ok := secret.Labels[options.SecretLabelKey]
	if !ok || v != options.SdsSecretLabelValue {
		r.log.V(1).Info("Not a xds controller secret")
		return false
	}
	if secret.Type != corev1.SecretTypeTLS {
		r.log.V(1).Info("Kuberentes Secret is not a type TLS. Skip")
		return false
	}
	return true
}

func (r *KubeSecretReconciler) makeEnvoySecret(kubeSecret *corev1.Secret) (*tlsv3.Secret, error) {
	envoySecret := &tlsv3.Secret{
		Name: fmt.Sprintf("%s-%s", kubeSecret.Namespace, kubeSecret.Name),
		Type: &tlsv3.Secret_TlsCertificate{
			TlsCertificate: &tlsv3.TlsCertificate{
				CertificateChain: &corev3.DataSource{
					Specifier: &corev3.DataSource_InlineBytes{
						InlineBytes: kubeSecret.Data["tls.crt"],
					},
				},
				PrivateKey: &corev3.DataSource{
					Specifier: &corev3.DataSource_InlineBytes{
						InlineBytes: kubeSecret.Data["tls.key"],
					},
				},
			},
		},
	}
	if err := envoySecret.ValidateAll(); err != nil {
		return nil, errors.Wrap(err, "cannot validate Envoy Secret")
	}

	return envoySecret, nil
}
