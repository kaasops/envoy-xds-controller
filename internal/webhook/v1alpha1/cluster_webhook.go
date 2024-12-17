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

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var clusterlog = logf.Log.WithName("cluster-resource")

// SetupClusterWebhookWithManager registers the webhook for Cluster in the manager.
func SetupClusterWebhookWithManager(mgr ctrl.Manager, cacheUpdater *updater.CacheUpdater) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.Cluster{}).
		WithValidator(&ClusterCustomValidator{Client: mgr.GetClient(), cacheUpdater: cacheUpdater}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=clusters,verbs=create;update;delete,versions=v1alpha1,name=vcluster-v1alpha1.kb.io,admissionReviewVersions=v1

// ClusterCustomValidator struct is responsible for validating the Cluster resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ClusterCustomValidator struct {
	Client       client.Client
	cacheUpdater *updater.CacheUpdater
}

var _ webhook.CustomValidator = &ClusterCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Cluster.
func (v *ClusterCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster, ok := obj.(*envoyv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("expected a Cluster object but got %T", obj)
	}
	clusterlog.Info("Validation for Cluster upon creation", "name", cluster.GetName())

	clusterV3, err := cluster.UnmarshalV3AndValidate()
	if err != nil {
		return nil, err
	}

	if val := v.cacheUpdater.GetSpecCluster(clusterV3.Name); val != nil &&
		(val.Name != cluster.Name || val.Namespace != cluster.Namespace) {
		return nil, fmt.Errorf("cluster %s already exists", clusterV3.Name)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Cluster.
func (v *ClusterCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cluster, ok := newObj.(*envoyv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("expected a Cluster object for the newObj but got %T", newObj)
	}
	clusterlog.Info("Validation for Cluster upon update", "name", cluster.GetName())

	if _, err := cluster.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Cluster.
func (v *ClusterCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster, ok := obj.(*envoyv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("expected a Cluster object but got %T", obj)
	}
	clusterlog.Info("Validation for Cluster upon deletion", "name", cluster.GetName())

	clusterV3, err := cluster.UnmarshalV3()
	if err != nil {
		return nil, err
	}

	var routes envoyv1alpha1.RouteList
	if err := v.Client.List(ctx, &routes, client.InNamespace(cluster.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list routes for cluster %s/%s: %v", cluster.Namespace, cluster.Name, err)
	}
	for _, route := range routes.Items {
		routesV3, err := route.UnmarshalV3AndValidate()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal routes for cluster %s/%s: %v", cluster.Namespace, cluster.Name, err)
		}
		for _, routeV3 := range routesV3 {
			if routeAction := routeV3.GetRoute(); routeAction != nil {
				if routeAction.GetCluster() == clusterV3.Name {
					return nil, fmt.Errorf("route for cluster %s/%s is still in use in Route %s", cluster.Namespace, cluster.Name, route.Name) // TODO: all routes
				}
			}
		}
	}

	return nil, nil
}
