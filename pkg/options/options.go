package options

import "google.golang.org/protobuf/encoding/protojson"

const (
	DefaultListenerName                  = "https"
	VirtualServiceListenerNameFeild      = "spec.listener.name"
	VirtualServiceListenerNamespaceFeild = "spec.listener.namespace"
	NodeIDAnnotation                     = "envoy.kaasops.io/node-id"
	SecretLabelKey                       = "envoy.kaasops.io/secret-type"
	SdsSecretLabelValue                  = "sds-cached"
	WebhookSecretLabelValue              = "webhook"

	AutoDiscoveryLabel = "envoy.kaasops.io/autoDiscovery"
	DomainAnnotation   = "envoy.kaasops.io/domains"
)

var (
	Unmarshaler = protojson.UnmarshalOptions{
		AllowPartial: false,
		// DiscardUnknown: true,
	}
)
