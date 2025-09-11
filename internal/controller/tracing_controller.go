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

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TracingReconciler reconciles a Tracing object
type TracingReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Updater        *updater.CacheUpdater
	CacheReadyChan chan struct{}
}

// +kubebuilder:rbac:groups=envoy.kaasops.io,resources=tracings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=envoy.kaasops.io,resources=tracings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=envoy.kaasops.io,resources=tracings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tracing object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TracingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	<-r.CacheReadyChan

	rlog := log.FromContext(ctx).WithName("tracing-reconciler").WithValues("tracing", req.NamespacedName)
	rlog.Info("Reconciling Tracing")

	var tracing envoyv1alpha1.Tracing
	if err := r.Get(ctx, req.NamespacedName, &tracing); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		r.Updater.DeleteTracing(ctx, req.NamespacedName)
		return ctrl.Result{}, nil
	}

	r.Updater.ApplyTracing(ctx, &tracing)

	rlog.Info("Finished Reconciling Tracing")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TracingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.Tracing{}).
		Named("tracing").
		Complete(r)
}
