package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir creates a temporary directory with various YAML files based on the requested setup
func setupTestDir(t *testing.T, setup string) (string, func()) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "validate-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create subdirectories
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Define all possible test files
	validYaml1 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: test-listener-http
  namespace: default
  annotations:
    envoy.kaasops.io/description: "HTTP listener on port 8080"
spec:
  name: http
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 8080
`
	validYaml2 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: Route
metadata:
  name: test-route
  namespace: default
  annotations:
    envoy.kaasops.io/description: "Route returning a JSON message for path /test"
spec:
  - name: test
    match:
      path: "/test"
    direct_response:
      status: 200
      body:
        inline_string: "{\"message\":\"test\"}"
`
	validYaml3 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: HttpFilter
metadata:
  name: test-http-filter
  namespace: default
  annotations:
    envoy.kaasops.io/description: "Basic Envoy router HTTP filter"
spec:
  - name: envoy.filters.http.router
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
`
	// Create duplicate manifest
	duplicateYaml := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: test-listener-http
  namespace: default
  annotations:
    envoy.kaasops.io/description: "Duplicate HTTP listener on port 8080"
spec:
  name: http
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 8080
`
	// Create Listener resource with node-id (should be considered duplicate after our change)
	listenerYaml1 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: test-listener
  namespace: default
  annotations:
    envoy.kaasops.io/node-id: node1
spec:
  name: test
`
	// Create duplicate Listener resource but with different node-id
	// (should be considered duplicate since node-id is ignored for non-VirtualService resources)
	listenerYaml2 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: test-listener
  namespace: default
  annotations:
    envoy.kaasops.io/node-id: node2
spec:
  name: test
`
	// Create VirtualService resource with node-id
	virtualServiceYaml1 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: test-vs
  namespace: default
  annotations:
    envoy.kaasops.io/node-id: node1
spec:
  virtualHost:
    domains:
      - example.com
`
	// Create duplicate VirtualService resource but with different node-id
	// (should NOT be considered duplicate since node-id is considered for VirtualService resources)
	virtualServiceYaml2 := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: test-vs
  namespace: default
  annotations:
    envoy.kaasops.io/node-id: node2
spec:
  virtualHost:
    domains:
      - example.com
`
	// Create a file with invalid YAML
	invalidYaml := `
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: invalid-listener
  namespace: default
spec:
  name: invalid
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 8081
  # Malformed YAML below
  malformedField: {
`

	// Create different directory setups based on the test needs
	switch setup {
	case "valid":
		// Only write valid files for the "valid" test case
		if err := os.WriteFile(filepath.Join(tempDir, "valid1.yaml"), []byte(validYaml1), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "valid2.yaml"), []byte(validYaml2), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "valid3.yaml"), []byte(validYaml3), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		// Only include one Listener resource to avoid duplicates
		if err := os.WriteFile(filepath.Join(tempDir, "listener1.yaml"), []byte(listenerYaml1), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		// Include both VirtualService resources with different node-ids
		// These should not be considered duplicates with our code change
		if err := os.WriteFile(filepath.Join(tempDir, "vs1.yaml"), []byte(virtualServiceYaml1), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "vs2.yaml"), []byte(virtualServiceYaml2), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

	case "node-id-duplicate":
		// Create a directory with Listener resources that have the same name but different node-ids
		// After our code change, these should be considered duplicates
		if err := os.WriteFile(filepath.Join(tempDir, "listener1.yaml"), []byte(listenerYaml1), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "listener2.yaml"), []byte(listenerYaml2), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

	case "duplicate":
		// Create files with duplicates for the "duplicate" test case
		if err := os.WriteFile(filepath.Join(tempDir, "valid1.yaml"), []byte(validYaml1), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "duplicate.yaml"), []byte(duplicateYaml), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

	case "invalid":
		// Create a directory with an invalid YAML file
		if err := os.WriteFile(filepath.Join(tempDir, "invalid.yaml"), []byte(invalidYaml), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Return cleanup function
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// TestValidate tests the Validate function
func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		setup     string
		recursive bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Valid directory, non-recursive",
			setup:     "valid",
			recursive: false,
			wantErr:   false,
		},
		{
			name:      "Valid directory, recursive",
			setup:     "valid",
			recursive: true,
			wantErr:   false,
		},
		{
			name:      "Non-existent directory",
			setup:     "valid", // The setup doesn't matter for this test
			recursive: false,
			wantErr:   true,
			errMsg:    "no such file or directory",
		},
		{
			name:      "Directory with duplicate manifests",
			setup:     "duplicate",
			recursive: false,
			wantErr:   true,
			errMsg:    "duplicate manifest found",
		},
		{
			name:      "Directory with invalid YAML",
			setup:     "invalid",
			recursive: false,
			wantErr:   true,
			errMsg:    "error parsing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, cleanup := setupTestDir(t, tt.setup)
			defer cleanup()

			// For the non-existent directory test, we need to use a non-existent path
			path := tempDir
			if tt.name == "Non-existent directory" {
				path = filepath.Join(tempDir, "nonexistent")
			}

			validators := []Validator{
				NewDuplicateValidator(),
			}

			err := Validate(path, tt.recursive, validators)

			// Check error result
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message if expected error
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error message = %v, expected to contain: %v", err, tt.errMsg)
			}
		})
	}
}

// TestDuplicateValidator tests the DuplicateValidator
func TestDuplicateValidator(t *testing.T) {
	tests := []struct {
		name       string
		manifests  []Manifest
		expectDupe bool
	}{
		{
			name: "No duplicates",
			manifests: []Manifest{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-deployment-1",
						Namespace: "default",
					},
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-deployment-2",
						Namespace: "default",
					},
				},
			},
			expectDupe: false,
		},
		{
			name: "Duplicate manifests",
			manifests: []Manifest{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-deployment",
						Namespace: "default",
					},
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-deployment",
						Namespace: "default",
					},
				},
			},
			expectDupe: true,
		},
		{
			name: "Same name but different namespace",
			manifests: []Manifest{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-deployment",
						Namespace: "default",
					},
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-deployment",
						Namespace: "test",
					},
				},
			},
			expectDupe: false,
		},
		{
			name: "Same name but different kind",
			manifests: []Manifest{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-resource",
						Namespace: "default",
					},
				},
				{
					APIVersion: "v1",
					Kind:       "Service",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-resource",
						Namespace: "default",
					},
				},
			},
			expectDupe: false,
		},
		{
			name: "Same name, same kind (Listener), different node-id - should be duplicate after change",
			manifests: []Manifest{
				{
					APIVersion: "envoy.kaasops.io/v1alpha1",
					Kind:       "Listener",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-listener",
						Namespace: "default",
						Annotations: map[string]string{
							"envoy.kaasops.io/node-id": "node1",
						},
					},
				},
				{
					APIVersion: "envoy.kaasops.io/v1alpha1",
					Kind:       "Listener",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-listener",
						Namespace: "default",
						Annotations: map[string]string{
							"envoy.kaasops.io/node-id": "node2",
						},
					},
				},
			},
			expectDupe: true, // Now we expect a duplicate because node-id is ignored for non-VirtualService resources
		},
		{
			name: "Same name, same kind (VirtualService), different node-id - should NOT be duplicate",
			manifests: []Manifest{
				{
					APIVersion: "envoy.kaasops.io/v1alpha1",
					Kind:       "VirtualService",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-vs",
						Namespace: "default",
						Annotations: map[string]string{
							"envoy.kaasops.io/node-id": "node1",
						},
					},
				},
				{
					APIVersion: "envoy.kaasops.io/v1alpha1",
					Kind:       "VirtualService",
					Metadata: struct {
						Name        string            `yaml:"name"`
						Namespace   string            `yaml:"namespace"`
						Annotations map[string]string `yaml:"annotations"`
					}{
						Name:      "test-vs",
						Namespace: "default",
						Annotations: map[string]string{
							"envoy.kaasops.io/node-id": "node2",
						},
					},
				},
			},
			expectDupe: false, // Not a duplicate because node-id is considered for VirtualService resources
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewDuplicateValidator()
			result := ValidationResult{Valid: true}

			for i, m := range tt.manifests {
				validator.Func(m, fmt.Sprintf("test-file-%d.yaml", i), &result, validator.Data)
			}

			if result.Valid == tt.expectDupe {
				t.Errorf("DuplicateValidator result.Valid = %v, expectDupe %v", result.Valid, tt.expectDupe)
			}
		})
	}
}

// TestValidateWithInvalidYAML tests the Validate function with invalid YAML files
func TestValidateWithInvalidYAML(t *testing.T) {
	tempDir, cleanup := setupTestDir(t, "invalid")
	defer cleanup()

	// Point to the file with invalid YAML
	invalidFilePath := filepath.Join(tempDir, "invalid.yaml")

	validators := []Validator{
		NewDuplicateValidator(),
	}

	// This should cause an error because the YAML is invalid
	err := Validate(invalidFilePath, false, validators)
	if err == nil {
		t.Errorf("Validate() with invalid YAML did not return an error")
	} else if !strings.Contains(err.Error(), "error parsing") {
		t.Errorf("Validate() error message = %v, expected to contain: 'error parsing'", err)
	}
}
