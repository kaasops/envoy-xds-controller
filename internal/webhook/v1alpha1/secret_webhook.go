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

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var secretlog = logf.Log.WithName("secret-resource")

// SetupSecretWebhookWithManager registers the webhook for Listener in the manager.
func SetupSecretWebhookWithManager(mgr ctrl.Manager, updater *updater.CacheUpdater) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Secret{}).
		WithValidator(&SecretCustomValidator{Client: mgr.GetClient(), updater: updater}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate--v1-secret,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=secrets,verbs=delete,versions=v1,name=vsecret-v1.kb.io,admissionReviewVersions=v1

// SecretCustomValidator struct is responsible for validating the Listener resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SecretCustomValidator struct {
	Client  client.Client
	updater *updater.CacheUpdater
}

var _ webhook.CustomValidator = &SecretCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Listener.
func (v *SecretCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Listener.
func (v *SecretCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Listener.
func (v *SecretCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("expected a Secret object but got %T", obj)
	}
	secretlog.Info("Validation for Secret upon deletion", "name", secret.GetName())

	usedSecrets := v.updater.GetUsedSecrets()
	if len(usedSecrets) > 0 {
		ns := secret.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		if vs, ok := usedSecrets[helpers.NamespacedName{Name: secret.GetName(), Namespace: ns}]; ok {
			return nil, fmt.Errorf("secret %s is still used in virtual service: %s/%s",
				secret.GetName(),
				vs.Namespace,
				vs.Name,
			)
		}
	}

	return nil, nil
}
