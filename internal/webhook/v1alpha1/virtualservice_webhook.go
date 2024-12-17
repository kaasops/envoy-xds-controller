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

	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
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
var virtualservicelog = logf.Log.WithName("virtualservice-resource")

// SetupVirtualServiceWebhookWithManager registers the webhook for VirtualService in the manager.
func SetupVirtualServiceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.VirtualService{}).
		WithValidator(&VirtualServiceCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-virtualservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=virtualservices,verbs=create;update,versions=v1alpha1,name=vvirtualservice-v1alpha1.kb.io,admissionReviewVersions=v1

// VirtualServiceCustomValidator struct is responsible for validating the VirtualService resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type VirtualServiceCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &VirtualServiceCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type VirtualService.
func (v *VirtualServiceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	virtualservice, ok := obj.(*envoyv1alpha1.VirtualService)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualService object but got %T", obj)
	}
	virtualservicelog.Info("Validation for VirtualService upon creation", "name", virtualservice.GetName())

	if err := v.validateVirtualService(ctx, virtualservice); err != nil {
		return nil, fmt.Errorf("failed to validate VirtualService %s: %w", virtualservice.Name, err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type VirtualService.
func (v *VirtualServiceCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	virtualservice, ok := newObj.(*envoyv1alpha1.VirtualService)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualService object for the newObj but got %T", newObj)
	}
	virtualservicelog.Info("Validation for VirtualService upon update", "name", virtualservice.GetName())

	if err := v.validateVirtualService(ctx, virtualservice); err != nil {
		return nil, fmt.Errorf("failed to validate VirtualService %s: %w", virtualservice.Name, err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type VirtualService.
func (v *VirtualServiceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *VirtualServiceCustomValidator) validateVirtualService(ctx context.Context, vs *envoyv1alpha1.VirtualService) error {
	if len(vs.GetNodeIDs()) == 0 {
		return fmt.Errorf("nodeIDs is required")
	}
	s := store.New()
	if err := s.Fill(ctx, v.Client); err != nil {
		return err
	}
	if _, _, err := resbuilder.BuildResources(vs, s); err != nil {
		return err
	}
	return nil
}
