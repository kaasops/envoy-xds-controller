package updater

import (
	"fmt"
	"os"
	"testing"

	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/config"
	"github.com/stretchr/testify/assert"
)

func TestShouldUseMainBuilderWithConsistentHashing(t *testing.T) {
	tests := []struct {
		name           string
		flags          config.FeatureFlags
		namespacedName string
		expected       bool
	}{
		{
			name: "disabled",
			flags: config.FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 0,
			},
			namespacedName: "default/test-vs",
			expected:       false,
		},
		{
			name: "enabled without percentage",
			flags: config.FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 0,
			},
			namespacedName: "default/test-vs",
			expected:       true,
		},
		{
			name: "enabled with 100 percentage",
			flags: config.FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 100,
			},
			namespacedName: "default/test-vs",
			expected:       true,
		},
		{
			name: "percentage-based rollout without explicit enable",
			flags: config.FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 100, // Use 100% to ensure predictable result
			},
			namespacedName: "default/test-vs",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldUseMainBuilder(tt.flags, tt.namespacedName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConsistentHashingDistribution(t *testing.T) {
	// Test that consistent hashing provides a good distribution
	flags := config.FeatureFlags{
		EnableMainBuilder:     false, // Use percentage-based rollout
		MainBuilderPercentage: 50,
	}

	// Generate many different virtual service names
	trueCount := 0
	totalCount := 1000

	for i := 0; i < totalCount; i++ {
		namespacedName := fmt.Sprintf("namespace-%d/virtualservice-%d", i%10, i)
		if config.ShouldUseMainBuilder(flags, namespacedName) {
			trueCount++
		}
	}

	// Check that distribution is roughly 50% (allow 10% margin for hash distribution)
	percentage := float64(trueCount) / float64(totalCount) * 100
	assert.Greater(t, percentage, 40.0, "Expected at least 40%% true results")
	assert.Less(t, percentage, 60.0, "Expected at most 60%% true results")
}

func TestConsistentHashingStability(t *testing.T) {
	// Test that the same VirtualService always gets the same result
	flags := config.FeatureFlags{
		EnableMainBuilder:     false,
		MainBuilderPercentage: 50,
	}

	namespacedName := "default/my-virtual-service"

	// Call multiple times - should always return the same result
	firstResult := config.ShouldUseMainBuilder(flags, namespacedName)

	for i := 0; i < 100; i++ {
		result := config.ShouldUseMainBuilder(flags, namespacedName)
		assert.Equal(t, firstResult, result, "Result should be consistent for the same VirtualService")
	}
}

func TestFeatureFlagsFromEnvironment(t *testing.T) {
	// Save current env values
	oldEnable := os.Getenv(config.EnvEnableMainBuilder)
	oldPercentage := os.Getenv(config.EnvMainBuilderPercentage)
	defer func() {
		_ = os.Setenv(config.EnvEnableMainBuilder, oldEnable)
		_ = os.Setenv(config.EnvMainBuilderPercentage, oldPercentage)
	}()

	tests := []struct {
		name               string
		enableEnv          string
		percentageEnv      string
		expectedEnable     bool
		expectedPercentage int
	}{
		{
			name:               "both enabled",
			enableEnv:          "true",
			percentageEnv:      "25",
			expectedEnable:     true,
			expectedPercentage: 25,
		},
		{
			name:               "disabled",
			enableEnv:          "false",
			percentageEnv:      "",
			expectedEnable:     false,
			expectedPercentage: 0,
		},
		{
			name:               "invalid percentage defaults to 0",
			enableEnv:          "true",
			percentageEnv:      "invalid",
			expectedEnable:     true,
			expectedPercentage: 0,
		},
		{
			name:               "percentage clamped to 100",
			enableEnv:          "true",
			percentageEnv:      "150",
			expectedEnable:     true,
			expectedPercentage: 100,
		},
		{
			name:               "negative percentage clamped to 0",
			enableEnv:          "true",
			percentageEnv:      "-50",
			expectedEnable:     true,
			expectedPercentage: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(config.EnvEnableMainBuilder, tt.enableEnv)
			_ = os.Setenv(config.EnvMainBuilderPercentage, tt.percentageEnv)

			flags := config.GetFeatureFlags()
			assert.Equal(t, tt.expectedEnable, flags.EnableMainBuilder)
			assert.Equal(t, tt.expectedPercentage, flags.MainBuilderPercentage)
		})
	}
}
