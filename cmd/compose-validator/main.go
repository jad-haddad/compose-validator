package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/compose-validator/internal/config"
	"github.com/yourusername/compose-validator/internal/fixer"
	"github.com/yourusername/compose-validator/internal/parser"
	"github.com/yourusername/compose-validator/internal/validator"
)

var (
	// Version is set during build
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Flags
	verbose    bool
	fixMode    bool
	configPath string
	checkOrder bool
	checkAlpha bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "compose-validator [files...]",
		Short: "Validate and fix Docker Compose YAML field ordering",
		Long: `Docker Compose Field Order Validator

A tool to validate and enforce consistent Docker Compose service definitions.
Ensures field ordering and alphabetization of environment variables, volumes, and labels.`,
		Args: cobra.MinimumNArgs(1),
		RunE: run,
	}

	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&fixMode, "fix", false, "Automatically fix violations")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Path to configuration file")
	rootCmd.Flags().BoolVar(&checkOrder, "check-order-only", false, "Only check field order")
	rootCmd.Flags().BoolVar(&checkAlpha, "check-alphabetization-only", false, "Only check alphabetization")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("compose-validator version %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}

	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no files specified")
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = config.LoadFromFile(configPath)
	} else {
		cfg, err = config.Load()
	}

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if verbose {
		color.Blue("Loaded configuration")
		fmt.Printf("Field order: %v\n", cfg.FieldOrder)
		fmt.Printf("Alphabetization: env=%v, volumes=%v, labels=%v\n",
			cfg.Alphabetization.Environment,
			cfg.Alphabetization.Volumes,
			cfg.Alphabetization.Labels)
	}

	// Process files
	allValid := true
	totalViolations := 0

	for _, pattern := range args {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}

		if len(files) == 0 {
			color.Yellow("Warning: no files match pattern %s", pattern)
			continue
		}

		for _, file := range files {
			if cfg.IsExcluded(file) {
				if verbose {
					color.Yellow("Skipping excluded file: %s", file)
				}
				continue
			}

			result, err := processFile(file, cfg)
			if err != nil {
				color.Red("Error processing %s: %v", file, err)
				allValid = false
				continue
			}

			if !result.Valid {
				allValid = false
				totalViolations += len(result.Violations)
			}
		}
	}

	if allValid {
		color.Green("✓ All files are valid!")
		return nil
	}

	color.Red("✗ Found %d violation(s)", totalViolations)
	if !fixMode {
		fmt.Println("\nRun with --fix to automatically correct issues")
	}
	os.Exit(1)
	return nil
}

func processFile(path string, cfg *config.Config) (*validator.ValidationResult, error) {
	file, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	result, err := validator.Validate(file, cfg)
	if err != nil {
		return nil, err
	}

	if fixMode && !result.Valid {
		fixResult, err := fixer.Fix(file, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to fix file: %w", err)
		}

		if fixResult.Fixed {
			color.Green("✓ Fixed %s:", path)
			for _, change := range fixResult.Changes {
				fmt.Printf("  - %s\n", change)
			}

			// Re-validate after fixing
			file, err = parser.ParseFile(path)
			if err != nil {
				return nil, err
			}

			result, err = validator.Validate(file, cfg)
			if err != nil {
				return nil, err
			}
		}
	} else if !result.Valid {
		// Print violations
		color.Red("✗ %s:", path)
		for _, v := range result.Violations {
			printViolation(v, cfg)
		}
	} else if verbose {
		color.Green("✓ %s: valid", path)
	}

	return result, nil
}

func printViolation(v validator.Violation, cfg *config.Config) {
	switch v.Type {
	case "order":
		fmt.Printf("  Service '%s': %s\n", v.Service, v.Message)
		if v.Expected != "" && v.Actual != "" {
			fmt.Printf("    Expected: '%s' at this position\n", v.Expected)
			fmt.Printf("    Actual: '%s'\n", v.Actual)
		}
		if v.Line > 0 {
			fmt.Printf("    Line: %d\n", v.Line)
		}

	case "alphabetization":
		fmt.Printf("  Service '%s': %s\n", v.Service, v.Message)
		fmt.Printf("    Field '%s' should be alphabetized\n", v.Field)
		if v.Line > 0 {
			fmt.Printf("    Line: %d\n", v.Line)
		}
	}
}
