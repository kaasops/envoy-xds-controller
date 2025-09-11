package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestTracing_UnmarshalV3AndValidate_Positive(t *testing.T) {
	// Valid tracing config with Zipkin provider
	json := []byte(`{
		"provider": {
			"name": "envoy.tracers.zipkin",
			"typed_config": {
				"@type": "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig",
				"collector_cluster": "zipkin",
				"collector_endpoint": "/api/v2/spans"
			}
		}
	}`)

	tr := &Tracing{Spec: &runtime.RawExtension{Raw: json}}
	if _, err := tr.UnmarshalV3AndValidate(); err != nil {
		// ValidateAll may be strict; provide clear output if it fails in CI
		t.Fatalf("expected valid tracing config, got error: %v", err)
	}
}

func TestTracing_UnmarshalV3AndValidate_SpecNil(t *testing.T) {
	tr := &Tracing{Spec: nil}
	if _, err := tr.UnmarshalV3AndValidate(); err == nil {
		t.Fatalf("expected error %v, got nil", ErrSpecNil)
	}
}
