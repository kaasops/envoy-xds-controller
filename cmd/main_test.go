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
	require.Equal(t, "envoy-xds-controller-frozen-nodes", cfg.SnapshotFreezeSecret)

	t.Setenv("API_SNAPSHOT_CONTROL", "false")
	t.Setenv("SNAPSHOT_FREEZE_SECRET", "release-frozen-nodes")
	require.NoError(t, envconfig.Process("APP", &cfg))
	require.False(t, cfg.SnapshotControl)
	require.Equal(t, "release-frozen-nodes", cfg.SnapshotFreezeSecret)
}

func TestSnapshotHistoryIsKeptOnlyForRollbacks(t *testing.T) {
	require.Equal(t, 10, snapshotHistoryLimit(true, true))
	require.Zero(t, snapshotHistoryLimit(true, false), "the cache API cannot roll back")
	require.Zero(t, snapshotHistoryLimit(false, true), "there is no cache API to roll back through")
}
