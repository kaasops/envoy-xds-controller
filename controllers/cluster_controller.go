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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/resources"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=clusters/finalizers,verbs=update

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log := log.FromContext(ctx).WithValues("Envoy Cluster", req.NamespacedName)

	log.Info("Start process Envoy Cluster")
	clusterCR, err := r.findClusterCustomResourceInstance(ctx, req)
	if err != nil {
		log.Error(err, "Failed to get Envoy Cluster CR")
		return ctrl.Result{}, err
	}
	if clusterCR == nil {
		log.Info("Envoy Cluster CR not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}
	if clusterCR.Spec == nil {
		log.Info("Envoy Cluster CR spec not found. Ignoring since object")
		return ctrl.Result{}, nil
	}

	cc := resources.NewClusterController(r.Client, *clusterCR)
	cluster, err := cc.GetCluster(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println(cluster.LbPolicy, cluster.ClusterDiscoveryType, cluster.ConnectTimeout)

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) findClusterCustomResourceInstance(ctx context.Context, req ctrl.Request) (*v1alpha1.Cluster, error) {
	cr := &v1alpha1.Cluster{}
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
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cluster{}).
		Complete(r)
}
