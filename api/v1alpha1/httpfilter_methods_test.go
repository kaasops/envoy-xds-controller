package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	// Registers typed_config types the same way the controller binary does.
	_ "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

func TestHttpFilter_UnmarshalV3AndValidate_DynamicModule(t *testing.T) {
	tests := []struct {
		name         string
		spec         string
		wantFilename string
		wantName     string
	}{
		{
			name: "module by path",
			spec: `{
				"name": "envoy.filters.http.dynamic_modules",
				"typed_config": {
					"@type": "type.googleapis.com/envoy.extensions.filters.http.dynamic_modules.v3.DynamicModuleFilter",
					"dynamic_module_config": {"module": {"local": {"filename": "/opt/envoy-modules/libja4plus.so"}}},
					"filter_name": "ja4h_metrics"
				}
			}`,
			wantFilename: testModulePath,
		},
		{
			name: "module by name",
			spec: `{
				"name": "envoy.filters.http.dynamic_modules",
				"typed_config": {
					"@type": "type.googleapis.com/envoy.extensions.filters.http.dynamic_modules.v3.DynamicModuleFilter",
					"dynamic_module_config": {"name": "ja4plus"},
					"filter_name": "ja4h_metrics"
				}
			}`,
			wantName: "ja4plus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hf := &HttpFilter{Spec: []*runtime.RawExtension{{Raw: []byte(tt.spec)}}}
			filters, err := hf.UnmarshalV3AndValidate()
			if err != nil {
				t.Fatalf("expected http filter to be accepted, got error: %v", err)
			}

			cfg := dynamicModuleConfig(t, filters[0].GetTypedConfig())
			if got := cfg.GetModule().GetLocal().GetFilename(); got != tt.wantFilename {
				t.Errorf("module.local.filename = %q, want %q", got, tt.wantFilename)
			}
			if got := cfg.GetName(); got != tt.wantName {
				t.Errorf("name = %q, want %q", got, tt.wantName)
			}
		})
	}
}
