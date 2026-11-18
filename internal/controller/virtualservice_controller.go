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
	"maps"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// VirtualServiceReconciler reconciles a VirtualService object
type VirtualServiceReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Updater         *updater.CacheUpdater
	CacheReadyChan  chan struct{}
	VSReconcileChan chan event.GenericEvent
}

// +kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *VirtualServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	<-r.CacheReadyChan

	rlog := log.FromContext(ctx).WithName("virtualService-reconciler").WithValues("virtualService", req.NamespacedName)
	rlog.Info("Reconciling VirtualService")

	var vs envoyv1alpha1.VirtualService
	if err := r.Get(ctx, req.NamespacedName, &vs); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Updater.DeleteVirtualService(ctx, req.NamespacedName)
	}

	r.Updater.ApplyVirtualService(ctx, vs.DeepCopy())
	status, observed := r.Updater.PublicationStatusFor(&vs)
	if observed && (vs.Status.Invalid != status.Invalid || vs.Status.Message != status.Message ||
		vs.Status.ObservedGeneration != vs.Generation) {
		vs.Status.Invalid = status.Invalid
		vs.Status.Message = status.Message
		vs.Status.ObservedGeneration = vs.Generation
		if vs.Status.Message != "" {
			if vs.Status.Invalid {
				rlog.Info("[WARN] VirtualService is invalid", "message", vs.Status.Message)
			} else {
				rlog.Info("[WARN] VirtualService updated with message", "message", vs.Status.Message)
			}
		}
		if err := r.Status().Update(ctx, &vs); err != nil {
			return ctrl.Result{}, err
		}
	}

	rlog.Info("Finished Reconciling VirtualService")

	return ctrl.Result{}, nil
}

// A status write by another replica must not trigger a competing status write.
// Keep informer resyncs (equal resource versions) and changes to VS inputs.
func virtualServiceInputChanged(e event.UpdateEvent) bool {
	old, current := e.ObjectOld, e.ObjectNew
	return old.GetResourceVersion() == current.GetResourceVersion() ||
		old.GetUID() != current.GetUID() || old.GetGeneration() != current.GetGeneration() ||
		!maps.Equal(old.GetAnnotations(), current.GetAnnotations()) ||
		!maps.Equal(old.GetLabels(), current.GetLabels()) ||
		!old.GetDeletionTimestamp().Equal(current.GetDeletionTimestamp())
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.VirtualService{}, builder.WithPredicates(predicate.Funcs{UpdateFunc: virtualServiceInputChanged})).
		Named("virtualservice").
		WatchesRawSource(source.Channel(r.VSReconcileChan, &handler.EnqueueRequestForObject{})).
		WatchesRawSource(source.Func(func(_ context.Context, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) error {
			r.Updater.WatchPublicationStatus(func(key helpers.NamespacedName) {
				queue.Add(ctrl.Request{NamespacedName: client.ObjectKey{Namespace: key.Namespace, Name: key.Name}})
			})
			return nil
		})).
		Complete(r)
}
