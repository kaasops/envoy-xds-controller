package snapshots

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

func TestPausedNodesSurviveInASecret(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	store := &SecretStore{Reader: c, Writer: c, Namespace: "controller", Name: "paused-nodes"}

	paused, floor, err := store.Load(t.Context())
	require.NoError(t, err, "no Secret yet means nothing is paused")
	require.Empty(t, paused)
	require.Zero(t, floor)

	want := map[string]wrapped.Pause{"node-a": {By: "alice", At: now, Reason: "listener switch"}}
	require.NoError(t, store.Save(t.Context(), want, 1763633400))
	require.NoError(t, store.Save(t.Context(), want, 1763633401), "an existing Secret is updated")

	paused, floor, err = store.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, want, paused)
	require.Equal(t, int64(1763633401), floor)

	require.NoError(t, store.Save(t.Context(), map[string]wrapped.Pause{}, 1763633401))
	paused, _, err = store.Load(t.Context())
	require.NoError(t, err)
	require.Empty(t, paused)

	var secret corev1.Secret
	require.NoError(t, c.Get(t.Context(), types.NamespacedName{Namespace: "controller", Name: "paused-nodes"}, &secret))
	require.Contains(t, secret.Data, "frozen.json")
	require.NotContains(t, secret.Labels, "envoy.kaasops.io/secret-type", "not a secret the builder serves")
}

func TestExistingPauseAndVersionFloorSurviveAnUpgrade(t *testing.T) {
	const name = "release-frozen-nodes"
	c := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1ObjectMeta("controller", name),
		Data: map[string][]byte{
			"frozen.json": []byte(`{"node-a":{"by":"alice","at":"2026-11-23T09:00:00Z","reason":"listener switch"}}`),
			"version":     []byte("1763633401"),
		},
	}).Build()
	store := &SecretStore{Reader: c, Writer: c, Namespace: "controller", Name: name}

	paused, floor, err := store.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, wrapped.Pause{By: "alice", At: now, Reason: "listener switch"}, paused["node-a"])
	require.Equal(t, int64(1763633401), floor)
}

func TestUnreadablePauseIsAnError(t *testing.T) {
	c := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1ObjectMeta("controller", "paused-nodes"),
		Data:       map[string][]byte{"frozen.json": []byte("{not json")},
	}).Build()
	store := &SecretStore{Reader: c, Writer: c, Namespace: "controller", Name: "paused-nodes"}

	_, _, err := store.Load(t.Context())
	require.Error(t, err, "starting as if nothing were paused would publish what was held back")
}

func metav1ObjectMeta(namespace, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: namespace, Name: name}
}
