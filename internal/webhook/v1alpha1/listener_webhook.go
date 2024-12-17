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
var listenerlog = logf.Log.WithName("listener-resource")

// SetupListenerWebhookWithManager registers the webhook for Listener in the manager.
func SetupListenerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&envoyv1alpha1.Listener{}).
		WithValidator(&ListenerCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-envoy-kaasops-io-v1alpha1-listener,mutating=false,failurePolicy=fail,sideEffects=None,groups=envoy.kaasops.io,resources=listeners,verbs=create;update;delete,versions=v1alpha1,name=vlistener-v1alpha1.kb.io,admissionReviewVersions=v1

// ListenerCustomValidator struct is responsible for validating the Listener resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ListenerCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &ListenerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Listener.
func (v *ListenerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	listener, ok := obj.(*envoyv1alpha1.Listener)
	if !ok {
		return nil, fmt.Errorf("expected a Listener object but got %T", obj)
	}
	listenerlog.Info("Validation for Listener upon creation", "name", listener.GetName())

	if _, err := listener.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Listener.
func (v *ListenerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	listener, ok := newObj.(*envoyv1alpha1.Listener)
	if !ok {
		return nil, fmt.Errorf("expected a Listener object for the newObj but got %T", newObj)
	}
	listenerlog.Info("Validation for Listener upon update", "name", listener.GetName())

	if _, err := listener.UnmarshalV3AndValidate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Listener.
func (v *ListenerCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	listener, ok := obj.(*envoyv1alpha1.Listener)
	if !ok {
		return nil, fmt.Errorf("expected a Listener object but got %T", obj)
	}
	listenerlog.Info("Validation for Listener upon deletion", "name", listener.GetName())

	// check references virtual services

	var virtualServiceList envoyv1alpha1.VirtualServiceList
	if err := v.Client.List(ctx, &virtualServiceList, client.InNamespace(listener.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualService resources: %w", err)
	}

	if len(virtualServiceList.Items) > 0 {
		var refVsNames []string
		for _, vs := range virtualServiceList.Items {
			if vs.Spec.Listener != nil && vs.Spec.Listener.Name == listener.GetName() {
				refVsNames = append(refVsNames, vs.GetName())
			}
		}
		if len(refVsNames) > 0 {
			return nil, fmt.Errorf("cannot delete Listener %s because it is still referenced by VirtualService(s) %s",
				listener.GetName(),
				refVsNames,
			)
		}
	}

	// check referenced virtual service templates

	var virtualServiceTemplateList envoyv1alpha1.VirtualServiceTemplateList
	if err := v.Client.List(ctx, &virtualServiceTemplateList, client.InNamespace(listener.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServiceTemplate resources: %w", err)
	}

	if len(virtualServiceTemplateList.Items) > 0 {
		var refVstNames []string
		for _, vst := range virtualServiceTemplateList.Items {
			if vst.Spec.Listener != nil && vst.Spec.Listener.Name == listener.GetName() {
				refVstNames = append(refVstNames, vst.GetName())
			}
		}
		if len(refVstNames) > 0 {
			return nil, fmt.Errorf("cannot delete Listener %s because it is still referenced by VirtualServiceTemplate(s) %s",
				listener.GetName(),
				refVstNames,
			)
		}
	}

	return nil, nil
}
