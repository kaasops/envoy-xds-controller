package v1alpha1

import (
	"context"
	"errors"
	"testing"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
)

// stubUpdater implements vsUpdater for tests.
type stubUpdater struct {
	heavyErr error
	lightErr error
}

func (s *stubUpdater) DryValidateVirtualServiceLight(ctx context.Context, vs *envoyv1alpha1.VirtualService, prevVS *envoyv1alpha1.VirtualService, validationIndices bool) error {
	return s.lightErr
}

func (s *stubUpdater) DryBuildSnapshotsWithVirtualService(ctx context.Context, vs *envoyv1alpha1.VirtualService) error {
	return s.heavyErr
}

// helper to make minimal VS with nodeIDs annotation
func makeVS(nodeIDs []string) *envoyv1alpha1.VirtualService {
	vs := &envoyv1alpha1.VirtualService{}
	vs.Namespace = "ns"
	vs.Name = "vs1"
	vs.SetAnnotations(map[string]string{})
	vs.SetNodeIDs(nodeIDs)
	return vs
}

func TestVirtualServiceWebhook_HeavyTimeoutFriendlyMessage(t *testing.T) {
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{heavyErr: context.DeadlineExceeded},
		Config: WebhookConfig{
			DryRunTimeoutMS:   10,
			LightDryRun:       false,
			ValidationIndices: false,
		},
	}
	vs := makeVS([]string{"n1"})

	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil {
		t.Fatalf("expected error")
	}
	e := err.Error()
	if !containsAll(e, []string{"validation timed out after", "WEBHOOK_DRYRUN_TIMEOUT_MS"}) {
		t.Fatalf("unexpected error message: %s", e)
	}
}

func TestVirtualServiceWebhook_HeavyGenericError(t *testing.T) {
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{heavyErr: errors.New("boom")},
		Config: WebhookConfig{
			DryRunTimeoutMS:   800,
			LightDryRun:       false,
			ValidationIndices: false,
		},
	}
	vs := makeVS([]string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil || !contains(err.Error(), "failed to build snapshot with virtual service: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualServiceWebhook_LightError_Propagates(t *testing.T) {
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{lightErr: errors.New("light broke")},
		Config: WebhookConfig{
			DryRunTimeoutMS:   800,
			LightDryRun:       true,
			ValidationIndices: false,
		},
	}
	vs := makeVS([]string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil || !contains(err.Error(), "light validation failed: light broke") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestVirtualServiceWebhook_LightCoverageMiss_FallbackHeavyTimeout(t *testing.T) {
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{lightErr: updater.ErrLightValidationInsufficientCoverage, heavyErr: context.DeadlineExceeded},
		Config: WebhookConfig{
			DryRunTimeoutMS:   5,
			LightDryRun:       true,
			ValidationIndices: false,
		},
	}
	vs := makeVS([]string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil {
		t.Fatalf("expected timeout after fallback")
	}
	e := err.Error()
	if !containsAll(e, []string{"validation timed out after", "WEBHOOK_DRYRUN_TIMEOUT_MS"}) {
		t.Fatalf("unexpected error message: %s", e)
	}
}

func TestVirtualServiceWebhook_LightCoverageMiss_FallbackHeavyOK(t *testing.T) {
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{lightErr: updater.ErrLightValidationInsufficientCoverage, heavyErr: nil},
		Config: WebhookConfig{
			DryRunTimeoutMS:   800,
			LightDryRun:       true,
			ValidationIndices: false,
		},
	}
	vs := makeVS([]string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

// local helpers (duplicated minimal versions to keep imports tidy)
func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (index(s, sub) >= 0)))
}

func index(s, sub string) int {
	n := len(s)
	m := len(sub)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}
