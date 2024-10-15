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
