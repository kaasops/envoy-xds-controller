package config

import (
	"os"
	"testing"
)

func TestGetFeatureFlags(t *testing.T) {
	// Save original environment variables to restore after test
	originalEnableMainBuilder := os.Getenv(EnvEnableMainBuilder)
	originalMainBuilderPercentage := os.Getenv(EnvMainBuilderPercentage)

	// Restore environment variables after test
	defer func() {
		_ = os.Setenv(EnvEnableMainBuilder, originalEnableMainBuilder)
		_ = os.Setenv(EnvMainBuilderPercentage, originalMainBuilderPercentage)
	}()

	tests := []struct {
		name                  string
		enableMainBuilder     string
		mainBuilderPercentage string
		expected              FeatureFlags
	}{
		{
			name:                  "Default values when environment variables not set",
			enableMainBuilder:     "",
			mainBuilderPercentage: "",
			expected: FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 0,
			},
		},
		{
			name:                  "Enable MainBuilder",
			enableMainBuilder:     "true",
			mainBuilderPercentage: "",
			expected: FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 0,
			},
		},
		{
			name:                  "Enable MainBuilder with percentage",
			enableMainBuilder:     "true",
			mainBuilderPercentage: "50",
			expected: FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 50,
			},
		},
		{
			name:                  "Alternative true value",
			enableMainBuilder:     "yes",
			mainBuilderPercentage: "",
			expected: FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 0,
			},
		},
		{
			name:                  "Numeric true value",
			enableMainBuilder:     "1",
			mainBuilderPercentage: "",
			expected: FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 0,
			},
		},
		{
			name:                  "Invalid boolean - should default to false",
			enableMainBuilder:     "invalid",
			mainBuilderPercentage: "",
			expected: FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 0,
			},
		},
		{
			name:                  "Invalid percentage - should default to 0",
			enableMainBuilder:     "true",
			mainBuilderPercentage: "invalid",
			expected: FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables for test case
			_ = os.Setenv(EnvEnableMainBuilder, tc.enableMainBuilder)
			_ = os.Setenv(EnvMainBuilderPercentage, tc.mainBuilderPercentage)

			// Get feature flags
			flags := GetFeatureFlags()

			// Check results
			if flags.EnableMainBuilder != tc.expected.EnableMainBuilder {
				t.Errorf("EnableMainBuilder: got %v, want %v",
					flags.EnableMainBuilder, tc.expected.EnableMainBuilder)
			}

			if flags.MainBuilderPercentage != tc.expected.MainBuilderPercentage {
				t.Errorf("MainBuilderPercentage: got %v, want %v",
					flags.MainBuilderPercentage, tc.expected.MainBuilderPercentage)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		defaultVal bool
		expected   bool
	}{
		{"Empty value", "", true, true},
		{"Empty value with false default", "", false, false},
		{"True", "true", false, true},
		{"False", "false", true, false},
		{"Yes", "yes", false, true},
		{"No", "no", true, false},
		{"1", "1", false, true},
		{"0", "0", true, false},
		{"TRUE (uppercase)", "TRUE", false, true},
		{"FALSE (uppercase)", "FALSE", true, false},
		{"Invalid value with true default", "invalid", true, true},
		{"Invalid value with false default", "invalid", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getBoolEnv("TEST_KEY", tc.defaultVal)

			// For empty value tests, don't set environment variable
			if tc.value != "" {
				_ = os.Setenv("TEST_KEY", tc.value)
				defer func() { _ = os.Unsetenv("TEST_KEY") }()
				result = getBoolEnv("TEST_KEY", tc.defaultVal)
			}

			if result != tc.expected {
				t.Errorf("getBoolEnv(%q, %v) = %v, want %v",
					tc.value, tc.defaultVal, result, tc.expected)
			}
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		defaultVal int
		expected   int
	}{
		{"Empty value", "", 42, 42},
		{"Valid integer", "123", 0, 123},
		{"Negative integer", "-456", 0, -456},
		{"Zero", "0", 42, 0},
		{"Invalid value", "not-an-int", 42, 42},
		{"Float value", "123.45", 42, 42},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getIntEnv("TEST_KEY", tc.defaultVal)

			// For empty value tests, don't set environment variable
			if tc.value != "" {
				_ = os.Setenv("TEST_KEY", tc.value)
				defer func() { _ = os.Unsetenv("TEST_KEY") }()
				result = getIntEnv("TEST_KEY", tc.defaultVal)
			}

			if result != tc.expected {
				t.Errorf("getIntEnv(%q, %v) = %v, want %v",
					tc.value, tc.defaultVal, result, tc.expected)
			}
		})
	}
}
