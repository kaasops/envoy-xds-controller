package testing

import (
	"os"
	"strconv"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnableMainBuilder tests the flag-based switching between implementations
func TestEnableMainBuilder(t *testing.T) {
	// Skip test if in short mode
	if testing.Short() {
		t.Skip("Skipping feature flag tests in short mode")
	}

	// Create a test store and setup required resources
	s := CreateTestStore()

	// Create the necessary listeners in the test store
	AddTestListener(s, "test-listener", "default")

	// Add test cluster
	AddTestCluster(s, "test-cluster", "default")

	// Create virtual services for testing
	vs1 := CreateTestVirtualService()
	vs1.ObjectMeta.Name = "test-vs-1"
	vs2 := CreateTestVirtualService()
	vs2.ObjectMeta.Name = "test-vs-2"

	AddTestVirtualService(s, vs1)
	AddTestVirtualService(s, vs2)

	// Create two ResourceBuilder instances to test both implementations
	rbOriginal := resbuilder_v2.NewResourceBuilder(s)
	rbMainBuilder := resbuilder_v2.NewResourceBuilder(s)

	// Enable MainBuilder on one instance
	rbMainBuilder.EnableMainBuilder(true)

	// Both should be able to build resources, even though they use different implementations
	_, err1 := rbOriginal.BuildResources(vs1)
	_, err2 := rbMainBuilder.BuildResources(vs2)

	assert.NoError(t, err1, "Original builder should build resources successfully")
	assert.NoError(t, err2, "MainBuilder should build resources successfully")

	// Now test flipping the flag on a single builder
	rb := resbuilder_v2.NewResourceBuilder(s)

	// Initially it should use the original implementation
	// Then enable MainBuilder
	rb.EnableMainBuilder(true)

	// Then disable MainBuilder
	rb.EnableMainBuilder(false)

	// Both modes should successfully build resources
	_, err3 := rb.BuildResources(vs1)
	assert.NoError(t, err3, "Builder should work after switching implementations")
}

// TestEnvironmentFlagConfiguration tests configuration via environment variables
func TestEnvironmentFlagConfiguration(t *testing.T) {
	// Save original environment variable values to restore later
	originalEnableValue := os.Getenv("ENABLE_MAIN_BUILDER")
	originalPercentageValue := os.Getenv("MAIN_BUILDER_PERCENTAGE")
	defer func() {
		os.Setenv("ENABLE_MAIN_BUILDER", originalEnableValue)
		os.Setenv("MAIN_BUILDER_PERCENTAGE", originalPercentageValue)
	}()

	// Test cases for ENABLE_MAIN_BUILDER
	testCases := []struct {
		name          string
		envValue      string
		expectedValue bool
	}{
		{"Empty", "", false},
		{"True", "true", true},
		{"False", "false", false},
		{"Yes", "yes", true},
		{"No", "no", false},
		{"1", "1", true},
		{"0", "0", false},
		{"Invalid", "invalid", false}, // Invalid should default to false
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("ENABLE_MAIN_BUILDER", tc.envValue)

			// Get feature flags
			flags := config.GetFeatureFlags()

			// Check result
			assert.Equal(t, tc.expectedValue, flags.EnableMainBuilder,
				"Expected EnableMainBuilder to be %v when ENABLE_MAIN_BUILDER=%s",
				tc.expectedValue, tc.envValue)
		})
	}

	// Test cases for MAIN_BUILDER_PERCENTAGE
	percentageCases := []struct {
		name          string
		envValue      string
		expectedValue int
	}{
		{"Empty", "", 0},
		{"Zero", "0", 0},
		{"Fifty", "50", 50},
		{"Hundred", "100", 100},
		{"Negative", "-10", 0},    // Invalid should default to 0
		{"TooLarge", "101", 100},  // Cap at 100
		{"Invalid", "invalid", 0}, // Invalid should default to 0
	}

	for _, tc := range percentageCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("MAIN_BUILDER_PERCENTAGE", tc.envValue)

			// Get feature flags
			flags := config.GetFeatureFlags()

			// Check result
			assert.Equal(t, tc.expectedValue, flags.MainBuilderPercentage,
				"Expected MainBuilderPercentage to be %v when MAIN_BUILDER_PERCENTAGE=%s",
				tc.expectedValue, tc.envValue)
		})
	}
}

// TestGradualRolloutStrategy tests the percentage-based gradual rollout strategy
func TestGradualRolloutStrategy(t *testing.T) {
	// Save original environment variable values to restore later
	originalEnableValue := os.Getenv("ENABLE_MAIN_BUILDER")
	originalPercentageValue := os.Getenv("MAIN_BUILDER_PERCENTAGE")
	defer func() {
		os.Setenv("ENABLE_MAIN_BUILDER", originalEnableValue)
		os.Setenv("MAIN_BUILDER_PERCENTAGE", originalPercentageValue)
	}()

	// Create a test store
	s := CreateTestStore()

	// Create a ResourceBuilder
	rb := resbuilder_v2.NewResourceBuilder(s)

	// Set ENABLE_MAIN_BUILDER=false to test percentage-based rollout
	os.Setenv("ENABLE_MAIN_BUILDER", "false")

	// Test with 0% - should never use MainBuilder
	os.Setenv("MAIN_BUILDER_PERCENTAGE", "0")
	rb.UpdateFeatureFlags()

	for i := 0; i < 100; i++ {
		vs := CreateTestVirtualService()
		vs.Name = vs.Name + "-" + strconv.Itoa(i)
		vs.Namespace = "default"
		AddTestVirtualService(s, vs)

		assert.False(t, config.ShouldUseMainBuilder(config.GetFeatureFlags(), vs.Name+"-"+vs.Namespace),
			"Should not use MainBuilder with 0% rollout")
	}

	// Test with 100% - should always use MainBuilder
	os.Setenv("MAIN_BUILDER_PERCENTAGE", "100")
	rb.UpdateFeatureFlags()

	for i := 0; i < 100; i++ {
		vs := CreateTestVirtualService()
		vs.Name = vs.Name + "-" + strconv.Itoa(i)
		vs.Namespace = "default"
		AddTestVirtualService(s, vs)

		assert.True(t, config.ShouldUseMainBuilder(config.GetFeatureFlags(), vs.Name+"-"+vs.Namespace),
			"Should always use MainBuilder with 100% rollout")
	}

	// Test with 50% - should use MainBuilder for approximately half of the cases
	os.Setenv("MAIN_BUILDER_PERCENTAGE", "50")
	rb.UpdateFeatureFlags()

	usedMainBuilder := 0
	for i := 0; i < 100; i++ {
		vs := CreateTestVirtualService()
		vs.Name = vs.Name + "-" + strconv.Itoa(i)
		vs.Namespace = "default"
		AddTestVirtualService(s, vs)

		if config.ShouldUseMainBuilder(config.GetFeatureFlags(), vs.Name+"-"+vs.Namespace) {
			usedMainBuilder++
		}
	}

	// With 50% and 100 samples, we'd expect around 50 to use MainBuilder
	// But hash-based distribution might not be perfect, so allow some variance
	assert.InDelta(t, 50, usedMainBuilder, 20,
		"Should use MainBuilder for approximately 50%% of cases, got %d%%", usedMainBuilder)
}

// TestErrorPropagation tests that errors are properly propagated in both implementations
func TestErrorPropagation(t *testing.T) {
	// Create a test store
	s := CreateTestStore()

	// Create a VirtualService with an error (missing listener)
	vs := &v1alpha1.VirtualService{
		Spec: v1alpha1.VirtualServiceSpec{
			// No listener specified, which should cause an error
		},
	}

	// Test original implementation
	originalBuilder := resbuilder_v2.NewResourceBuilder(s)
	originalBuilder.EnableMainBuilder(false)

	_, errOriginal := originalBuilder.BuildResources(vs)
	require.Error(t, errOriginal, "Original implementation should return an error for VS with missing listener")

	// Test MainBuilder implementation
	newBuilder := resbuilder_v2.NewResourceBuilder(s)
	newBuilder.EnableMainBuilder(true)

	_, errNew := newBuilder.BuildResources(vs)
	require.Error(t, errNew, "MainBuilder implementation should return an error for VS with missing listener")

	// Both implementations should return similar errors
	// We don't check the exact error message as they might be formatted differently,
	// but they should both be about the missing listener
	assert.Contains(t, errOriginal.Error(), "listener",
		"Original implementation error should mention the listener issue")
	assert.Contains(t, errNew.Error(), "listener",
		"MainBuilder implementation error should mention the listener issue")
}
