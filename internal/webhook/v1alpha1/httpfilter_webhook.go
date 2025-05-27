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
var httpfilterlog = logf.Log.WithName("httpfilter-resource")

// SetupHttpFilterWebhookWithManager registers the webhook for HttpFilter in the manager.
func SetupHttpFilterWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.HttpFilter{}).
		WithValidator(&HttpFilterCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-httpfilter,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=httpfilters,verbs=create;update;delete,versions=v1alpha1,name=vhttpfilter-v1alpha1.envoy.kaasops.io,admissionReviewVersions=v1

// HttpFilterCustomValidator struct is responsible for validating the HttpFilter resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type HttpFilterCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &HttpFilterCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type HttpFilter.
func (v *HttpFilterCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	httpfilter, ok := obj.(*envoyv1alpha1.HttpFilter)
	if !ok {
		return nil, fmt.Errorf("expected a HttpFilter object but got %T", obj)
	}
	httpfilterlog.Info("Validation for HttpFilter upon creation", "name", httpfilter.GetName())

	if _, err := httpfilter.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type HttpFilter.
func (v *HttpFilterCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	httpfilter, ok := newObj.(*envoyv1alpha1.HttpFilter)
	if !ok {
		return nil, fmt.Errorf("expected a HttpFilter object for the newObj but got %T", newObj)
	}
	httpfilterlog.Info("Validation for HttpFilter upon update", "name", httpfilter.GetName())

	if _, err := httpfilter.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type HttpFilter.
func (v *HttpFilterCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	httpfilter, ok := obj.(*envoyv1alpha1.HttpFilter)
	if !ok {
		return nil, fmt.Errorf("expected a HttpFilter object but got %T", obj)
	}
	httpfilterlog.Info("Validation for HttpFilter upon deletion", "name", httpfilter.GetName())

	// check references virtual services

	var virtualServiceList envoyv1alpha1.VirtualServiceList
	if err := v.Client.List(ctx, &virtualServiceList, client.InNamespace(httpfilter.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualService resources: %w", err)
	}

	if len(virtualServiceList.Items) > 0 {
		var refVsNames []string
		for _, vs := range virtualServiceList.Items {
			if len(vs.Spec.AdditionalHttpFilters) > 0 {
				for _, additionalHTTPFilter := range vs.Spec.AdditionalHttpFilters {
					if additionalHTTPFilter.Name == httpfilter.Name {
						refVsNames = append(refVsNames, vs.GetLabelName())
					}
				}
			}
		}
		if len(refVsNames) > 0 {
			return nil, fmt.Errorf("cannot delete HttpFilter %s because it is still referenced by VirtualService(s) %s",
				httpfilter.GetName(),
				refVsNames,
			)
		}
	}

	// check references virtual service templates

	var virtualServiceTemplateList envoyv1alpha1.VirtualServiceTemplateList
	if err := v.Client.List(ctx, &virtualServiceTemplateList, client.InNamespace(httpfilter.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualService resources: %w", err)
	}

	if len(virtualServiceTemplateList.Items) > 0 {
		var refVstNames []string
		for _, vst := range virtualServiceTemplateList.Items {
			if len(vst.Spec.AdditionalHttpFilters) > 0 {
				for _, additionalHTTPFilter := range vst.Spec.AdditionalHttpFilters {
					if additionalHTTPFilter.Name == httpfilter.Name {
						refVstNames = append(refVstNames, vst.GetName())
					}
				}
			}
		}
		if len(refVstNames) > 0 {
			return nil, fmt.Errorf("cannot delete HttpFilter %s because it is still referenced by VirtualServiceTemplate(s) %s",
				httpfilter.GetName(),
				refVstNames,
			)
		}
	}

	return nil, nil
}
