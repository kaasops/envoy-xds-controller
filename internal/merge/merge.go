package merge

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type OperationType string

const (
	OperationMerge   OperationType = "merge"
	OperationReplace OperationType = "replace"
	OperationDelete  OperationType = "delete"
)

type Opt struct {
	Path      string
	Operation OperationType
}

type parsedOpts struct {
	replace map[string]struct{}
	delete  map[string]struct{}
}

func parseOpts(opts []Opt) *parsedOpts {
	o := &parsedOpts{
		replace: make(map[string]struct{}),
		delete:  make(map[string]struct{}),
	}
	for _, opt := range opts {
		if opt.Operation == OperationReplace {
			o.replace[opt.Path] = struct{}{}
		} else if opt.Operation == OperationDelete {
			o.delete[opt.Path] = struct{}{}
		}
	}
	return o
}

func JSONRawMessages(a, b json.RawMessage, opts []Opt) json.RawMessage {
	mapA, mapB := parseJSON(a), parseJSON(b)
	optsMap := make(map[string]OperationType, len(opts))
	for _, opt := range opts {
		optsMap[opt.Path] = opt.Operation
	}
	result := mergeMaps(mapA, mapB, parseOpts(opts), "")
	mergedJSON, _ := json.Marshal(result)
	return mergedJSON
}

func parseJSON(data json.RawMessage) map[string]any {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil && !errors.Is(err, io.EOF) {
		return make(map[string]any)
	}
	return result
}

func mergeMaps(a, b map[string]any, opts *parsedOpts, currentPath string) map[string]any {
	result := make(map[string]any, len(a)+len(b))

	for k, v := range a {
		result[k] = v
	}

	for k, v := range b {
		keyPath := buildPath(currentPath, k)
		if existingValue, exists := result[k]; exists {
			switch newVal := v.(type) {
			case map[string]any:
				if existingMap, ok := existingValue.(map[string]any); ok {
					if _, ok := opts.replace[keyPath]; ok {
						result[k] = newVal
					} else {
						result[k] = mergeMaps(existingMap, newVal, opts, keyPath)
					}
				} else {
					result[k] = v
				}
			case []any:
				if existingArray, ok := existingValue.([]any); ok {
					result[k] = mergeArrays(existingArray, newVal, opts, keyPath)
				} else {
					result[k] = v
				}
			default:
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	for key := range opts.delete {
		deleteKey(result, key)
	}

	return result
}

// mergeArrays combines two arrays while preserving the uniqueness of elements.
// It handles replacement based on the provided path in opting.
// Any marshaling errors will result in skipping the problematic element.
func mergeArrays(a, b []any, opts *parsedOpts, path string) []any {
	// If a replacement flag is set for the current path, return array b as is
	if _, ok := opts.replace[path]; ok {
		return b
	}

	// If one of the arrays is nil, return the other one
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Create a map for fast lookup of existing elements
	exists := make(map[string]struct{}, len(a))
	for _, itemA := range a {
		if jsonA, err := json.Marshal(itemA); err == nil {
			exists[string(jsonA)] = struct{}{}
		}
	}

	// Initialize the result slice with optimal capacity
	result := make([]any, len(a), len(a)+len(b))
	copy(result, a)

	// Add unique elements from array b
	for _, itemB := range b {
		if jsonB, err := json.Marshal(itemB); err == nil {
			if _, found := exists[string(jsonB)]; !found {
				result = append(result, itemB)
			}
		}
	}

	return result
}

func buildPath(currentPath, newSegment string) string {
	if currentPath == "" {
		return newSegment
	}
	return currentPath + "." + newSegment
}

func deleteKey(m map[string]any, key string) {
	parts := strings.Split(key, ".")
	deleteRecursive(m, parts)
}

func deleteRecursive(m map[string]any, keys []string) {
	if len(keys) == 0 {
		return
	}

	if len(keys) == 1 {
		delete(m, keys[0])
		return
	}

	if nextMap, ok := m[keys[0]].(map[string]any); ok {
		deleteRecursive(nextMap, keys[1:])
	}
}
