package v1alpha1

const AnnotationSecretDomains = "envoy.kaasops.io/domains"

type Message string

type ResourceRef struct {
	Name      string  `json:"name,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}
