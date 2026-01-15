package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindClusterNames(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		clusterField string
		expected     []string
	}{
		{
			name:         "nil data",
			data:         nil,
			clusterField: "cluster",
			expected:     nil,
		},
		{
			name:         "empty map",
			data:         map[string]interface{}{},
			clusterField: "cluster",
			expected:     nil,
		},
		{
			name: "simple cluster field",
			data: map[string]interface{}{
				"cluster": "my-cluster",
			},
			clusterField: "cluster",
			expected:     []string{"my-cluster"},
		},
		{
			name: "nested cluster field",
			data: map[string]interface{}{
				"config": map[string]interface{}{
					"cluster": "nested-cluster",
				},
			},
			clusterField: "cluster",
			expected:     []string{"nested-cluster"},
		},
		{
			name: "multiple clusters at different depths",
			data: map[string]interface{}{
				"cluster": "top-level",
				"config": map[string]interface{}{
					"cluster": "nested",
					"inner": map[string]interface{}{
						"cluster": "deep-nested",
					},
				},
			},
			clusterField: "cluster",
			expected:     []string{"top-level", "nested", "deep-nested"},
		},
		{
			name: "cluster in array",
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"cluster": "cluster-1"},
					map[string]interface{}{"cluster": "cluster-2"},
				},
			},
			clusterField: "cluster",
			expected:     []string{"cluster-1", "cluster-2"},
		},
		{
			name: "empty cluster value is skipped",
			data: map[string]interface{}{
				"cluster": "",
			},
			clusterField: "cluster",
			expected:     nil,
		},
		{
			name: "non-string cluster value is skipped",
			data: map[string]interface{}{
				"cluster": 123,
			},
			clusterField: "cluster",
			expected:     nil,
		},
		{
			name: "different field name",
			data: map[string]interface{}{
				"cluster_name": "my-cluster-name",
			},
			clusterField: "cluster_name",
			expected:     []string{"my-cluster-name"},
		},
		{
			name: "collector_cluster field",
			data: map[string]interface{}{
				"tracing": map[string]interface{}{
					"provider": map[string]interface{}{
						"collector_cluster": "jaeger-cluster",
					},
				},
			},
			clusterField: "collector_cluster",
			expected:     []string{"jaeger-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindClusterNames(tt.data, tt.clusterField)
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.ElementsMatch(t, tt.expected, result)
			}
		})
	}
}

func TestFindClusterNames_MaxDepthProtection(t *testing.T) {
	// Create a deeply nested structure that exceeds maxRecursionDepth
	var deeplyNested interface{}
	deeplyNested = map[string]interface{}{
		"cluster": "should-find-this",
	}

	// Wrap it maxRecursionDepth + 10 times
	for i := 0; i < maxRecursionDepth+10; i++ {
		deeplyNested = map[string]interface{}{
			"level": deeplyNested,
		}
	}

	// The function should not crash and should return without finding
	// the deeply nested cluster (it's beyond maxRecursionDepth)
	result := FindClusterNames(deeplyNested, "cluster")
	assert.Empty(t, result, "Should not find clusters beyond max recursion depth")
}

func TestFindClusterNames_AtMaxDepth(t *testing.T) {
	// Create a structure where the cluster is at exactly depth = maxRecursionDepth
	// The function should NOT find it because we stop at depth >= maxRecursionDepth
	var nested interface{}
	nested = map[string]interface{}{
		"cluster": "at-max-depth",
	}

	// Wrap it maxRecursionDepth times (so cluster is at depth maxRecursionDepth)
	// depth starts at 0, so we need maxRecursionDepth levels of wrapping
	for i := 0; i < maxRecursionDepth; i++ {
		nested = map[string]interface{}{
			"level": nested,
		}
	}

	// The function should not find clusters at exactly maxRecursionDepth
	// because we stop at depth >= maxRecursionDepth
	result := FindClusterNames(nested, "cluster")
	assert.Empty(t, result, "Should not find clusters at max depth boundary")
}

func TestFindClusterNames_BelowMaxDepth(t *testing.T) {
	// Create a structure that is well below maxRecursionDepth
	nested := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"cluster": "found-at-depth-4",
				},
			},
		},
	}

	result := FindClusterNames(nested, "cluster")
	assert.Equal(t, []string{"found-at-depth-4"}, result)
}

func TestGetWildcardDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{
			name:     "standard domain",
			domain:   "api.example.com",
			expected: "*.example.com",
		},
		{
			name:     "subdomain",
			domain:   "www.api.example.com",
			expected: "*.api.example.com",
		},
		{
			name:     "single part domain",
			domain:   "localhost",
			expected: "",
		},
		{
			name:     "two part domain",
			domain:   "example.com",
			expected: "*.com",
		},
		{
			name:     "empty domain",
			domain:   "",
			expected: "",
		},
		// Edge cases
		{
			name:     "trailing dot",
			domain:   "api.example.com.",
			expected: "*.example.com.",
		},
		{
			name:     "leading dot",
			domain:   ".example.com",
			expected: "*.example.com",
		},
		{
			name:     "uppercase domain preserved",
			domain:   "API.EXAMPLE.COM",
			expected: "*.EXAMPLE.COM",
		},
		{
			name:     "mixed case domain preserved",
			domain:   "Api.Example.Com",
			expected: "*.Example.Com",
		},
		{
			name:     "only dots - invalid domain rejected",
			domain:   "...",
			expected: "",
		},
		{
			name:     "single dot - invalid domain rejected",
			domain:   ".",
			expected: "",
		},
		{
			name:     "double dot - invalid domain rejected",
			domain:   "..",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWildcardDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckAllDomainsUnique(t *testing.T) {
	tests := []struct {
		name      string
		domains   []string
		expectErr bool
	}{
		{
			name:      "nil domains",
			domains:   nil,
			expectErr: false,
		},
		{
			name:      "empty domains",
			domains:   []string{},
			expectErr: false,
		},
		{
			name:      "single domain",
			domains:   []string{"example.com"},
			expectErr: false,
		},
		{
			name:      "unique domains",
			domains:   []string{"example.com", "api.example.com", "test.com"},
			expectErr: false,
		},
		{
			name:      "duplicate domains",
			domains:   []string{"example.com", "api.example.com", "example.com"},
			expectErr: true,
		},
		{
			name:      "empty strings are skipped",
			domains:   []string{"example.com", "", "", "api.example.com"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckAllDomainsUnique(tt.domains)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
