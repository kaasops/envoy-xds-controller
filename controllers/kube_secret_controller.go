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

	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/go-logr/logr"
	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// SecretReconciler reconciles a Secret object
type KubeSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:resources=secrets/status,verbs=get;update;patch
//+kubebuilder:rbac:resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Secret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *KubeSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log := log.FromContext(ctx).WithValues("Kubernetes Certificate Secret", req.NamespacedName)

	log.Info("Start process Kuberentes Secret with certificate")
	kubeSecret, err := r.findkubeSecret(ctx, req)
	if err != nil {
		log.Error(err, "Failed to get Envoy Secret CR")
		return ctrl.Result{}, err
	}
	if kubeSecret == nil {
		log.Info("Kuberentes Secret CR not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	if !checkSecret(log, kubeSecret) {
		return ctrl.Result{}, nil
	}

	// fmt.Printf("SEEEECRET: %+v", kubeSecret)

	err = r.createEnvoySecret(log, kubeSecret)
	if err != nil {
		return ctrl.Result{}, err
	}

	// if secretCR.Spec == nil {
	// 	log.Info("Envoy Secret CR spec not found. Ignoring since object")
	// 	return ctrl.Result{}, nil
	// }

	// if err := xds.Ensure(ctx, r.Cache, secretCR); err != nil {
	// 	return ctrl.Result{}, err
	// }

	return ctrl.Result{}, nil
}

func (r *KubeSecretReconciler) findkubeSecret(ctx context.Context, req ctrl.Request) (*corev1.Secret, error) {
	cr := &corev1.Secret{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return cr, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}

// Check if Kubernetes Secret it TLS secret with ALT names
func checkSecret(log logr.Logger, secret *corev1.Secret) bool {
	if secret.Type != corev1.SecretTypeTLS {
		log.Info("Kuberentes Secret is not a type TLS. Skip")
		return false
	}
	return true
}

func (r *KubeSecretReconciler) createEnvoySecret(log logr.Logger, kubeSecret *corev1.Secret) error {
	log.Info("Create Envoy Secret")

	secret := &tlsv3.Secret{
		Name: kubeSecret.Name,
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

	marshalSecret, err := protojson.Marshal(secret)
	if err != nil {
		return err
	}

	envoySecret := v1alpha1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        kubeSecret.Name,
			Namespace:   kubeSecret.Namespace,
			Annotations: kubeSecret.Annotations,
		},
		Spec: &runtime.RawExtension{
			Raw: marshalSecret,
		},
	}

	createOrUpdateEnvoySecret(context.TODO(), &envoySecret, r.Client)

	return nil

}

func createOrUpdateEnvoySecret(ctx context.Context, desired *v1alpha1.Secret, c client.Client) error {

	// Create Deployment
	err := c.Create(ctx, desired)
	if api_errors.IsAlreadyExists(err) {
		// If alredy exist - compare with existed
		existing := &v1alpha1.Secret{}
		err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing)
		if err != nil {
			return err
		}

		// init Interface for compare
		desiredFields := []interface{}{
			desired.GetAnnotations(),
			desired.GetLabels(),
			desired.Spec,
		}
		existingFields := []interface{}{
			existing.GetAnnotations(),
			existing.GetLabels(),
			existing.Spec,
		}

		// Compare
		if !equality.Semantic.DeepDerivative(desiredFields, existingFields) {
			// Update if not equal
			existing.Labels = desired.Labels
			existing.Annotations = desired.Annotations
			existing.Spec = desired.Spec
			return c.Update(ctx, existing)
		}
		return nil
	}
	return err
}
