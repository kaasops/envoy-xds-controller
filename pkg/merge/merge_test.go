package merge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

type TestCase struct {
	A        json.RawMessage
	B        json.RawMessage
	Options  []Opt
	Expected json.RawMessage
}

func TestMergeJSON(t *testing.T) {
	testCases := []TestCase{
		{
			A:        json.RawMessage(`{"a":1}`),
			B:        json.RawMessage(`{"b":2}`),
			Expected: json.RawMessage(`{"a":1,"b":2}`),
		},
		{
			A:        json.RawMessage(`{"a":1}`),
			B:        json.RawMessage(`{"a":2}`),
			Expected: json.RawMessage(`{"a":2}`),
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test"}}`),
			B:        json.RawMessage(`{"a":2}`),
			Expected: json.RawMessage(`{"a":2,"object":{"name":"test"}}`),
		},
		{
			A:        json.RawMessage(`null`),
			B:        json.RawMessage(`{"a":2}`),
			Expected: json.RawMessage(`{"a":2}`),
		},
		{
			A:        json.RawMessage(`{"a":1}`),
			B:        json.RawMessage(`null`),
			Expected: json.RawMessage(`{"a":1}`),
		},
		{
			A:        json.RawMessage(`{"a":[]}`),
			B:        json.RawMessage(`{"a":[1]}`),
			Expected: json.RawMessage(`{"a":[1]}`),
		},
		{
			A:        json.RawMessage(`{"a":[1]}`),
			B:        json.RawMessage(`{"a":[2]}`),
			Expected: json.RawMessage(`{"a":[1,2]}`),
		},
		{
			A:        json.RawMessage(`{"a":{"b":[1]}}`),
			B:        json.RawMessage(`{"a":{"b":[2]}}`),
			Expected: json.RawMessage(`{"a":{"b":[1,2]}}`),
		},
		{
			A:        json.RawMessage(`{"a":{"b":[1]}}`),
			B:        json.RawMessage(`{"a":{"b":[2]}}`),
			Expected: json.RawMessage(`{"a":{"b":[1,2]}}`),
			Options: []Opt{
				{
					Path:      "a.b",
					Operation: OperationMerge,
				},
			},
		},
		{
			A:        json.RawMessage(`{"a":{"b":[1]}}`),
			B:        json.RawMessage(`{"a":{"b":[2]}}`),
			Expected: json.RawMessage(`{"a":{"b":[2]}}`),
			Options: []Opt{
				{
					Path:      "a.b",
					Operation: OperationReplace,
				},
			},
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test"}}`),
			B:        json.RawMessage(`{"a":2,"object":{"age":"10"}}`),
			Expected: json.RawMessage(`{"a":2,"object":{"age":"10","name":"test"}}`),
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test"}}`),
			B:        json.RawMessage(`{"a":2,"object":{"age":"10"}}`),
			Expected: json.RawMessage(`{"a":2,"object":{"age":"10"}}`),
			Options: []Opt{
				{
					Path:      "object",
					Operation: OperationReplace,
				},
			},
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test"},"object2":{"name":"test2"}}`),
			B:        json.RawMessage(`{"a":2,"object":{"age":"10"},"object2":{"age":10}}`),
			Expected: json.RawMessage(`{"a":2,"object":{"age":"10"},"object2":{"age":10,"name":"test2"}}`),
			Options: []Opt{
				{
					Path:      "object",
					Operation: OperationReplace,
				},
			},
		},
		{
			A:        json.RawMessage(`{"object":[{"test":1}]}`),
			B:        json.RawMessage(`{"object":[{"test":3}]}`),
			Expected: json.RawMessage(`{"object":[{"test":1},{"test":3}]}`),
		},
		{
			A:        json.RawMessage(`{"object":[{"test":1}]}`),
			B:        json.RawMessage(`{"object":[{"test":3}]}`),
			Expected: json.RawMessage(`{"object":[{"test":3}]}`),
			Options: []Opt{
				{
					Path:      "object",
					Operation: OperationReplace,
				},
			},
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test"}}`),
			B:        json.RawMessage(`{"object":{"name":"test2"}}`),
			Expected: json.RawMessage(`{"object":{"name":"test2"}}`),
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test"}}`),
			B:        json.RawMessage(`{"object":{"name":"test2"}}`),
			Expected: json.RawMessage(`{"object":{"name":"test2"}}`),
			Options: []Opt{
				{
					Path:      "object",
					Operation: OperationReplace,
				},
			},
		},
		{
			A:        json.RawMessage(`{"a":1,"c":3}`),
			B:        json.RawMessage(`{"b":2}`),
			Expected: json.RawMessage(`{"b":2,"c":3}`),
			Options: []Opt{
				{
					Path:      "a",
					Operation: OperationDelete,
				},
			},
		},
		{
			A:        json.RawMessage(`{"object":{"name":"test","foo":"bar"}}`),
			B:        json.RawMessage(`{"b":2}`),
			Expected: json.RawMessage(`{"b":2,"object":{"name":"test"}}`),
			Options: []Opt{
				{
					Path:      "object.foo",
					Operation: OperationDelete,
				},
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test_%d", i+1), func(t *testing.T) {
			result := JSONRawMessages(testCase.A, testCase.B, testCase.Options)
			if !reflect.DeepEqual(result, testCase.Expected) {
				t.Errorf("Expected %v, got %v", string(testCase.Expected), string(result))
			}
		})
	}
}

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
