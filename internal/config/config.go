package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// FieldOrder defines the default order of fields in Docker Compose services
var DefaultFieldOrder = []string{
	// Identity
	"container_name",
	"image",
	"build",
	// Permissions
	"user",
	// Environment
	"environment",
	"env_file",
	// Runtime
	"networks",
	"network_mode",
	"ports",
	"devices",
	// Health
	"healthcheck",
	// Lifecycle
	"restart",
	"cap_add",
	"privileged",
	"extra_hosts",
	// Storage
	"volumes",
	// Labels (always last)
	"labels",
}

// AlphabetizationRules defines which fields must be alphabetized
type AlphabetizationRules struct {
	Environment bool `yaml:"environment"`
	Volumes     bool `yaml:"volumes"`
	Labels      bool `yaml:"labels"`
}

// ServiceOverride allows custom field order for specific services
type ServiceOverride struct {
	FieldOrder []string `yaml:"field_order"`
}

// Config represents the validator configuration
type Config struct {
	FieldOrder       []string                   `yaml:"field_order"`
	Alphabetization  AlphabetizationRules       `yaml:"alphabetization"`
	Strict           bool                       `yaml:"strict"`
	Exclude          []string                   `yaml:"exclude"`
	ServiceOverrides map[string]ServiceOverride `yaml:"service_overrides"`
}

// NewDefaultConfig creates a default configuration
func NewDefaultConfig() *Config {
	return &Config{
		FieldOrder: DefaultFieldOrder,
		Alphabetization: AlphabetizationRules{
			Environment: true,
			Volumes:     true,
			Labels:      true,
		},
		Strict:           false,
		Exclude:          []string{},
		ServiceOverrides: make(map[string]ServiceOverride),
	}
}

// Possible config file locations (in order of priority)
var configFileNames = []string{
	".compose-validator.yaml",
	".compose-validator.yml",
	"compose-validator.yaml",
	"compose-validator.yml",
}

// Load attempts to load configuration from standard locations
func Load() (*Config, error) {
	// Try to find config in current directory and parent directories
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		for _, name := range configFileNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return LoadFromFile(path)
			}
		}

		// Move up to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// No config found, return default
	return NewDefaultConfig(), nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	cfg := NewDefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Validate that field order is not empty
	if len(cfg.FieldOrder) == 0 {
		cfg.FieldOrder = DefaultFieldOrder
	}

	return cfg, nil
}

// GetFieldOrder returns the field order for a specific service
func (c *Config) GetFieldOrder(serviceName string) []string {
	if override, ok := c.ServiceOverrides[serviceName]; ok && len(override.FieldOrder) > 0 {
		return override.FieldOrder
	}
	return c.FieldOrder
}

// ShouldAlphabetize checks if a field should be alphabetized
func (c *Config) ShouldAlphabetize(field string) bool {
	switch field {
	case "environment":
		return c.Alphabetization.Environment
	case "volumes":
		return c.Alphabetization.Volumes
	case "labels":
		return c.Alphabetization.Labels
	default:
		return false
	}
}

// IsExcluded checks if a file path matches any exclusion pattern
func (c *Config) IsExcluded(path string) bool {
	for _, pattern := range c.Exclude {
		// Handle **/ prefix for recursive matching
		if strings.HasPrefix(pattern, "**/") {
			suffix := pattern[3:]
			// Remove trailing /** if present for simpler matching
			suffix = strings.TrimSuffix(suffix, "/**")
			suffix = strings.TrimSuffix(suffix, "/")

			// Check if the suffix appears as a path component
			pathComponents := strings.Split(path, string(filepath.Separator))
			for _, component := range pathComponents {
				if matched, _ := filepath.Match(suffix, component); matched {
					return true
				}
			}
			continue
		}

		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}
