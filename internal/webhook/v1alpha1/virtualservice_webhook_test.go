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

func (s *stubUpdater) DryValidateVirtualServiceLight(ctx context.Context, vs *envoyv1alpha1.VirtualService, prevVS *envoyv1alpha1.VirtualService) error {
	return s.lightErr
}

func (s *stubUpdater) DryBuildSnapshotsWithVirtualService(ctx context.Context, vs *envoyv1alpha1.VirtualService) error {
	return s.heavyErr
}

// helper to make minimal VS with nodeIDs annotation
func makeVS(ns, name string, nodeIDs []string) *envoyv1alpha1.VirtualService {
	vs := &envoyv1alpha1.VirtualService{}
	vs.Namespace = ns
	vs.Name = name
	vs.SetAnnotations(map[string]string{})
	vs.SetNodeIDs(nodeIDs)
	return vs
}

func TestVirtualServiceWebhook_HeavyTimeoutFriendlyMessage(t *testing.T) {
	// Ensure light mode is disabled
	t.Setenv("EXC_WEBHOOK_LIGHT_DRYRUN", "")
	// Make timeout small to have deterministic string, though we stub immediately
	t.Setenv("EXC_WEBHOOK_DRYRUN_TIMEOUT_MS", "10")

	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{heavyErr: context.DeadlineExceeded},
	}
	vs := makeVS("ns", "vs1", []string{"n1"})

	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil {
		t.Fatalf("expected error")
	}
	e := err.Error()
	if !containsAll(e, []string{"validation timed out after", "EXC_WEBHOOK_DRYRUN_TIMEOUT_MS"}) {
		t.Fatalf("unexpected error message: %s", e)
	}
}

func TestVirtualServiceWebhook_HeavyGenericError(t *testing.T) {
	t.Setenv("EXC_WEBHOOK_LIGHT_DRYRUN", "")
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{heavyErr: errors.New("boom")},
	}
	vs := makeVS("ns", "vs1", []string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil || !contains(err.Error(), "failed to build snapshot with virtual service: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualServiceWebhook_LightError_Propagates(t *testing.T) {
	t.Setenv("EXC_WEBHOOK_LIGHT_DRYRUN", "1")
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{lightErr: errors.New("light broke")},
	}
	vs := makeVS("ns", "vs1", []string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil || !contains(err.Error(), "light validation failed: light broke") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestVirtualServiceWebhook_LightCoverageMiss_FallbackHeavyTimeout(t *testing.T) {
	t.Setenv("EXC_WEBHOOK_LIGHT_DRYRUN", "1")
	// small timeout for deterministic string
	t.Setenv("EXC_WEBHOOK_DRYRUN_TIMEOUT_MS", "5")
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{lightErr: updater.ErrLightValidationInsufficientCoverage, heavyErr: context.DeadlineExceeded},
	}
	vs := makeVS("ns", "vs1", []string{"n"})
	_, err := v.ValidateCreate(context.Background(), vs)
	if err == nil {
		t.Fatalf("expected timeout after fallback")
	}
	e := err.Error()
	if !containsAll(e, []string{"validation timed out after", "EXC_WEBHOOK_DRYRUN_TIMEOUT_MS"}) {
		t.Fatalf("unexpected error message: %s", e)
	}
}

func TestVirtualServiceWebhook_LightCoverageMiss_FallbackHeavyOK(t *testing.T) {
	t.Setenv("EXC_WEBHOOK_LIGHT_DRYRUN", "1")
	v := &VirtualServiceCustomValidator{
		Client:  nil,
		updater: &stubUpdater{lightErr: updater.ErrLightValidationInsufficientCoverage, heavyErr: nil},
	}
	vs := makeVS("ns", "vs1", []string{"n"})
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
