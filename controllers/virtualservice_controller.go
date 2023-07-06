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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
)

// VirtualServiceReconciler reconciles a VirtualService object
type VirtualServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *VirtualServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log := log.FromContext(ctx).WithValues("Envoy Cluster", req.NamespacedName)

	log.Info("Start process Envoy Cluster")
	cr, err := r.findVirtualServiceCustomResourceInstance(ctx, req)
	if err != nil {
		log.Error(err, "Failed to get Envoy Cluster CR")
		return ctrl.Result{}, err
	}
	if cr == nil {
		log.Info("Envoy Cluster CR not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	routeConfiguration := &routev3.VirtualHost{}

	unmarshaler := &protojson.UnmarshalOptions{
		AllowPartial: false,
	}

	if err := unmarshaler.Unmarshal(cr.Spec.VirtualHost.Raw, routeConfiguration); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.VirtualService{}).
		Complete(r)
}

func (r *VirtualServiceReconciler) findVirtualServiceCustomResourceInstance(ctx context.Context, req ctrl.Request) (*v1alpha1.VirtualService, error) {
	cr := &v1alpha1.VirtualService{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return cr, nil
}
