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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var tracinglog = logf.Log.WithName("tracing-resource")

// SetupTracingWebhookWithManager registers the webhook for Tracing in the manager.
func SetupTracingWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.Tracing{}).
		WithValidator(&TracingCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//nolint:lll // kubebuilder marker must be on single line
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-tracing,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=tracings,verbs=create;update;delete,versions=v1alpha1,name=vtracing-v1alpha1.kb.io,admissionReviewVersions=v1

// TracingCustomValidator struct is responsible for validating the Tracing resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type TracingCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &TracingCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Tracing.
func (v *TracingCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tracing, ok := obj.(*envoyv1alpha1.Tracing)
	if !ok {
		return nil, fmt.Errorf("expected a Tracing object but got %T", obj)
	}
	tracinglog.Info("Validation for Tracing upon creation", "name", tracing.GetName())

	if _, err := tracing.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	tracinglog.Info("Tracing is valid", "name", tracing.GetName())

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (v *TracingCustomValidator) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj runtime.Object,
) (admission.Warnings, error) {
	tracing, ok := newObj.(*envoyv1alpha1.Tracing)
	if !ok {
		return nil, fmt.Errorf("expected a Tracing object for the newObj but got %T", newObj)
	}
	tracinglog.Info("Validation for Tracing upon update", "name", tracing.GetName())

	if _, err := tracing.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	tracinglog.Info("Tracing is valid", "name", tracing.GetName())

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Tracing.
func (v *TracingCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tracing, ok := obj.(*envoyv1alpha1.Tracing)
	if !ok {
		return nil, fmt.Errorf("expected a Tracing object but got %T", obj)
	}
	tracinglog.Info("Validation for Tracing upon deletion", "name", tracing.GetName())

	// check references in VirtualService
	var virtualServiceList envoyv1alpha1.VirtualServiceList
	if err := v.Client.List(ctx, &virtualServiceList, client.InNamespace(tracing.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualService resources: %w", err)
	}
	if len(virtualServiceList.Items) > 0 {
		var refVsNames []string
		for _, vs := range virtualServiceList.Items {
			if vs.Spec.TracingRef != nil && vs.Spec.TracingRef.Name == tracing.GetName() {
				refVsNames = append(refVsNames, vs.GetLabelName())
			}
		}
		if len(refVsNames) > 0 {
			return nil, fmt.Errorf(
				"cannot delete Tracing %s because it is still referenced by VirtualService(s) %s",
				tracing.GetName(), refVsNames)
		}
	}

	// check references in VirtualServiceTemplate
	var virtualServiceTemplateList envoyv1alpha1.VirtualServiceTemplateList
	if err := v.Client.List(ctx, &virtualServiceTemplateList, client.InNamespace(tracing.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServiceTemplate resources: %w", err)
	}
	if len(virtualServiceTemplateList.Items) > 0 {
		var refVstNames []string
		for _, vst := range virtualServiceTemplateList.Items {
			if vst.Spec.TracingRef != nil && vst.Spec.TracingRef.Name == tracing.GetName() {
				refVstNames = append(refVstNames, vst.GetName())
			}
		}
		if len(refVstNames) > 0 {
			return nil, fmt.Errorf(
				"cannot delete Tracing %s because it is still referenced by VirtualServiceTemplate(s) %s",
				tracing.GetName(), refVstNames)
		}
	}

	return nil, nil
}
