package main

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestSnapshotControlIsOnByDefault(t *testing.T) {
	var cfg Config
	require.NoError(t, envconfig.Process("APP", &cfg))
	require.True(t, cfg.SnapshotControl)
	require.Equal(t, "envoy-xds-controller-frozen-nodes", cfg.pauseSecretName())

	t.Setenv("API_SNAPSHOT_CONTROL", "false")
	t.Setenv("SNAPSHOT_FREEZE_SECRET", "existing-release-secret")
	require.NoError(t, envconfig.Process("APP", &cfg))
	require.Equal(t, "existing-release-secret", cfg.pauseSecretName())
	t.Setenv("SNAPSHOT_PAUSE_SECRET", "release-paused-nodes")
	require.NoError(t, envconfig.Process("APP", &cfg))
	require.False(t, cfg.SnapshotControl)
	require.Equal(t, "release-paused-nodes", cfg.pauseSecretName())
}

func TestSnapshotHistoryIsKeptOnlyForRollbacks(t *testing.T) {
	require.Equal(t, 10, snapshotHistoryLimit(true, true))
	require.Zero(t, snapshotHistoryLimit(true, false), "the cache API cannot roll back")
	require.Zero(t, snapshotHistoryLimit(false, true), "there is no cache API to roll back through")
}
