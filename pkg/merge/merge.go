package merge

import (
	"encoding/json"
	"errors"
	"io"
)

type OperationType string

const (
	OperationMerge   OperationType = "merge"
	OperationReplace OperationType = "replace"
	OperationRemove  OperationType = "remove"
)

type Opt struct {
	Path      string
	Operation OperationType
}

func JSONRawMessages(a, b json.RawMessage, opts []Opt) json.RawMessage {
	mapA, mapB := parseJSON(a), parseJSON(b)
	optsMap := make(map[string]OperationType, len(opts))
	for _, opt := range opts {
		optsMap[opt.Path] = opt.Operation
	}
	result := mergeMaps(mapA, mapB, optsMap, "")
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

func mergeMaps(a, b map[string]any, opts map[string]OperationType, currentPath string) map[string]any {
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
					if v, ok := opts[keyPath]; ok && v == OperationReplace {
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

	return result
}

func mergeArrays(a, b []any, opts map[string]OperationType, path string) []any {
	if v, ok := opts[path]; ok && v == OperationReplace {
		return b
	}
	return append(a, b...)
}

func buildPath(currentPath, newSegment string) string {
	if currentPath == "" {
		return newSegment
	}
	return currentPath + "." + newSegment
}
