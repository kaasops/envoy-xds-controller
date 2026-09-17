package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Runs against the real e2e manifests, so an edit that stops the override from
// matching fails here instead of silently testing the default Envoy image.
func TestOverrideEnvoyImage(t *testing.T) {
	manifests, err := readManifests("envoy")
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(manifests, "image: envoyproxy/envoy:"))

	got, err := overrideEnvoyImage(manifests, "envoyproxy/envoy:contrib-v1.39.1")
	require.NoError(t, err)
	assert.Contains(t, got, "image: envoyproxy/envoy:contrib-v1.39.1\n")
	assert.Equal(t, 1, strings.Count(got, "image: envoyproxy/envoy:"))

	unchanged, err := overrideEnvoyImage(manifests, "")
	require.NoError(t, err)
	assert.Equal(t, manifests, unchanged)

	_, err = overrideEnvoyImage("kind: ConfigMap\n", "envoyproxy/envoy:contrib-v1.39.1")
	assert.Error(t, err)
}

// A matrix job whose ENVOY_IMAGE got lost would otherwise pass on the default image.
func TestEnvoyImageFromEnv(t *testing.T) {
	t.Setenv(envoyImageEnv, "")
	t.Setenv("CI", "")
	image, err := envoyImageFromEnv()
	require.NoError(t, err)
	assert.Empty(t, image)

	t.Setenv("CI", "true")
	_, err = envoyImageFromEnv()
	assert.Error(t, err)

	t.Setenv(envoyImageEnv, "envoyproxy/envoy:contrib-v1.39.1")
	image, err = envoyImageFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "envoyproxy/envoy:contrib-v1.39.1", image)
}
