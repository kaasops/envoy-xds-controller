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
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
)

// RouteReconciler reconciles a Route object
type RouteReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Cache       *xdscache.Cache
	Unmarshaler *protojson.UnmarshalOptions
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=routes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=routes/finalizers,verbs=update

func (r *RouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("Envoy Route", req.NamespacedName)
	log.Info("Reconciling route")

	// Get Route instance
	instance := &v1alpha1.Route{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			log.Info("Route instance not found. Delete object fron xDS cache")
			for _, nodeID := range NodeIDs(instance, r.Cache) {
				if err := r.Cache.Delete(nodeID, resourcev3.RouteType, getResourceName(req.Namespace, req.Name)); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.Spec == nil {
		return ctrl.Result{}, ErrEmptySpec
	}

	// get envoy route from route instance spec
	route := &routev3.RouteConfiguration{}
	if err := r.Unmarshaler.Unmarshal(instance.Spec.Raw, route); err != nil {
		return ctrl.Result{}, err
	}

	for _, nodeID := range NodeIDs(instance, r.Cache) {
		if err := r.Cache.Update(nodeID, route); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Route{}).
		Complete(r)
}
