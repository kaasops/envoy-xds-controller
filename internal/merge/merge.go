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

func mergeArrays(a, b []any, opts *parsedOpts, path string) []any {
	if _, ok := opts.replace[path]; ok {
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
