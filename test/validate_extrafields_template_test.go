package test

import (
	"strings"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

func TestValidateExtraFieldsAgainstTemplate(t *testing.T) {
	// Create a template with defined extraFields
	template := &v1alpha1.VirtualServiceTemplate{
		Spec: v1alpha1.VirtualServiceTemplateSpec{
			ExtraFields: []*v1alpha1.ExtraField{
				{
					Name:        "Field1",
					Type:        "string",
					Description: "Test field 1",
					Required:    true,
				},
				{
					Name:        "Field2",
					Type:        "enum",
					Description: "Test field 2",
					Required:    false,
					Enum:        []string{"value1", "value2", "value3"},
				},
			},
		},
	}

	// Test cases
	testCases := []struct {
		name          string
		extraFields   map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid extraFields - all defined in template",
			extraFields: map[string]string{
				"Field1": "test value",
				"Field2": "value1",
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name: "Invalid extraFields - missing required field",
			extraFields: map[string]string{
				"Field2": "value1",
			},
			expectError:   true,
			errorContains: "required extra field 'Field1' is missing or empty",
		},
		{
			name: "Invalid extraFields - invalid enum value",
			extraFields: map[string]string{
				"Field1": "test value",
				"Field2": "invalid value",
			},
			expectError:   true,
			errorContains: "extra field 'Field2' has invalid value 'invalid value'",
		},
		{
			name: "Invalid extraFields - field not defined in template",
			extraFields: map[string]string{
				"Field1": "test value",
				"Field2": "value1",
				"Field3": "unexpected field",
			},
			expectError:   true,
			errorContains: "extra field 'Field3' is not defined in the template",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a virtual service with the test extraFields
			vs := &v1alpha1.VirtualService{
				Spec: v1alpha1.VirtualServiceSpec{
					ExtraFields: tc.extraFields,
				},
			}

			// Call FillFromTemplate to validate extraFields
			err := vs.FillFromTemplate(template)

			// Check if error matches expectation
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return s != "" && substr != "" && strings.Contains(s, substr)
}
