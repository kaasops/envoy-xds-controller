package merge

import (
	"reflect"
	"testing"
)

func TestDeleteKey(t *testing.T) {
	tests := []struct {
		name     string
		inputMap map[string]any
		key      string
		expected map[string]any
	}{
		{
			name: "Delete nested key",
			inputMap: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"city": "Wonderland", "street": "Elm"},
				},
				"score": 100,
			},
			key: "user.address.city",
			expected: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"street": "Elm"},
				},
				"score": 100,
			},
		},
		{
			name: "Delete top-level key",
			inputMap: map[string]any{
				"user":  map[string]any{"name": "Alice"},
				"score": 100,
			},
			key: "score",
			expected: map[string]any{
				"user": map[string]any{"name": "Alice"},
			},
		},
		{
			name: "Delete last nested key",
			inputMap: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"city": "Wonderland"},
				},
			},
			key: "user.address.city",
			expected: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{},
				},
			},
		},
		{
			name: "Delete non-existent key",
			inputMap: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"city": "Wonderland", "street": "Elm"},
				},
				"score": 100,
			},
			key: "user.phone",
			expected: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"city": "Wonderland", "street": "Elm"},
				},
				"score": 100,
			},
		},
		{
			name: "Delete with empty key",
			inputMap: map[string]any{
				"user":  map[string]any{"name": "Alice"},
				"score": 100,
			},
			key: "",
			expected: map[string]any{
				"user":  map[string]any{"name": "Alice"},
				"score": 100,
			},
		},
		{
			name: "Delete non-existent key from array",
			inputMap: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"city": "Wonderland", "street": "Elm"},
					"items": []map[string]any{
						{
							"name": "Item 1",
						},
					},
				},
				"score": 100,
			},
			key: "user.items.name",
			expected: map[string]any{
				"user": map[string]any{
					"name":    "Alice",
					"address": map[string]any{"city": "Wonderland", "street": "Elm"},
					"items": []map[string]any{
						{
							"name": "Item 1",
						},
					},
				},
				"score": 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteKey(tt.inputMap, tt.key)

			if !reflect.DeepEqual(tt.inputMap, tt.expected) {
				t.Errorf("Expected: %v, Got: %v", tt.expected, tt.inputMap)
			}
		})
	}
}
