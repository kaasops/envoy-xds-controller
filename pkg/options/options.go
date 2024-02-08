package options

import "google.golang.org/protobuf/encoding/protojson"

const (
	DefaultListenerName         = "https"
	VirtualServiceListenerFeild = "spec.listener.name"
	NodeIDAnnotation            = "envoy.kaasops.io/node-id"
	SecretLabelKey              = "envoy.kaasops.io/secret-type"
	SdsSecretLabelValue         = "sds-cached"
	WebhookSecretLabelValue     = "webhook"

	AutoDiscoveryLabel = "envoy.kaasops.io/autoDiscovery"
	DomainAnnotation   = "envoy.kaasops.io/domains"
)

var (
	Unmarshaler = protojson.UnmarshalOptions{
		AllowPartial: false,
		// DiscardUnknown: true,
	}
)
