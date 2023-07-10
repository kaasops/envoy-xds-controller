package xds

import "github.com/kaasops/envoy-xds-controller/api/v1alpha1"

const (
	DefaultListenerName = "defaultHTTPS"
	ControllerNamespace = "xds-controller"
)

var (
	DefaultListener = v1alpha1.ResourceRef{
		Name:      DefaultListenerName,
		Namespace: ControllerNamespace,
	}
)
