package filters

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

// RBAC policies are a map. Packed without deterministic marshaling, they come out in another order on
// every build of the same service, and Envoy is sent a changed listener each time.
func TestRBACFilterKeepsItsBytes(t *testing.T) {
	policies := map[string]*runtime.RawExtension{}
	for i := range 8 {
		policies[fmt.Sprintf("policy-%d", i)] = &runtime.RawExtension{Raw: []byte(fmt.Sprintf(
			`{"permissions": [{"any": true}], "principals": [{"remote_ip": {"address_prefix": "10.0.%d.0", "prefix_len": 24}}]}`,
			i))}
	}
	vs := &v1alpha1.VirtualService{Spec: v1alpha1.VirtualServiceSpec{
		VirtualServiceCommonSpec: v1alpha1.VirtualServiceCommonSpec{
			RBAC: &v1alpha1.VirtualServiceRBACSpec{Action: "ALLOW", Policies: policies},
		},
	}}
	build := func() []byte {
		// a fresh cache: the shared one would hand back the bytes of the first build
		b := &Builder{cache: NewHTTPFilterLRUCache(5, 0)}
		filters, err := b.BuildHTTPFilters(vs)
		require.NoError(t, err)
		require.NotEmpty(t, filters)
		return filters[0].GetTypedConfig().GetValue()
	}

	first := build()
	for range 20 {
		require.Equal(t, first, build())
	}
}
