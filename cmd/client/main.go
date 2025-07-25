package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string            `yaml:"name"`
		Namespace   string            `yaml:"namespace"`
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
}

type ValidationResult struct {
	Valid  bool
	Errors []string
}

func validate(path string, recursive bool) error {
	// Initialize validation state
	manifestTracker := make(map[string]map[string]map[string]bool)
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

			// Try parsing as single manifest first
			var singleManifest Manifest
			if err := yaml.Unmarshal(data, &singleManifest); err == nil && singleManifest.APIVersion != "" {
				runValidations(singleManifest, currentPath, &result, manifestTracker)
			} else {
				// If single manifest fails, try parsing as list
				var manifests []Manifest
				if err := yaml.Unmarshal(data, &manifests); err != nil {
					return fmt.Errorf("error parsing %s: %v", currentPath, err)
				}
				for _, m := range manifests {
					runValidations(m, currentPath, &result, manifestTracker)
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
		os.Exit(1)
	}

	return nil
}

func runValidations(m Manifest, path string, result *ValidationResult, tracker map[string]map[string]map[string]bool) {
	// Skip invalid manifests
	if m.APIVersion == "" || m.Kind == "" || m.Metadata.Name == "" {
		return
	}

	// Run all validation checks
	checkDuplicateManifests(m, path, result, tracker)
	// Additional validation checks can be added here
}

func checkDuplicateManifests(m Manifest, path string, result *ValidationResult, tracker map[string]map[string]map[string]bool) {
	ns := m.Metadata.Namespace
	if ns == "" {
		ns = "default"
	}

	nodeID := ""
	if m.Metadata.Annotations != nil {
		nodeID = m.Metadata.Annotations["envoy.kaasops.io/node-id"]
	}

	apiKind := fmt.Sprintf("%s/%s", m.APIVersion, m.Kind)
	if _, exists := tracker[apiKind]; !exists {
		tracker[apiKind] = make(map[string]map[string]bool)
	}
	if _, exists := tracker[apiKind][ns]; !exists {
		tracker[apiKind][ns] = make(map[string]bool)
	}

	key := m.Metadata.Name
	if nodeID != "" {
		key = fmt.Sprintf("%s|%s", m.Metadata.Name, nodeID)
	}

	if tracker[apiKind][ns][key] {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Error: duplicate manifest found: %s/%s/%s in %s", apiKind, ns, key, path))
	}
	tracker[apiKind][ns][key] = true
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "envoy-xds-controller",
		Short: "Root command for Envoy XDS Controller",
	}

	var path string
	var recursive bool
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configurations by scanning for YAML files and running validation checks",
		Run: func(cmd *cobra.Command, args []string) {
			if path == "" {
				fmt.Println("Error: --path is required")
				os.Exit(1)
			}

			if err := validate(path, recursive); err != nil {
				fmt.Printf("Validation error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	validateCmd.Flags().StringVarP(&path, "path", "p", "", "Path to directory to validate")
	validateCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively validate directory")
	validateCmd.MarkFlagRequired("path")

	rootCmd.AddCommand(validateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
