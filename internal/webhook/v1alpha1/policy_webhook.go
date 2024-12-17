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
var policylog = logf.Log.WithName("policy-resource")

// SetupPolicyWebhookWithManager registers the webhook for Policy in the manager.
func SetupPolicyWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.Policy{}).
		WithValidator(&PolicyCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-policy,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=policies,verbs=create;update;delete,versions=v1alpha1,name=vpolicy-v1alpha1.kb.io,admissionReviewVersions=v1

// PolicyCustomValidator struct is responsible for validating the Policy resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PolicyCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &PolicyCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Policy.
func (v *PolicyCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	policy, ok := obj.(*envoyv1alpha1.Policy)
	if !ok {
		return nil, fmt.Errorf("expected a Policy object but got %T", obj)
	}
	policylog.Info("Validation for Policy upon creation", "name", policy.GetName())

	if _, err := policy.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Policy.
func (v *PolicyCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	policy, ok := newObj.(*envoyv1alpha1.Policy)
	if !ok {
		return nil, fmt.Errorf("expected a Policy object for the newObj but got %T", newObj)
	}
	policylog.Info("Validation for Policy upon update", "name", policy.GetName())

	if _, err := policy.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Policy.
func (v *PolicyCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	policy, ok := obj.(*envoyv1alpha1.Policy)
	if !ok {
		return nil, fmt.Errorf("expected a Policy object but got %T", obj)
	}
	policylog.Info("Validation for Policy upon deletion", "name", policy.GetName())

	// check references virtual services

	var virtualServiceList envoyv1alpha1.VirtualServiceList
	if err := v.Client.List(ctx, &virtualServiceList, client.InNamespace(policy.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualService resources: %w", err)
	}

	if len(virtualServiceList.Items) > 0 {
		var refVsNames []string
	LOOP:
		for _, vs := range virtualServiceList.Items {
			if vs.Spec.RBAC != nil && len(vs.Spec.RBAC.AdditionalPolicies) > 0 {
				for _, additionalPolicy := range vs.Spec.RBAC.AdditionalPolicies {
					if additionalPolicy.Name == policy.Name {
						refVsNames = append(refVsNames, vs.Name)
						continue LOOP
					}
				}
			}
		}
		if len(refVsNames) > 0 {
			return nil, fmt.Errorf("cannot delete Policy %s because it is still referenced by VirtualService(s) %s",
				policy.GetName(),
				refVsNames,
			)
		}
	}

	// check references virtual services templates

	var virtualServiceTemplateList envoyv1alpha1.VirtualServiceTemplateList
	if err := v.Client.List(ctx, &virtualServiceTemplateList, client.InNamespace(policy.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServiceTemplate resources: %w", err)
	}

	if len(virtualServiceTemplateList.Items) > 0 {
		var refVstNames []string
	LOOP2:
		for _, vst := range virtualServiceTemplateList.Items {
			if vst.Spec.RBAC != nil && len(vst.Spec.RBAC.AdditionalPolicies) > 0 {
				for _, additionalPolicy := range vst.Spec.RBAC.AdditionalPolicies {
					if additionalPolicy.Name == policy.Name {
						refVstNames = append(refVstNames, vst.Name)
						continue LOOP2
					}
				}
			}
		}
		if len(refVstNames) > 0 {
			return nil, fmt.Errorf("cannot delete Policy %s because it is still referenced by VirtualServiceTemplate(s) %s",
				policy.GetName(),
				refVstNames,
			)
		}
	}

	return nil, nil
}
