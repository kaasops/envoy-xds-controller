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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Modifier string

const (
	ModifierMerge   Modifier = "merge"
	ModifierReplace Modifier = "replace"
	ModifierDelete  Modifier = "delete"
)

type TemplateOpts struct {
	Field    string   `json:"field,omitempty"`
	Modifier Modifier `json:"modifier,omitempty"`
}

// VirtualServiceTemplateSpec defines the desired state of VirtualServiceTemplate
type VirtualServiceTemplateSpec struct {
	VirtualServiceCommonSpec `json:",inline"`
	ExtraFields              []*ExtraField `json:"extraFields,omitempty"`
}

type ExtraField struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default,omitempty"`
}

// VirtualServiceTemplateStatus defines the observed state of VirtualServiceTemplate.
type VirtualServiceTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VirtualServiceTemplate is the Schema for the virtualservicetemplates API.
type VirtualServiceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualServiceTemplateSpec   `json:"spec,omitempty"`
	Status VirtualServiceTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualServiceTemplateList contains a list of VirtualServiceTemplate.
type VirtualServiceTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualServiceTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualServiceTemplate{}, &VirtualServiceTemplateList{})
}
