package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig() returned nil")
	}

	// Check field order is not empty
	if len(cfg.FieldOrder) == 0 {
		t.Error("Default field order should not be empty")
	}

	// Check expected fields exist in default order
	expectedFields := []string{"container_name", "image", "environment", "volumes", "labels"}
	for _, field := range expectedFields {
		found := false
		for _, f := range cfg.FieldOrder {
			if f == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field '%s' in default field order", field)
		}
	}

	// Check alphabetization defaults
	if !cfg.Alphabetization.Environment {
		t.Error("Environment alphabetization should be enabled by default")
	}
	if !cfg.Alphabetization.Volumes {
		t.Error("Volumes alphabetization should be enabled by default")
	}
	if !cfg.Alphabetization.Labels {
		t.Error("Labels alphabetization should be enabled by default")
	}

	// Check strict mode is false by default
	if cfg.Strict {
		t.Error("Strict mode should be disabled by default")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		configContent  string
		expectedError  bool
		expectedFields int
		expectedStrict bool
	}{
		{
			name: "valid full config",
			configContent: `
field_order:
  - container_name
  - image
  - environment
alphabetization:
  environment: true
  volumes: false
  labels: true
strict: true
exclude:
  - "**/test/**"
`,
			expectedError:  false,
			expectedFields: 3,
			expectedStrict: true,
		},
		{
			name: "minimal config",
			configContent: `
field_order:
  - container_name
  - image
`,
			expectedError:  false,
			expectedFields: 2,
			expectedStrict: false,
		},
		{
			name:           "empty config uses defaults",
			configContent:  "",
			expectedError:  false,
			expectedFields: len(DefaultFieldOrder),
			expectedStrict: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, test.name+".yaml")
			if err := os.WriteFile(configPath, []byte(test.configContent), 0644); err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}

			cfg, err := LoadFromFile(configPath)

			if test.expectedError && err == nil {
				t.Error("Expected error but got none")
			}

			if !test.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil {
				if len(cfg.FieldOrder) != test.expectedFields {
					t.Errorf("Expected %d fields, got %d", test.expectedFields, len(cfg.FieldOrder))
				}
				if cfg.Strict != test.expectedStrict {
					t.Errorf("Expected strict=%v, got %v", test.expectedStrict, cfg.Strict)
				}
			}
		})
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoad(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Test: Load from current directory
	t.Run("load from current directory", func(t *testing.T) {
		os.Chdir(tmpDir)

		configContent := `
field_order:
  - container_name
  - image
`
		if err := os.WriteFile(filepath.Join(tmpDir, ".compose-validator.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(cfg.FieldOrder) != 2 {
			t.Errorf("Expected 2 fields from loaded config, got %d", len(cfg.FieldOrder))
		}
	})

	// Clean up
	os.Remove(filepath.Join(tmpDir, ".compose-validator.yaml"))
}

func TestGetFieldOrder(t *testing.T) {
	cfg := NewDefaultConfig()

	// Test default order
	order := cfg.GetFieldOrder("default-service")
	if len(order) != len(DefaultFieldOrder) {
		t.Errorf("Expected %d fields in default order, got %d", len(DefaultFieldOrder), len(order))
	}

	// Test service override
	cfg.ServiceOverrides = map[string]ServiceOverride{
		"database": {
			FieldOrder: []string{"container_name", "image", "environment"},
		},
	}

	customOrder := cfg.GetFieldOrder("database")
	if len(customOrder) != 3 {
		t.Errorf("Expected 3 fields in custom order, got %d", len(customOrder))
	}

	// Test non-overridden service still uses default
	defaultOrder := cfg.GetFieldOrder("web")
	if len(defaultOrder) != len(DefaultFieldOrder) {
		t.Errorf("Expected %d fields for non-overridden service, got %d", len(DefaultFieldOrder), len(defaultOrder))
	}
}

func TestShouldAlphabetize(t *testing.T) {
	cfg := NewDefaultConfig()

	tests := []struct {
		field    string
		expected bool
	}{
		{"environment", true},
		{"volumes", true},
		{"labels", true},
		{"image", false},
		{"container_name", false},
		{"ports", false},
		{"restart", false},
		{"", false},
		{"unknown", false},
	}

	for _, test := range tests {
		result := cfg.ShouldAlphabetize(test.field)
		if result != test.expected {
			t.Errorf("ShouldAlphabetize(%q) = %v, expected %v", test.field, result, test.expected)
		}
	}
}

func TestShouldAlphabetize_Disabled(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Alphabetization.Environment = false
	cfg.Alphabetization.Volumes = false

	if cfg.ShouldAlphabetize("environment") {
		t.Error("Should not alphabetize environment when disabled")
	}

	if cfg.ShouldAlphabetize("volumes") {
		t.Error("Should not alphabetize volumes when disabled")
	}

	if !cfg.ShouldAlphabetize("labels") {
		t.Error("Should still alphabetize labels when enabled")
	}
}

func TestIsExcluded(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Exclude = []string{
		"**/test/**",
		"docker-compose.override.yml",
		"*.tmp",
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"docker-compose.yml", false},
		{"docker-compose.override.yml", true},
		{"test/docker-compose.yml", true},
		{"/path/to/test/docker-compose.yml", true},
		{"backup.yml.tmp", true},
		{"production.yml", false},
	}

	for _, test := range tests {
		result := cfg.IsExcluded(test.path)
		if result != test.expected {
			t.Errorf("IsExcluded(%q) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

func TestDefaultFieldOrder(t *testing.T) {
	// Check that DefaultFieldOrder has expected structure
	if len(DefaultFieldOrder) == 0 {
		t.Fatal("DefaultFieldOrder is empty")
	}

	// Identity fields should be first
	if DefaultFieldOrder[0] != "container_name" {
		t.Errorf("Expected 'container_name' as first field, got '%s'", DefaultFieldOrder[0])
	}

	// Labels should be last
	if DefaultFieldOrder[len(DefaultFieldOrder)-1] != "labels" {
		t.Errorf("Expected 'labels' as last field, got '%s'", DefaultFieldOrder[len(DefaultFieldOrder)-1])
	}
}

func TestLoad_MultipleLocations(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "stacks", "production")
	os.MkdirAll(nestedDir, 0755)

	// Create config in parent directory
	parentConfig := `
field_order:
  - container_name
  - image
strict: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".compose-validator.yaml"), []byte(parentConfig), 0644); err != nil {
		t.Fatalf("Failed to create parent config: %v", err)
	}

	// Change to nested directory
	os.Chdir(nestedDir)

	// Should load config from parent
	cfg, err := Load()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(cfg.FieldOrder) != 2 {
		t.Errorf("Expected 2 fields from parent config, got %d", len(cfg.FieldOrder))
	}

	if !cfg.Strict {
		t.Error("Expected strict=true from parent config")
	}
}
