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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kaasops/envoy-xds-controller/internal/xds/cache"
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
var virtualservicetemplatelog = logf.Log.WithName("virtualservicetemplate-resource")

// SetupVirtualServiceTemplateWebhookWithManager registers the webhook for VirtualServiceTemplate in the manager.
func SetupVirtualServiceTemplateWebhookWithManager(mgr ctrl.Manager, cacheUpdater *updater.CacheUpdater) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.VirtualServiceTemplate{}).
		WithValidator(&VirtualServiceTemplateCustomValidator{Client: mgr.GetClient(), cacheUpdater: cacheUpdater}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-virtualservicetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=virtualservicetemplates,verbs=create;update;delete,versions=v1alpha1,name=vvirtualservicetemplate-v1alpha1.envoy.kaasops.io,admissionReviewVersions=v1

// VirtualServiceTemplateCustomValidator struct is responsible for validating the VirtualServiceTemplate resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type VirtualServiceTemplateCustomValidator struct {
	Client       client.Client
	cacheUpdater *updater.CacheUpdater
}

var _ webhook.CustomValidator = &VirtualServiceTemplateCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type VirtualServiceTemplate.
func (v *VirtualServiceTemplateCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	virtualservicetemplate, ok := obj.(*envoyv1alpha1.VirtualServiceTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualServiceTemplate object but got %T", obj)
	}
	virtualservicetemplatelog.Info("Validation for VirtualServiceTemplate upon creation", "name", virtualservicetemplate.GetName())

	// Validate that all extraFields are used in the template
	if err := validateExtraFieldsUsage(virtualservicetemplate); err != nil {
		return nil, err
	}

	return nil, nil
}

// validateExtraFieldsUsage checks that all extraFields defined in the template are actually used in the template
func validateExtraFieldsUsage(vst *envoyv1alpha1.VirtualServiceTemplate) error {
	// If there are no extraFields, there's nothing to validate
	if len(vst.Spec.ExtraFields) == 0 {
		return nil
	}

	// Valid extraField types
	validTypes := map[string]bool{
		"string": true,
		"enum":   true,
	}

	// Create a map to track which extraFields are used
	extraFieldsUsed := make(map[string]bool)
	for _, field := range vst.Spec.ExtraFields {
		if field.Name == "" {
			return fmt.Errorf("extraField name cannot be empty")
		}
		if field.Type == "" {
			return fmt.Errorf("extraField '%s' type cannot be empty", field.Name)
		}
		if !validTypes[field.Type] {
			return fmt.Errorf("extraField '%s' has unknown type '%s', valid types are: string, enum", field.Name, field.Type)
		}
		if field.Type == "enum" && len(field.Enum) == 0 {
			return fmt.Errorf("extraField '%s' type is 'enum' but no enum values are defined", field.Name)
		}
		extraFieldsUsed[field.Name] = false
	}

	// Convert the template spec to JSON to search for template references
	specJSON, err := json.Marshal(vst.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal template spec: %w", err)
	}

	// Use regex to find all template references in the form {{ .FieldName }}
	// This regex matches {{ .Name }} with optional whitespace
	re := regexp.MustCompile(`{{\s*\.([A-Za-z0-9_]+)\s*}}`)
	matches := re.FindAllStringSubmatch(string(specJSON), -1)

	// Mark each extraField that is used in the template
	for _, match := range matches {
		if len(match) > 1 {
			fieldName := match[1]
			if _, exists := extraFieldsUsed[fieldName]; exists {
				extraFieldsUsed[fieldName] = true
			}
		}
	}

	// Check if any extraField is not used
	var unusedFields []string
	for fieldName, used := range extraFieldsUsed {
		if !used {
			unusedFields = append(unusedFields, fieldName)
		}
	}

	// Return an error if there are unused extraFields
	if len(unusedFields) > 0 {
		return fmt.Errorf("the following extraFields are defined but not used in the template: %s", strings.Join(unusedFields, ", "))
	}

	return nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type VirtualServiceTemplate.
func (v *VirtualServiceTemplateCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	virtualservicetemplate, ok := newObj.(*envoyv1alpha1.VirtualServiceTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualServiceTemplate object but got %T", newObj)
	}
	virtualservicetemplatelog.Info("Validation for VirtualServiceTemplate upon update", "name", virtualservicetemplate.GetName())

	// Validate that all extraFields are used in the template
	if err := validateExtraFieldsUsage(virtualservicetemplate); err != nil {
		return nil, err
	}

	cacheUpdater := updater.NewCacheUpdater(cache.NewSnapshotCache(), v.cacheUpdater.CopyStore())
	if err := cacheUpdater.RebuildSnapshots(ctx); err != nil {
		return nil, fmt.Errorf("failed build snapshot for validation: %w", err)
	}
	if err := cacheUpdater.ApplyVirtualServiceTemplate(ctx, virtualservicetemplate); err != nil {
		return nil, fmt.Errorf("failed to apply VirtualServiceTemplate: %w", err)
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type VirtualServiceTemplate.
func (v *VirtualServiceTemplateCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	virtualservicetemplate, ok := obj.(*envoyv1alpha1.VirtualServiceTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualServiceTemplate object but got %T", obj)
	}
	virtualservicetemplatelog.Info("Validation for VirtualServiceTemplate upon deletion", "name", virtualservicetemplate.GetName())

	var virtualServiceList envoyv1alpha1.VirtualServiceList
	if err := v.Client.List(ctx, &virtualServiceList, client.InNamespace(virtualservicetemplate.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualService resources: %w", err)
	}

	if len(virtualServiceList.Items) > 0 {
		var refVsNames []string
		for _, vs := range virtualServiceList.Items {
			if vs.Spec.Template != nil && vs.Spec.Template.Name == virtualservicetemplate.GetName() {
				refVsNames = append(refVsNames, vs.GetLabelName())
			}
		}
		if len(refVsNames) > 0 {
			return nil, fmt.Errorf("cannot delete VirtualServiceTemplate %s because it is still referenced by VirtualService(s) %s",
				virtualservicetemplate.GetName(),
				refVsNames,
			)
		}
	}
	return nil, nil
}
