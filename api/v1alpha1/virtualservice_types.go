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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualServiceSpec defines the desired state of VirtualService
type VirtualServiceSpec struct {
	VirtualHost *runtime.RawExtension `json:"virtualHost,omitempty"`
	// +kubebuilder:validation:Required
	Listener         *ResourceRef          `json:"listener,omitempty"`
	TlsConfig        *TlsConfig            `json:"tlsConfig,omitempty"`
	AccessLog        *runtime.RawExtension `json:"accessLog,omitempty"`
	AccessLogConfig  *ResourceRef          `json:"accessLogConfig,omitempty"`
	AdditionalRoutes []*ResourceRef        `json:"additionalRoutes,omitempty"`

	// HTTPFilters for use custom HTTP filters
	HTTPFilters []*runtime.RawExtension `json:"httpFilters,omitempty"`
}

type TlsConfig struct {
	CertManager *CertManager `json:"certManager,omitempty"`
	SecretRef   *ResourceRef `json:"secretRef,omitempty"`

	// Find secret with domain in annotation "envoy.kaasops.io/domains"
	AutoDiscovery *bool `json:"autoDiscovery,omitempty"`
}

type CertManager struct {
	// Enabled used if Issuer and ClusterIssuer not set (If you want use default issuer fron ENV)
	// If install Enabled and Issuer or ClusterIssuer - specified issuer will be used
	Enabled *bool `json:"enabled,omitempty"`

	Issuer        *string `json:"issuer,omitempty"`
	ClusterIssuer *string `json:"clusterIssuer,omitempty"`
}

type ResourceRef struct {
	Name string `json:"name,omitempty"`
	// Namespace string `json:"namespace,omitempty"`
}

// VirtualServiceStatus defines the observed state of VirtualService
type VirtualServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Errors          map[string]string `json:"errors,omitempty"`
	LastAppliedHash *uint32           `json:"LastAppliedHash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VirtualService is the Schema for the virtualservices API
type VirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualServiceSpec   `json:"spec,omitempty"`
	Status VirtualServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualServiceList contains a list of VirtualService
type VirtualServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualService{}, &VirtualServiceList{})
}

func (v *VirtualService) GetListener() string {
	return v.Spec.Listener.Name
}

func (v *VirtualService) GetAccessLogConfig() string {
	return v.Spec.AccessLogConfig.Name
}
