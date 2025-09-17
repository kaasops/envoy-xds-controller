package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	// EnvEnableMainBuilder is the environment variable name for enabling the MainBuilder implementation
	EnvEnableMainBuilder = "ENABLE_MAIN_BUILDER"

	// EnvMainBuilderPercentage is the environment variable name for controlling the percentage of requests
	// that should use the MainBuilder implementation
	EnvMainBuilderPercentage = "MAIN_BUILDER_PERCENTAGE"
)

// FeatureFlags contains all feature flag settings for ResBuilder
type FeatureFlags struct {
	// EnableMainBuilder indicates whether to use the MainBuilder implementation
	EnableMainBuilder bool

	// MainBuilderPercentage controls what percentage of requests should use MainBuilder
	// when gradual rollout is enabled (value between 0-100)
	MainBuilderPercentage int
}

// GetFeatureFlags returns the feature flag configuration based on environment variables
func GetFeatureFlags() FeatureFlags {
	return FeatureFlags{
		EnableMainBuilder:     getBoolEnv(EnvEnableMainBuilder, false),
		MainBuilderPercentage: getIntEnv(EnvMainBuilderPercentage, 0),
	}
}

// getBoolEnv reads a boolean value from an environment variable
// Returns the default value if the environment variable is not set or has an invalid value
func getBoolEnv(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}

	// Check for true/false strings
	valLower := strings.ToLower(val)
	if valLower == "true" || valLower == "yes" || valLower == "1" {
		return true
	}
	if valLower == "false" || valLower == "no" || valLower == "0" {
		return false
	}

	// For any other value, return the default
	return defaultVal
}

// getIntEnv reads an integer value from an environment variable
// Returns the default value if the environment variable is not set or has an invalid value
func getIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return intVal
}