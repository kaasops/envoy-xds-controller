package config

import (
	"testing"
)

func TestShouldUseMainBuilder(t *testing.T) {
	tests := []struct {
		name           string
		flags          FeatureFlags
		namespacedName string
		expected       bool
	}{
		{
			name: "EnableMainBuilder true should always return true",
			flags: FeatureFlags{
				EnableMainBuilder:     true,
				MainBuilderPercentage: 0,
			},
			namespacedName: "default/test-vs",
			expected:       true,
		},
		{
			name: "MainBuilderPercentage 0 should return false",
			flags: FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 0,
			},
			namespacedName: "default/test-vs",
			expected:       false,
		},
		{
			name: "MainBuilderPercentage 100 should return true",
			flags: FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 100,
			},
			namespacedName: "default/test-vs",
			expected:       true,
		},
		{
			name: "MainBuilderPercentage above 100 should return true",
			flags: FeatureFlags{
				EnableMainBuilder:     false,
				MainBuilderPercentage: 110,
			},
			namespacedName: "default/test-vs",
			expected:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ShouldUseMainBuilder(tc.flags, tc.namespacedName)
			if result != tc.expected {
				t.Errorf("ShouldUseMainBuilder(%+v, %q) = %v, want %v",
					tc.flags, tc.namespacedName, result, tc.expected)
			}
		})
	}
}

func TestConsistentHashing(t *testing.T) {
	// Test that the same namespacedName always gets the same hash
	namespacedName := "default/test-vs"
	hash1 := getHash(namespacedName)
	hash2 := getHash(namespacedName)

	if hash1 != hash2 {
		t.Errorf("getHash not consistent: getHash(%q) = %d, %d", namespacedName, hash1, hash2)
	}

	// Test different namespacedNames get different hashes
	namespacedName2 := "default/test-vs-2"
	hash3 := getHash(namespacedName2)

	if hash1 == hash3 {
		t.Errorf("getHash not producing different values: getHash(%q) = getHash(%q) = %d",
			namespacedName, namespacedName2, hash1)
	}
}

func TestPercentageDistribution(t *testing.T) {
	// This test checks that the percentage distribution is roughly correct
	// by generating many namespacedNames and checking the percentage that return true

	percentages := []int{10, 25, 50, 75, 90}

	for _, percentage := range percentages {
		flags := FeatureFlags{
			EnableMainBuilder:     false,
			MainBuilderPercentage: percentage,
		}

		// Generate many namespacedNames and count how many return true
		count := 0
		iterations := 1000

		for i := 0; i < iterations; i++ {
			namespacedName := "default/test-vs-" + string(rune(i))
			if ShouldUseMainBuilder(flags, namespacedName) {
				count++
			}
		}

		// Calculate the actual percentage
		actualPercentage := (count * 100) / iterations

		// Allow for some variance due to the hashing function
		allowedVariance := 5
		if actualPercentage < percentage-allowedVariance || actualPercentage > percentage+allowedVariance {
			t.Errorf("Percentage distribution not within expected range for %d%%: got %d%%, expected within +/-%d%%",
				percentage, actualPercentage, allowedVariance)
		}
	}
}
