package test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

// validateExtraFieldsUsage is a copy of the function from the webhook
func validateExtraFieldsUsage(vst *v1alpha1.VirtualServiceTemplate) error {
	// If there are no extraFields, there's nothing to validate
	if len(vst.Spec.ExtraFields) == 0 {
		return nil
	}

	// Valid extraField types
	validTypes := map[string]bool{
		"string": true,
		"enum":   true,
	}

	// Create a map to track which extraFields are used
	extraFieldsUsed := make(map[string]bool)
	for _, field := range vst.Spec.ExtraFields {
		if field.Name == "" {
			return fmt.Errorf("extraField name cannot be empty")
		}
		if field.Type == "" {
			return fmt.Errorf("extraField '%s' type cannot be empty", field.Name)
		}
		if !validTypes[field.Type] {
			return fmt.Errorf("extraField '%s' has unknown type '%s', valid types are: string, enum", field.Name, field.Type)
		}
		if field.Type == "enum" && len(field.Enum) == 0 {
			return fmt.Errorf("extraField '%s' type is 'enum' but no enum values are defined", field.Name)
		}
		extraFieldsUsed[field.Name] = false
	}

	// Convert the template spec to JSON to search for template references
	specJSON, err := json.Marshal(vst.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal template spec: %w", err)
	}

	// Use regex to find all template references in the form {{ .FieldName }}
	// This regex matches {{ .Name }} with optional whitespace
	re := regexp.MustCompile(`{{\s*\.([A-Za-z0-9_]+)\s*}}`)
	matches := re.FindAllStringSubmatch(string(specJSON), -1)

	// Mark each extraField that is used in the template
	for _, match := range matches {
		if len(match) > 1 {
			fieldName := match[1]
			if _, exists := extraFieldsUsed[fieldName]; exists {
				extraFieldsUsed[fieldName] = true
			}
		}
	}

	// Check if any extraField is not used
	var unusedFields []string
	for fieldName, used := range extraFieldsUsed {
		if !used {
			unusedFields = append(unusedFields, fieldName)
		}
	}

	// Return an error if there are unused extraFields
	if len(unusedFields) > 0 {
		// nolint: lll
		return fmt.Errorf("the following extraFields are defined but not used in the template: %s", strings.Join(unusedFields, ", "))
	}

	return nil
}

func TestValidateExtraFieldsUsage(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		templatePath  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Invalid template with unused extraField",
			templatePath:  "../dev/testdata/invalid/test-unused-extrafield.yaml",
			expectError:   true,
			errorContains: "UnusedField",
		},
		{
			name:          "Invalid template with empty extraField name",
			templatePath:  "../dev/testdata/invalid/test-empty-name-extrafield.yaml",
			expectError:   true,
			errorContains: "extraField name cannot be empty",
		},
		{
			name:          "Invalid template with unknown extraField type",
			templatePath:  "../dev/testdata/invalid/test-unknown-type-extrafield.yaml",
			expectError:   true,
			errorContains: "unknown type",
		},
		{
			name:          "Invalid template with enum type but no enum values",
			templatePath:  "../dev/testdata/invalid/test-empty-enum-extrafield.yaml",
			expectError:   true,
			errorContains: "no enum values are defined",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load the template from file
			yamlData, err := os.ReadFile(tc.templatePath)
			if err != nil {
				t.Fatalf("Failed to read template file: %v", err)
			}

			// Convert YAML to JSON
			jsonData, err := yaml.YAMLToJSON(yamlData)
			if err != nil {
				t.Fatalf("Failed to convert YAML to JSON: %v", err)
			}

			// Unmarshal JSON into VirtualServiceTemplate
			var vst v1alpha1.VirtualServiceTemplate
			if err := json.Unmarshal(jsonData, &vst); err != nil {
				t.Fatalf("Failed to unmarshal template: %v", err)
			}

			// Validate extraFields usage
			err = validateExtraFieldsUsage(&vst)

			// Check if error matches expectation
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
