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

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
)

// EndpointReconciler reconciles a Endpoint object
type EndpointReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Cache       xdscache.Cache
	Unmarshaler *protojson.UnmarshalOptions
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=endpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=endpoints/finalizers,verbs=update

func (r *EndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("Envoy Endpoint", req.NamespacedName)
	log.Info("Reconciling endpoint")

	// Get Endpoint instance
	instance := &v1alpha1.Endpoint{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			log.Info("Endpoint instance not found. Delete object fron xDS cache")
			for _, nodeID := range NodeIDs(instance, r.Cache) {
				if err := r.Cache.Delete(nodeID, &endpointv3.Endpoint{}, getResourceName(req.Namespace, req.Name)); err != nil {
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

	// get envoy endpoint from endpoint instance spec
	endpoint := &endpointv3.Endpoint{}
	if err := r.Unmarshaler.Unmarshal(instance.Spec.Raw, endpoint); err != nil {
		return ctrl.Result{}, err
	}

	for _, nodeID := range NodeIDs(instance, r.Cache) {
		if err := r.Cache.Update(nodeID, endpoint, getResourceName(instance.Namespace, instance.Name)); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Endpoint{}).
		Complete(r)
}
