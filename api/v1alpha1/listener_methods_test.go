package v1alpha1

import (
	"testing"

	dynamicmodulesv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/dynamic_modules/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/apimachinery/pkg/runtime"

	// Registers typed_config types the same way the controller binary does.
	_ "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

const testModulePath = "/opt/envoy-modules/libja4plus.so"

// dynamicModuleFilter is implemented by the listener and HTTP dynamic module filter configs.
type dynamicModuleFilter interface {
	GetDynamicModuleConfig() *dynamicmodulesv3.DynamicModuleConfig
}

func dynamicModuleConfig(t *testing.T, typedConfig *anypb.Any) *dynamicmodulesv3.DynamicModuleConfig {
	t.Helper()
	msg, err := typedConfig.UnmarshalNew()
	if err != nil {
		t.Fatalf("failed to unpack typed_config: %v", err)
	}
	filter, ok := msg.(dynamicModuleFilter)
	if !ok {
		t.Fatalf("typed_config %T is not a dynamic module filter", msg)
	}
	return filter.GetDynamicModuleConfig()
}

// The module goes into listener_filters: a Listener CR with filter_chains
// cannot be combined with HTTP filters of a VirtualService.
func TestListener_UnmarshalV3AndValidate_DynamicModule(t *testing.T) {
	tests := []struct {
		name         string
		spec         string
		wantFilename string
		wantName     string
	}{
		{
			name: "module by path",
			spec: `{
				"name": "https",
				"address": {"socket_address": {"address": "0.0.0.0", "port_value": 10443}},
				"listener_filters": [{
					"name": "envoy.filters.listener.dynamic_modules",
					"typed_config": {
						"@type": "type.googleapis.com/envoy.extensions.filters.listener.dynamic_modules.v3.DynamicModuleListenerFilter",
						"dynamic_module_config": {"module": {"local": {"filename": "/opt/envoy-modules/libja4plus.so"}}},
						"filter_name": "conn_probe"
					}
				}]
			}`,
			wantFilename: testModulePath,
		},
		{
			name: "module by name",
			spec: `{
				"name": "https",
				"address": {"socket_address": {"address": "0.0.0.0", "port_value": 10443}},
				"listener_filters": [{
					"name": "envoy.filters.listener.dynamic_modules",
					"typed_config": {
						"@type": "type.googleapis.com/envoy.extensions.filters.listener.dynamic_modules.v3.DynamicModuleListenerFilter",
						"dynamic_module_config": {"name": "ja4plus"},
						"filter_name": "conn_probe"
					}
				}]
			}`,
			wantName: "ja4plus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Listener{Spec: &runtime.RawExtension{Raw: []byte(tt.spec)}}
			if _, err := l.UnmarshalV3AndValidate(); err != nil {
				t.Fatalf("expected listener to be accepted, got error: %v", err)
			}

			listener, err := l.UnmarshalV3()
			if err != nil {
				t.Fatalf("failed to unmarshal listener: %v", err)
			}
			cfg := dynamicModuleConfig(t, listener.GetListenerFilters()[0].GetTypedConfig())
			if got := cfg.GetModule().GetLocal().GetFilename(); got != tt.wantFilename {
				t.Errorf("module.local.filename = %q, want %q", got, tt.wantFilename)
			}
			if got := cfg.GetName(); got != tt.wantName {
				t.Errorf("name = %q, want %q", got, tt.wantName)
			}
		})
	}
}
