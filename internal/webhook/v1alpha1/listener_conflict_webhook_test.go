package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

// conflictOnNodeA returns the dry-run error for node-a whose listeners https and https-module share 0.0.0.0:443.
func conflictOnNodeA(t *testing.T) error {
	t.Helper()
	listener := func(name string) *listenerv3.Listener {
		return &listenerv3.Listener{
			Name: name,
			Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{
				Address:       "0.0.0.0",
				PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 443},
			}}},
		}
	}
	snapshot, err := gcp.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.ListenerType: {listener("ns/https"), listener("ns/https-module")},
	})
	require.NoError(t, err)
	err = wrapped.NewSnapshotCache().SetSnapshot(t.Context(), "node-a", snapshot)
	require.Error(t, err)
	return multierr.Combine(err)
}

// recordingUpdater is a vsUpdater that records which dry run the webhook chose.
type recordingUpdater struct {
	heavyErr, lightErr, listenerErr error
	heavyCalls                      int
	lightTolerated, listenerChecked [][]string
	// slow makes the full dry run return only when its context is done
	slow bool
	// frozen are the nodes frozen on their served snapshot
	frozen []string
}

func (r *recordingUpdater) FrozenNodeIDs() []string { return r.frozen }

func (r *recordingUpdater) DryValidateListenerAddresses(
	_ context.Context,
	_ *envoyv1alpha1.VirtualService,
	_ bool,
	toleratedNodeIDs []string,
) error {
	r.listenerChecked = append(r.listenerChecked, toleratedNodeIDs)
	return r.listenerErr
}

func (r *recordingUpdater) DryValidateVirtualServiceLight(
	_ context.Context,
	_ *envoyv1alpha1.VirtualService,
	_ *envoyv1alpha1.VirtualService,
	_ bool,
	toleratedNodeIDs []string,
) error {
	r.lightTolerated = append(r.lightTolerated, toleratedNodeIDs)
	return r.lightErr
}

func (r *recordingUpdater) DryBuildSnapshotsWithVirtualService(
	ctx context.Context,
	_ *envoyv1alpha1.VirtualService,
) error {
	r.heavyCalls++
	if r.slow {
		<-ctx.Done()
	}
	return r.heavyErr
}

const conflictWarning = `node "node-a" gets no new configuration while ` +
	`listeners ns/https and ns/https-module share 0.0.0.0:443`

func TestVirtualServiceWebhookToleratesListenerConflict(t *testing.T) {
	tests := []struct {
		name      string
		tolerated []string
		heavyErr  func(t *testing.T) error
		wantErr   string
	}{
		{
			name:     "rejected when no node is frozen",
			heavyErr: func(t *testing.T) error { return conflictOnNodeA(t) },
			wantErr:  "duplicate address",
		},
		{
			name:      "accepted with a warning on a frozen node",
			tolerated: []string{"node-a"},
			heavyErr:  func(t *testing.T) error { return conflictOnNodeA(t) },
		},
		{
			name:      "rejected on a node that is not frozen",
			tolerated: []string{"node-b"},
			heavyErr:  func(t *testing.T) error { return conflictOnNodeA(t) },
			wantErr:   "duplicate address",
		},
		{
			name:      "rejected when another error comes with it",
			tolerated: []string{"node-a"},
			heavyErr: func(t *testing.T) error {
				return multierr.Combine(conflictOnNodeA(t), errors.New("missing dependency"))
			},
			wantErr: "missing dependency",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &VirtualServiceCustomValidator{
				updater: &recordingUpdater{heavyErr: tt.heavyErr(t), frozen: tt.tolerated},
			}
			old, updated := makeVS([]string{"node-a"}), makeVS([]string{"node-a"})
			updated.Spec.Listener = &envoyv1alpha1.ResourceRef{Name: "https-module"}

			for op, validate := range map[string]func() (admission.Warnings, error){
				"create": func() (admission.Warnings, error) { return v.ValidateCreate(t.Context(), updated) },
				"update": func() (admission.Warnings, error) { return v.ValidateUpdate(t.Context(), old, updated) },
			} {
				warnings, err := validate()
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr, op)
					require.Empty(t, warnings, op)
					continue
				}
				require.NoError(t, err, op)
				require.Equal(t, admission.Warnings{conflictWarning}, warnings, op)
			}
		})
	}
}

func TestLightModeChecksFrozenNodesWithFullDryRun(t *testing.T) {
	r := &recordingUpdater{heavyErr: conflictOnNodeA(t), frozen: []string{"node-a"}}
	v := &VirtualServiceCustomValidator{
		updater: r,
		Config:  WebhookConfig{LightDryRun: true},
	}

	warnings, err := v.ValidateCreate(t.Context(), makeVS([]string{"node-a"}))

	require.NoError(t, err)
	require.Equal(t, admission.Warnings{conflictWarning}, warnings)
	require.Equal(t, 1, r.heavyCalls)
	require.Empty(t, r.lightTolerated, "light check cannot see listeners from templates")
	require.Equal(t, [][]string{{"node-a"}}, r.listenerChecked, "listener checks of the light mode stay")
}

func TestLightModeKeepsListenerChecksOfFrozenNodes(t *testing.T) {
	r := &recordingUpdater{
		listenerErr: errors.New("listener ns/bare has no address configured"),
		frozen:      []string{"node-a"},
	}
	v := &VirtualServiceCustomValidator{
		updater: r,
		Config:  WebhookConfig{LightDryRun: true},
	}

	warnings, err := v.ValidateCreate(t.Context(), makeVS([]string{"node-a"}))

	require.ErrorContains(t, err, "has no address configured")
	require.Empty(t, warnings)
	require.Zero(t, r.heavyCalls)
}

func TestLightModePassesFrozenNodesToOtherServices(t *testing.T) {
	r := &recordingUpdater{frozen: []string{"node-a"}}
	v := &VirtualServiceCustomValidator{
		updater: r,
		Config:  WebhookConfig{LightDryRun: true},
	}

	warnings, err := v.ValidateCreate(t.Context(), makeVS([]string{"node-b"}))

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Zero(t, r.heavyCalls)
	require.Equal(t, [][]string{{"node-a"}}, r.lightTolerated)
}

// stubTemplateUpdater is a vstUpdater returning a fixed dry-run error.
type stubTemplateUpdater struct {
	err    error
	frozen []string
}

func (s stubTemplateUpdater) FrozenNodeIDs() []string { return s.frozen }

func (s stubTemplateUpdater) DryBuildSnapshotsWithVirtualServiceTemplate(
	context.Context,
	*envoyv1alpha1.VirtualServiceTemplate,
) error {
	return s.err
}

func TestTemplateWebhookToleratesListenerConflict(t *testing.T) {
	template := &envoyv1alpha1.VirtualServiceTemplate{}
	template.Namespace, template.Name = "ns", "shared"
	for _, tc := range []struct {
		tolerated []string
		wantErr   bool
	}{{nil, true}, {[]string{"node-b"}, true}, {[]string{"node-a"}, false}} {
		t.Run(fmt.Sprint(tc.tolerated), func(t *testing.T) {
			v := &VirtualServiceTemplateCustomValidator{
				cacheUpdater: stubTemplateUpdater{err: conflictOnNodeA(t), frozen: tc.tolerated},
			}

			warnings, err := v.ValidateUpdate(t.Context(), template, template)

			if tc.wantErr {
				require.ErrorContains(t, err, "duplicate address")
				require.Empty(t, warnings)
				return
			}
			require.NoError(t, err)
			require.Equal(t, admission.Warnings{conflictWarning}, warnings)
		})
	}
}

func TestLightModeChecksServicesLeavingFrozenNodesInFull(t *testing.T) {
	r := &recordingUpdater{heavyErr: conflictOnNodeA(t), frozen: []string{"node-a"}}
	v := &VirtualServiceCustomValidator{
		updater: r,
		Config:  WebhookConfig{LightDryRun: true},
	}
	old, moved := makeVS([]string{"node-a"}), makeVS([]string{"node-b"})

	warnings, err := v.ValidateUpdate(t.Context(), old, moved)

	require.NoError(t, err)
	require.Equal(t, 1, r.heavyCalls, "the held node loses a service, the light check cannot see it")
	require.Equal(t, admission.Warnings{conflictWarning}, warnings)
}

func TestToleratedConflictAfterTimeoutIsATimeout(t *testing.T) {
	v := &VirtualServiceCustomValidator{
		updater: &recordingUpdater{heavyErr: conflictOnNodeA(t), slow: true, frozen: []string{"node-a"}},
		Config:  WebhookConfig{DryRunTimeoutMS: 10},
	}

	warnings, err := v.ValidateCreate(t.Context(), makeVS([]string{"node-a"}))

	require.ErrorContains(t, err, "validation timed out")
	require.Empty(t, warnings)
}

func TestWarningHidesListenersOfOtherNamespaces(t *testing.T) {
	warnings := toleratedListenerConflict(conflictOnNodeA(t), []string{"node-a"}, "team")

	require.Equal(t, admission.Warnings{
		`node "node-a" gets no new configuration while two of its listeners share an address`,
	}, warnings)
}

// A frozen node gets nothing until it is unfrozen, so a switch on it needs no setting in the chart.
func TestFrozenNodesTolerateListenerConflicts(t *testing.T) {
	r := &recordingUpdater{heavyErr: conflictOnNodeA(t), frozen: []string{"node-a"}}
	v := &VirtualServiceCustomValidator{updater: r, Config: WebhookConfig{LightDryRun: true}}

	warnings, err := v.ValidateCreate(t.Context(), makeVS([]string{"node-a"}))

	require.NoError(t, err)
	require.Equal(t, admission.Warnings{conflictWarning}, warnings)
	require.Equal(t, 1, r.heavyCalls, "a service of a frozen node takes the full dry run")

	template := &envoyv1alpha1.VirtualServiceTemplate{}
	template.Namespace, template.Name = "ns", "shared"
	tv := &VirtualServiceTemplateCustomValidator{
		cacheUpdater: stubTemplateUpdater{err: conflictOnNodeA(t), frozen: []string{"node-a"}},
	}
	warnings, err = tv.ValidateUpdate(t.Context(), template, template)
	require.NoError(t, err)
	require.Equal(t, admission.Warnings{conflictWarning}, warnings)
}
