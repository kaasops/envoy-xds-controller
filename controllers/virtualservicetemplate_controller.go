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
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// VirtualServiceTemplateReconciler reconciles a VirtualServiceTemplate object
type VirtualServiceTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	EventChan chan event.GenericEvent
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservicetemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservicetemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservicetemplates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualServiceTemplate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *VirtualServiceTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var template envoyv1alpha1.VirtualServiceTemplate
	if err := r.Get(ctx, req.NamespacedName, &template); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var vsList envoyv1alpha1.VirtualServiceList
	if err := r.List(ctx, &vsList, client.MatchingFields{options.VirtualServiceTemplateNameField: req.Name}); err != nil {
		return ctrl.Result{}, err
	}

	for _, vs := range vsList.Items {
		if (vs.Spec.Template.Namespace != nil && *vs.Spec.Template.Namespace == req.Namespace) || req.Namespace == vs.Namespace {
			r.EventChan <- event.GenericEvent{
				Object: &vs,
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualServiceTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.VirtualServiceTemplate{}).
		Complete(r)
}
