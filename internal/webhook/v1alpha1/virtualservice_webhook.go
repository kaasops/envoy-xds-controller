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
	"errors"
	"fmt"
	"time"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// WebhookConfig holds the webhook-related configuration
type WebhookConfig struct {
	DryRunTimeoutMS   int
	LightDryRun       bool
	ValidationIndices bool
}

// Annotation key for skipping validation (testing only)
const (
	skipValidationAnnotation = "envoy.kaasops.io/skip-validation"
	annotationValueTrue      = "true"
)

// nolint:unused
// log is for logging in this package.
var virtualservicelog = logf.Log.WithName("virtualservice-resource")

// SetupVirtualServiceWebhookWithManager registers the webhook for VirtualService in the manager.
func SetupVirtualServiceWebhookWithManager(mgr ctrl.Manager, cacheUpdater *updater.CacheUpdater, config WebhookConfig) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.VirtualService{}).
		WithValidator(&VirtualServiceCustomValidator{Client: mgr.GetClient(), updater: cacheUpdater, Config: config}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-virtualservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=virtualservices,verbs=create;update,versions=v1alpha1,name=vvirtualservice-v1alpha1.envoy.kaasops.io,admissionReviewVersions=v1

// VirtualServiceCustomValidator struct is responsible for validating the VirtualService resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
// vsUpdater abstracts updater methods used by the webhook for easier testing.
type vsUpdater interface {
	DryValidateVirtualServiceLight(ctx context.Context, vs *envoyv1alpha1.VirtualService, prevVS *envoyv1alpha1.VirtualService, validationIndices bool) error
	DryBuildSnapshotsWithVirtualService(ctx context.Context, vs *envoyv1alpha1.VirtualService) error
}

type VirtualServiceCustomValidator struct {
	Client  client.Client
	updater vsUpdater
	Config  WebhookConfig
}

var _ webhook.CustomValidator = &VirtualServiceCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type VirtualService.
func (v *VirtualServiceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	virtualservice, ok := obj.(*envoyv1alpha1.VirtualService)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualService object but got %T", obj)
	}
	virtualservicelog.Info("Validation for VirtualService upon creation", "name", virtualservice.GetLabelName())

	// Allow skipping validation for testing purposes
	if virtualservice.Annotations != nil {
		if skip, exists := virtualservice.Annotations[skipValidationAnnotation]; exists && skip == annotationValueTrue {
			virtualservicelog.Info("Skipping validation due to annotation", "name", virtualservice.GetLabelName())
			return admission.Warnings{"Validation skipped via annotation - use only for testing"}, nil
		}
	}

	if err := v.validateVirtualService(ctx, virtualservice, nil); err != nil {
		return nil, fmt.Errorf("failed to validate VirtualService %s: %w", virtualservice.Name, err)
	}

	virtualservicelog.Info("VirtualService is valid", "name", virtualservice.GetLabelName())

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type VirtualService.
func (v *VirtualServiceCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	virtualservice, ok := newObj.(*envoyv1alpha1.VirtualService)
	if !ok {
		return nil, fmt.Errorf("expected a VirtualService object for the newObj but got %T", newObj)
	}
	var prevVS *envoyv1alpha1.VirtualService
	// Try to short-circuit heavy validation if spec hasn't changed
	if oldVS, ok := oldObj.(*envoyv1alpha1.VirtualService); ok {
		prevVS = oldVS
		if oldVS.IsEqual(virtualservice) {
			virtualservicelog.Info("Skip validation on update: spec unchanged", "name", virtualservice.GetLabelName())
			observeVSValidation("skipped", "ok", time.Now())
			return nil, nil
		}
	}

	virtualservicelog.Info("Validation for VirtualService upon update", "name", virtualservice.GetLabelName())

	// Allow skipping validation for testing purposes
	if virtualservice.Annotations != nil {
		if skip, exists := virtualservice.Annotations[skipValidationAnnotation]; exists && skip == annotationValueTrue {
			virtualservicelog.Info("Skipping validation due to annotation", "name", virtualservice.GetLabelName())
			return admission.Warnings{"Validation skipped via annotation - use only for testing"}, nil
		}
	}

	if err := v.validateVirtualService(ctx, virtualservice, prevVS); err != nil {
		return nil, fmt.Errorf("failed to validate VirtualService %s: %w", virtualservice.Name, err)
	}

	virtualservicelog.Info("VirtualService is valid", "name", virtualservice.GetLabelName())

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type VirtualService.
func (v *VirtualServiceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *VirtualServiceCustomValidator) validateVirtualService(ctx context.Context, vs *envoyv1alpha1.VirtualService, prevVS *envoyv1alpha1.VirtualService) error {
	if len(vs.GetNodeIDs()) == 0 {
		return fmt.Errorf("nodeIDs is required")
	}

	// Validate tracing fields using a pure helper (XOR + existence check)
	if err := validateVSTracing(ctx, v.Client, vs); err != nil {
		return err
	}

	// Apply timeout for dry-run path (light or heavy)
	ctxTO, cancel := context.WithTimeout(ctx, v.getDryRunTimeout())
	defer cancel()

	// Common timeout error factory to keep message consistent
	timeoutError := func() error {
		return fmt.Errorf("validation timed out after %s; please retry or increase WEBHOOK_DRYRUN_TIMEOUT_MS", v.getDryRunTimeout())
	}

	// Heavy validation runner with unified metrics/logging
	heavy := func(phase string) error {
		start := time.Now()
		if err := v.updater.DryBuildSnapshotsWithVirtualService(ctxTO, vs); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				observeVSValidation(phase, "timeout", start)
				return timeoutError()
			}
			observeVSValidation(phase, "error", start)
			return fmt.Errorf("failed to build snapshot with virtual service: %w", err)
		}
		observeVSValidation(phase, "ok", start)
		return nil
	}

	// Light validation runner that may fallback to heavy
	light := func() error {
		start := time.Now()
		if err := v.updater.DryValidateVirtualServiceLight(ctxTO, vs, prevVS, v.Config.ValidationIndices); err != nil {
			switch {
			case errors.Is(err, updater.ErrLightValidationInsufficientCoverage):
				observeVSValidation("light", "coverage_miss", start)
				incVSValidationFallback()
				virtualservicelog.Info("Light validation insufficient; falling back to heavy dry-run", "name", vs.GetLabelName())
				return heavy("heavy_fallback")
			case errors.Is(err, context.DeadlineExceeded):
				observeVSValidation("light", "timeout", start)
				return timeoutError()
			default:
				observeVSValidation("light", "error", start)
				return fmt.Errorf("light validation failed: %w", err)
			}
		}
		observeVSValidation("light", "ok", start)
		return nil
	}

	// Prefer lightweight validation if enabled, otherwise heavy-only
	if v.getLightDryRunEnabled() {
		return light()
	}
	return heavy("heavy")
}

// validateVSTracing applies XOR rule between inline spec.tracing and spec.tracingRef
// and if tracingRef is provided, verifies that the referenced Tracing exists.
func validateVSTracing(ctx context.Context, cl client.Client, vs *envoyv1alpha1.VirtualService) error {
	if vs == nil {
		return nil
	}

	// Tracing XOR rule: only one of spec.tracing or spec.tracingRef may be set
	if vs.Spec.Tracing != nil && vs.Spec.TracingRef != nil {
		return fmt.Errorf("only one of spec.tracing or spec.tracingRef may be set")
	}

	// If tracingRef is set, ensure referenced Tracing exists (namespace defaults to VS namespace)
	if vs.Spec.TracingRef != nil {
		if vs.Spec.TracingRef.Name == "" {
			return fmt.Errorf("spec.tracingRef.name must not be empty when spec.tracingRef is set")
		}
		ns := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
		var tracing envoyv1alpha1.Tracing
		if err := cl.Get(ctx, types.NamespacedName{Namespace: ns, Name: vs.Spec.TracingRef.Name}, &tracing); err != nil {
			return fmt.Errorf("referenced Tracing %s/%s not found or not accessible: %w", ns, vs.Spec.TracingRef.Name, err)
		}
	}

	return nil
}

// getDryRunTimeout returns the timeout for dry-run validations from Config.
func (v *VirtualServiceCustomValidator) getDryRunTimeout() time.Duration {
	if v.Config.DryRunTimeoutMS > 0 {
		return time.Duration(v.Config.DryRunTimeoutMS) * time.Millisecond
	}
	return time.Duration(1000) * time.Millisecond // default fallback
}

// getLightDryRunEnabled returns true if lightweight validation mode is enabled from Config.
func (v *VirtualServiceCustomValidator) getLightDryRunEnabled() bool {
	return v.Config.LightDryRun
}
