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

	"github.com/go-logr/logr"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	api_errors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cache  *xdscache.Cache

	log logr.Logger
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=clusters/finalizers,verbs=update

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx).WithValues("Envoy Cluster", req.NamespacedName)
	r.log.Info("Reconciling cluster")

	// Get Cluster instance
	instance := &v1alpha1.Cluster{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			r.log.Info("Cluster instance not found. Delete object fron xDS cache")
			nodeIDs, err := r.Cache.GetNodeIDsForResource(resourcev3.ClusterType, getResourceName(req.Namespace, req.Name))
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, errors.GetNodeIDForResource)
			}
			for _, nodeID := range nodeIDs {
				if err := r.Cache.Delete(nodeID, resourcev3.ClusterType, getResourceName(req.Namespace, req.Name)); err != nil {
					return ctrl.Result{}, errors.Wrap(err, errors.CannotDeleteFromCacheMessage)
				}
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, errors.GetFromKubernetesMessage)
	}

	if instance.Spec == nil {
		return ctrl.Result{}, errors.New(errors.EmptySpecMessage)
	}

	// get envoy cluster from cluster instance spec
	cluster := &clusterv3.Cluster{}
	if err := options.Unmarshaler.Unmarshal(instance.Spec.Raw, cluster); err != nil {
		return ctrl.Result{}, errors.Wrap(err, errors.UnmarshalMessage)
	}

	nodeIDs := k8s.NodeIDs(instance)
	if len(nodeIDs) == 0 {
		defaultNodeIDs, err := defaultNodeIDs(ctx, r.Client, req.Namespace)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, errors.GetDefaultNodeIDMessage)
		}
		nodeIDs = append(nodeIDs, defaultNodeIDs...)
	}

	for _, nodeID := range nodeIDs {
		if err := r.Cache.Update(nodeID, cluster); err != nil {
			return ctrl.Result{}, errors.Wrap(err, errors.CannotUpdateCacheMessage)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cluster{}).
		Complete(r)
}
