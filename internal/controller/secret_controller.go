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

package controller

import (
	"context"

	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Updater        *updater.CacheUpdater
	CacheReadyChan chan struct{}
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Policy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	<-r.CacheReadyChan
	rlog := log.FromContext(ctx).WithName("secret-reconciler").WithValues("secret", req.NamespacedName)
	rlog.Info("Reconciling Secret")

	var secret v1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Updater.DeleteSecret(ctx, req.NamespacedName)
	}

	if err := r.Updater.ApplySecret(ctx, &secret); err != nil {
		return ctrl.Result{}, err
	}

	rlog.Info("Finished Reconciling Secret")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).
		Named("kubernetes-secret").
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return filterSecret(e.Object.(*v1.Secret))
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return filterSecret(e.ObjectNew.(*v1.Secret))
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return filterSecret(e.Object.(*v1.Secret))
			},
		}).
		Complete(r)
}

func filterSecret(s *v1.Secret) bool {
	if s.Type != v1.SecretTypeOpaque && s.Type != v1.SecretTypeTLS {
		return false
	}
	if _, ok := s.Labels["envoy.kaasops.io/secret-type"]; !ok {
		return false
	}
	return true
}
