package main

import (
	"bytes"
	"fmt"
	"io"
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

// Track both manifest key and file path
type ManifestInfo struct {
	Key  string
	Path string
}

func validate(path string, recursive bool) error {
	// map[apiVersion/kind]map[namespace]map[name+nodeID]ManifestInfo
	manifestTracker := make(map[string]map[string]map[string]ManifestInfo)
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
				var manifest Manifest
				if err := decoder.Decode(&manifest); err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("error parsing %s: %v", currentPath, err)
				}

				if manifest.APIVersion != "" {
					runValidations(manifest, currentPath, &result, manifestTracker)
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

func runValidations(m Manifest, path string, result *ValidationResult, tracker map[string]map[string]map[string]ManifestInfo) {
	if m.APIVersion == "" || m.Kind == "" || m.Metadata.Name == "" {
		return
	}

	checkDuplicateManifests(m, path, result, tracker)
}

func checkDuplicateManifests(m Manifest, path string, result *ValidationResult, tracker map[string]map[string]map[string]ManifestInfo) {
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
		tracker[apiKind] = make(map[string]map[string]ManifestInfo)
	}
	if _, exists := tracker[apiKind][ns]; !exists {
		tracker[apiKind][ns] = make(map[string]ManifestInfo)
	}

	key := m.Metadata.Name
	if nodeID != "" {
		key = fmt.Sprintf("%s|%s", m.Metadata.Name, nodeID)
	}

	if existing, exists := tracker[apiKind][ns][key]; exists {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Error: duplicate manifest found:\n- First occurrence: %s\n- Duplicate: %s\nFor: %s/%s/%s",
				existing.Path, path, apiKind, ns, key))
	} else {
		tracker[apiKind][ns][key] = ManifestInfo{
			Key:  key,
			Path: path,
		}
	}
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
