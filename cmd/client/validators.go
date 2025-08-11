package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
}

type ValidationResult struct {
	Valid  bool
	Errors []string
}

type ValidatorFunc func(Manifest, string, *ValidationResult, interface{})

type Validator struct {
	Name string
	Func ValidatorFunc
	Data interface{}
}

func Validate(path string, recursive bool, validators []Validator) error {
	result := ValidationResult{Valid: true}

	walkFunc := func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && currentPath != path && !recursive {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			data, err := os.ReadFile(currentPath)
			if err != nil {
				return fmt.Errorf("error reading %s: %v", currentPath, err)
			}

			decoder := yaml.NewDecoder(bytes.NewReader(data))
			for {
				var m Manifest
				if err := decoder.Decode(&m); err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("error parsing %s: %v", currentPath, err)
				}

				if m.APIVersion != "" {
					for _, validator := range validators {
						validator.Func(m, currentPath, &result, validator.Data)
					}
				}
			}

			relPath, err := filepath.Rel(path, currentPath)
			if err != nil {
				return err
			}
			fmt.Println(relPath)
		}
		return nil
	}

	if err := filepath.Walk(path, walkFunc); err != nil {
		return err
	}

	if !result.Valid {
		for _, err := range result.Errors {
			fmt.Println(err)
		}
		// Return an error instead of calling os.Exit(1)
		return fmt.Errorf("validation failed: %s", strings.Join(result.Errors, "; "))
	}

	return nil
}

// NewDuplicateValidator creates a validator that checks for duplicate Kubernetes manifests.
// It tracks manifests using a three-level map structure:
// 1. apiVersion/kind (string)
// 2. namespace (string)
// 3. resource name (string) or name+nodeID for Envoy resources
// Returns a Validator instance initialized with empty tracking data.
func NewDuplicateValidator() Validator {
	return Validator{
		Name: "duplicate-checker",
		Func: checkDuplicateManifests,
		Data: make(map[string]map[string]map[string]manifestInfo),
	}
}

// manifestInfo tracks the first occurrence of a manifest for duplicate detection
type manifestInfo struct {
	Key  string // Composite key (name or name|nodeID)
	Path string // File path where manifest was first found
}

// checkDuplicateManifests validates that a manifest is unique within its apiVersion/kind, namespace, and name.
// Parameters:
//   - m: Parsed manifest to validate
//   - path: File path where manifest was found
//   - result: ValidationResult to update with errors
//   - data: Tracking data map[apiVersion/kind]map[namespace]map[name]manifestInfo
func checkDuplicateManifests(m Manifest, path string, result *ValidationResult, data interface{}) {
	tracker := data.(map[string]map[string]map[string]manifestInfo)

	ns := m.Metadata.Namespace
	if ns == "" {
		ns = "default"
	}

	apiKind := fmt.Sprintf("%s/%s", m.APIVersion, m.Kind)
	if _, exists := tracker[apiKind]; !exists {
		tracker[apiKind] = make(map[string]map[string]manifestInfo)
	}
	if _, exists := tracker[apiKind][ns]; !exists {
		tracker[apiKind][ns] = make(map[string]manifestInfo)
	}

	key := m.Metadata.Name

	if existing, exists := tracker[apiKind][ns][key]; exists {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Error: duplicate manifest found:\n- First occurrence: %s\n- Duplicate: %s\nFor: %s/%s/%s",
				existing.Path, path, apiKind, ns, key))
	} else {
		tracker[apiKind][ns][key] = manifestInfo{
			Key:  key,
			Path: path,
		}
	}
}
