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

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/go-logr/logr"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/util/k8s"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"google.golang.org/protobuf/encoding/protojson"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VirtualHostReconciler reconciles a VirtualHost object
type VirtualHostReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Cache       xdscache.Cache
	Unmarshaler protojson.UnmarshalOptions

	log logr.Logger
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualhosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualhosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualhosts/finalizers,verbs=update

func (r *VirtualHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx).WithValues("Envoy Virtualhost", req.NamespacedName)
	r.log.Info("Reconciling virtualhost")

	// Get Virtualhost instance
	instance := &v1alpha1.VirtualHost{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			r.log.Info("Virtualhost instance not found. Delete object fron xDS cache")
			for _, nodeID := range k8s.NodeIDs(instance) {
				if err := r.Cache.Delete(nodeID, resourcev3.VirtualHostType, getResourceName(req.Namespace, req.Name)); err != nil {
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

	// get envoy virtualhost from virtualhost instance spec
	virtualhost := &routev3.VirtualHost{}
	if err := r.Unmarshaler.Unmarshal(instance.Spec.Raw, virtualhost); err != nil {
		return ctrl.Result{}, err
	}

	for _, nodeID := range k8s.NodeIDs(instance) {
		if err := r.Cache.Update(nodeID, virtualhost); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VirtualHost{}).
		Complete(r)
}
