package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

type VirtualServiceCommonSpec struct {
	VirtualHost *runtime.RawExtension `json:"virtualHost,omitempty"`
	// +kubebuilder:validation:Required
	Listener              *ResourceRef          `json:"listener,omitempty"`
	TlsConfig             *TlsConfig            `json:"tlsConfig,omitempty"`
	AccessLog             *runtime.RawExtension `json:"accessLog,omitempty"`
	AccessLogConfig       *ResourceRef          `json:"accessLogConfig,omitempty"`
	AdditionalHttpFilters []*ResourceRef        `json:"additionalHttpFilters,omitempty"`
	AdditionalRoutes      []*ResourceRef        `json:"additionalRoutes,omitempty"`

	// HTTPFilters for use custom HTTP filters
	HTTPFilters []*runtime.RawExtension `json:"httpFilters,omitempty"`

	// Controller HCM Extentions (https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto)
	// UseRemoteAddress - use remote address for x-forwarded-for header (https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#extensions-filters-network-http-connection-manager-v3-httpconnectionmanager)
	UseRemoteAddress *bool `json:"useRemoteAddress,omitempty"`

	// UpgradeConfigs - https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#envoy-v3-api-msg-extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-upgradeconfig
	UpgradeConfigs []*runtime.RawExtension `json:"upgradeConfigs,omitempty"`
	RBAC           *VirtualServiceRBACSpec `json:"rbac,omitempty"`
}

type TlsConfig struct {
	SecretRef *ResourceRef `json:"secretRef,omitempty"`

	// Find secret with domain in annotation "envoy.kaasops.io/domains"
	AutoDiscovery *bool `json:"autoDiscovery,omitempty"`
}
