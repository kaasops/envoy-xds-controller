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

func TestImageVersionAtLeast(t *testing.T) {
	tests := []struct {
		image string
		want  bool
	}{
		{"envoyproxy/envoy:contrib-v1.34.14", false},
		{"envoyproxy/envoy:contrib-v1.38.0", true},
		{"envoyproxy/envoy:v1.39.1", true},
		{"envoyproxy/envoy:contrib-v2.0.0", true},
		// Registry prefixes, image variants and digests that ENVOY_IMAGE may carry
		{"docker.io/envoyproxy/envoy:contrib-v1.39.1", true},
		{"mirror.local:5000/envoyproxy/envoy:contrib-v1.39.1", true},
		{"envoyproxy/envoy:distroless-v1.37.0", false},
		{"envoyproxy/envoy:contrib-debug-v1.39.1", true},
		{"envoyproxy/envoy:v1.39-latest", true},
		{"envoyproxy/envoy:contrib-v1.39.1@sha256:0123456789abcdef", true},
	}
	for _, tt := range tests {
		got, err := imageVersionAtLeast(tt.image, 1, 38)
		require.NoError(t, err, tt.image)
		assert.Equal(t, tt.want, got, tt.image)
	}

	for _, image := range []string{
		"envoyproxy/envoy:contrib-dev",
		"envoyproxy/envoy@sha256:0123456789abcdef",
		"mirror.local:5000/envoyproxy/envoy",
	} {
		_, err := imageVersionAtLeast(image, 1, 38)
		assert.Error(t, err, image)
	}
}

// Without ENVOY_IMAGE the version comes from the image in the e2e manifests.
func TestEnvoyImage(t *testing.T) {
	manifests, err := readManifests("envoy")
	require.NoError(t, err)

	t.Setenv(envoyImageEnv, "")
	t.Setenv("CI", "")
	image, err := envoyImage(manifests)
	require.NoError(t, err)
	_, err = imageVersionAtLeast(image, 1, 0)
	assert.NoError(t, err, image)

	t.Setenv(envoyImageEnv, "envoyproxy/envoy:contrib-v1.39.1")
	image, err = envoyImage(manifests)
	require.NoError(t, err)
	assert.Equal(t, "envoyproxy/envoy:contrib-v1.39.1", image)
}
