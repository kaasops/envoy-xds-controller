package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

			validators := []Validator{
				NewDuplicateValidator(),
				// Additional validators can be added here
			}

			if err := Validate(path, recursive, validators); err != nil {
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
