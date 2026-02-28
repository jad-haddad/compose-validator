package validator

import (
	"testing"

	"github.com/yourusername/compose-validator/internal/config"
	"github.com/yourusername/compose-validator/internal/parser"
)

// Test Helpers

func createService(name string, fieldOrder []string, config map[string]interface{}) parser.Service {
	return parser.Service{
		Name:       name,
		FieldOrder: fieldOrder,
		Config:     config,
		Line:       1,
		Column:     1,
	}
}

// Test Cases

func TestValidate_ValidFieldOrder(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "environment", "ports", "restart"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment":    []interface{}{"KEY=value"},
			"ports":          []interface{}{"8080:80"},
			"restart":        "always",
		},
	)

	violations := validateFieldOrder("web", service, cfg.FieldOrder, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for valid field order, got %d: %v", len(violations), violations)
	}
}

func TestValidate_InvalidFieldOrder(t *testing.T) {
	cfg := config.NewDefaultConfig()

	// Image before container_name - wrong order
	service := createService(
		"web",
		[]string{"image", "container_name", "environment"},
		map[string]interface{}{
			"image":          "nginx:latest",
			"container_name": "web-server",
			"environment":    []interface{}{"KEY=value"},
		},
	)

	violations := validateFieldOrder("web", service, cfg.FieldOrder, cfg)

	if len(violations) != 2 {
		t.Errorf("Expected 2 violations, got %d: %v", len(violations), violations)
	}

	// Check first violation
	if violations[0].Field != "image" {
		t.Errorf("Expected first violation on 'image', got '%s'", violations[0].Field)
	}
}

func TestValidate_AlphabetizedEnvironment_List(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment": []interface{}{
				"AAA=value1",
				"BBB=value2",
				"CCC=value3",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for alphabetized env vars, got %d: %v", len(violations), violations)
	}
}

func TestValidate_UnalphabetizedEnvironment_List(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment": []interface{}{
				"ZZZ=value1",
				"AAA=value2",
				"MMM=value3",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for unalphabetized env vars, got %d: %v", len(violations), violations)
	}

	if violations[0].Field != "environment" {
		t.Errorf("Expected violation on 'environment', got '%s'", violations[0].Field)
	}
}

func TestValidate_AlphabetizedEnvironment_Map(t *testing.T) {
	// Skip this test because Go maps don't preserve insertion order,
	// making it impossible to reliably check if a map was alphabetized.
	// In practice, environment variables should use list format for ordering.
	t.Skip("Skipping map alphabetization test - Go maps don't preserve order")
}

func TestValidate_AlphabetizedVolumes(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "volumes"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"volumes": []interface{}{
				"/aaa:/container/aaa",
				"/bbb:/container/bbb",
				"/ccc:/container/ccc",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for alphabetized volumes, got %d: %v", len(violations), violations)
	}
}

func TestValidate_UnalphabetizedVolumes(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "volumes"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"volumes": []interface{}{
				"/zzz:/container/zzz",
				"/aaa:/container/aaa",
				"/mmm:/container/mmm",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for unalphabetized volumes, got %d: %v", len(violations), violations)
	}

	if violations[0].Field != "volumes" {
		t.Errorf("Expected violation on 'volumes', got '%s'", violations[0].Field)
	}
}

func TestValidate_AlphabetizedLabels(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "labels"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"labels": []interface{}{
				"aaa.label=value1",
				"bbb.label=value2",
				"ccc.label=value3",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for alphabetized labels, got %d: %v", len(violations), violations)
	}
}

func TestValidate_UnalphabetizedLabels(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "labels"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"labels": []interface{}{
				"zzz.label=value1",
				"aaa.label=value2",
				"mmm.label=value3",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for unalphabetized labels, got %d: %v", len(violations), violations)
	}
}

func TestValidate_StrictMode_ExtraField(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Strict = true

	// Add a custom field not in default order
	service := createService(
		"web",
		[]string{"container_name", "image", "custom_field"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"custom_field":   "custom_value",
		},
	)

	violations := validateFieldOrder("web", service, cfg.FieldOrder, cfg)

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for extra field in strict mode, got %d: %v", len(violations), violations)
	}

	if violations[0].Field != "custom_field" {
		t.Errorf("Expected violation on 'custom_field', got '%s'", violations[0].Field)
	}
}

func TestValidate_NonStrictMode_ExtraField(t *testing.T) {
	cfg := config.NewDefaultConfig()
	// Strict is false by default

	service := createService(
		"web",
		[]string{"container_name", "image", "custom_field"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"custom_field":   "custom_value",
		},
	)

	violations := validateFieldOrder("web", service, cfg.FieldOrder, cfg)

	// Should not report violations for extra fields in non-strict mode
	for _, v := range violations {
		if v.Field == "custom_field" && v.Type == "order" {
			t.Error("Should not report extra field as violation in non-strict mode")
		}
	}
}

func TestValidate_CaseInsensitiveAlphabetization(t *testing.T) {
	cfg := config.NewDefaultConfig()

	// Mixed case should be sorted case-insensitively
	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment": []interface{}{
				"aaa=value1", // lowercase
				"BBB=value2", // uppercase
				"ccc=value3", // lowercase
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for case-insensitive alphabetization, got %d: %v", len(violations), violations)
	}
}

func TestValidate_UnalphabetizedCaseInsensitive(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment": []interface{}{
				"ZZZ=value1",
				"aaa=value2", // should come before ZZZ when case-insensitive
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for unalphabetized case-insensitive env vars, got %d: %v", len(violations), violations)
	}
}

func TestValidate_EmptyEnvironment(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment":    []interface{}{},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for empty environment, got %d: %v", len(violations), violations)
	}
}

func TestValidate_SingleItemEnvironment(t *testing.T) {
	cfg := config.NewDefaultConfig()

	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment":    []interface{}{"SINGLE=value"},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for single-item environment, got %d: %v", len(violations), violations)
	}
}

func TestValidate_DisabledAlphabetization(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Alphabetization.Environment = false

	service := createService(
		"web",
		[]string{"container_name", "image", "environment"},
		map[string]interface{}{
			"container_name": "web-server",
			"image":          "nginx:latest",
			"environment": []interface{}{
				"ZZZ=value1",
				"AAA=value2",
			},
		},
	)

	violations := validateAlphabetization("web", service, cfg)

	// Should not report violations when alphabetization is disabled
	for _, v := range violations {
		if v.Field == "environment" {
			t.Error("Should not report environment violation when alphabetization is disabled")
		}
	}
}
