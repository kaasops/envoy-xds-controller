package v1alpha1

const AnnotationSecretDomains = "envoy.kaasops.io/domains" // TODO: make private, access via getter
const annotationDescription = "envoy.kaasops.io/description"
const GeneralAccessGroup = "general"

type Message string

type ResourceRef struct {
	Name      string  `json:"name,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}
