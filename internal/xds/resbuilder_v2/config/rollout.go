package config

import (
	"hash/fnv"
	"log"
)

// ShouldUseMainBuilder determines whether to use the MainBuilder implementation
// for a given VirtualService based on feature flags and gradual rollout settings.
// It uses a consistent hashing approach to ensure the same VS always gets the same
// implementation for a given percentage setting.
func ShouldUseMainBuilder(flags FeatureFlags, namespacedName string) bool {
	// If main builder is explicitly enabled, always use it
	if flags.EnableMainBuilder {
		return true
	}
	
	// If percentage is 0 or less, don't use main builder
	if flags.MainBuilderPercentage <= 0 {
		return false
	}
	
	// If percentage is 100 or more, always use main builder
	if flags.MainBuilderPercentage >= 100 {
		return true
	}
	
	// Otherwise, use consistent hashing to determine whether to use main builder
	hash := getHash(namespacedName)
	
	// Convert hash to percentage (0-99)
	percentage := hash % 100
	
	// Log decision for monitoring
	shouldUse := percentage < flags.MainBuilderPercentage
	log.Printf("Rollout decision for %s: hash=%d, percentage=%d, threshold=%d, use=%v",
		namespacedName, hash, percentage, flags.MainBuilderPercentage, shouldUse)
	
	// If hash percentage is less than the configured percentage, use main builder
	return shouldUse
}

// getHash computes a consistent hash for a string
func getHash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() % 100)
}