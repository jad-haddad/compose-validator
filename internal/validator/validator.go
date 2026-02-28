package validator

import (
	"fmt"
	"strings"

	"github.com/yourusername/compose-validator/internal/config"
	"github.com/yourusername/compose-validator/internal/parser"
)

// Violation represents a validation error
type Violation struct {
	Type     string // "order" or "alphabetization"
	Service  string
	Field    string
	Message  string
	Expected string
	Actual   string
	Line     int
	Column   int
}

// ValidationResult contains all violations found in a file
type ValidationResult struct {
	File       string
	Valid      bool
	Violations []Violation
}

// Validate validates a Docker Compose file
func Validate(file *parser.ComposeFile, cfg *config.Config) (*ValidationResult, error) {
	result := &ValidationResult{
		File:       file.Path,
		Valid:      true,
		Violations: make([]Violation, 0),
	}

	services := file.GetServices()

	for serviceName, service := range services {
		// Get field order for this service
		fieldOrder := cfg.GetFieldOrder(serviceName)

		// Validate field order
		orderViolations := validateFieldOrder(serviceName, service, fieldOrder, cfg)
		result.Violations = append(result.Violations, orderViolations...)

		// Validate alphabetization
		alphaViolations := validateAlphabetization(serviceName, service, cfg)
		result.Violations = append(result.Violations, alphaViolations...)
	}

	if len(result.Violations) > 0 {
		result.Valid = false
	}

	return result, nil
}

// validateFieldOrder checks if fields are in the correct order
func validateFieldOrder(serviceName string, service parser.Service, fieldOrder []string, cfg *config.Config) []Violation {
	violations := make([]Violation, 0)

	// Get actual fields in the order they appear in the YAML file
	actualFields := make([]string, 0)
	for _, field := range service.FieldOrder {
		// Only check fields that are in our field order
		if isInFieldOrder(field, fieldOrder) {
			actualFields = append(actualFields, field)
		}
	}

	// Filter expected fields to only those present in the service
	expectedFields := make([]string, 0)
	for _, field := range fieldOrder {
		if _, ok := service.Config[field]; ok {
			expectedFields = append(expectedFields, field)
		}
	}

	// Check order
	for i, actual := range actualFields {
		if i < len(expectedFields) && actual != expectedFields[i] {
			violations = append(violations, Violation{
				Type:     "order",
				Service:  serviceName,
				Field:    actual,
				Message:  fmt.Sprintf("field '%s' is out of order", actual),
				Expected: expectedFields[i],
				Actual:   actual,
				Line:     service.Line,
				Column:   service.Column,
			})
		}
	}

	// Check for extra fields if strict mode
	if cfg.Strict {
		for _, field := range service.FieldOrder {
			if !isInFieldOrder(field, fieldOrder) {
				violations = append(violations, Violation{
					Type:    "order",
					Service: serviceName,
					Field:   field,
					Message: fmt.Sprintf("field '%s' is not allowed in strict mode", field),
				})
			}
		}
	}

	return violations
}

// validateAlphabetization checks if environment, volumes, and labels are alphabetized
func validateAlphabetization(serviceName string, service parser.Service, cfg *config.Config) []Violation {
	violations := make([]Violation, 0)

	// Check environment variables
	if cfg.ShouldAlphabetize("environment") {
		if env, ok := service.Config["environment"].([]interface{}); ok {
			if !isAlphabetized(env, extractEnvKey) {
				violations = append(violations, Violation{
					Type:    "alphabetization",
					Service: serviceName,
					Field:   "environment",
					Message: "environment variables are not alphabetized",
					Line:    service.Line,
				})
			}
		} else if envMap, ok := service.Config["environment"].(map[string]interface{}); ok {
			// Environment can also be a map
			if !isMapAlphabetized(envMap) {
				violations = append(violations, Violation{
					Type:    "alphabetization",
					Service: serviceName,
					Field:   "environment",
					Message: "environment variables are not alphabetized",
					Line:    service.Line,
				})
			}
		}
	}

	// Check volumes
	if cfg.ShouldAlphabetize("volumes") {
		if vols, ok := service.Config["volumes"].([]interface{}); ok {
			if !isAlphabetized(vols, extractVolumeKey) {
				violations = append(violations, Violation{
					Type:    "alphabetization",
					Service: serviceName,
					Field:   "volumes",
					Message: "volumes are not alphabetized by source path",
					Line:    service.Line,
				})
			}
		}
	}

	// Check labels
	if cfg.ShouldAlphabetize("labels") {
		if labels, ok := service.Config["labels"].([]interface{}); ok {
			if !isAlphabetized(labels, extractLabelKey) {
				violations = append(violations, Violation{
					Type:    "alphabetization",
					Service: serviceName,
					Field:   "labels",
					Message: "labels are not alphabetized",
					Line:    service.Line,
				})
			}
		} else if labelMap, ok := service.Config["labels"].(map[string]interface{}); ok {
			// Labels can also be a map
			if !isMapAlphabetized(labelMap) {
				violations = append(violations, Violation{
					Type:    "alphabetization",
					Service: serviceName,
					Field:   "labels",
					Message: "labels are not alphabetized",
					Line:    service.Line,
				})
			}
		}
	}

	return violations
}

// isInFieldOrder checks if a field is in the field order list
func isInFieldOrder(field string, fieldOrder []string) bool {
	for _, f := range fieldOrder {
		if f == field {
			return true
		}
	}
	return false
}

// isAlphabetized checks if a slice is alphabetized by the given key extractor
func isAlphabetized(items []interface{}, keyExtractor func(interface{}) string) bool {
	if len(items) < 2 {
		return true
	}

	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, keyExtractor(item))
	}

	sortedKeys := make([]string, len(keys))
	copy(sortedKeys, keys)
	// Sort case-insensitive
	for i := 0; i < len(sortedKeys)-1; i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if strings.ToLower(sortedKeys[i]) > strings.ToLower(sortedKeys[j]) {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}

	for i, key := range keys {
		if key != sortedKeys[i] {
			return false
		}
	}

	return true
}

// isMapAlphabetized checks if map keys are alphabetized
func isMapAlphabetized(m map[string]interface{}) bool {
	if len(m) < 2 {
		return true
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sortedKeys := make([]string, len(keys))
	copy(sortedKeys, keys)
	// Sort case-insensitive
	for i := 0; i < len(sortedKeys)-1; i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if strings.ToLower(sortedKeys[i]) > strings.ToLower(sortedKeys[j]) {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}

	for i, key := range keys {
		if key != sortedKeys[i] {
			return false
		}
	}

	return true
}

// extractEnvKey extracts the key from an environment variable entry
func extractEnvKey(item interface{}) string {
	switch v := item.(type) {
	case string:
		// Handle format "KEY=value" or "KEY"
		if idx := strings.Index(v, "="); idx > 0 {
			return v[:idx]
		}
		return v
	case map[string]interface{}:
		// Handle map format (rare)
		for k := range v {
			return k
		}
	}
	return ""
}

// extractVolumeKey extracts the source path from a volume entry
func extractVolumeKey(item interface{}) string {
	switch v := item.(type) {
	case string:
		// Format: "/host/path:/container/path"
		parts := strings.Split(v, ":")
		if len(parts) > 0 {
			return parts[0]
		}
		return v
	}
	return ""
}

// extractLabelKey extracts the key from a label entry
func extractLabelKey(item interface{}) string {
	switch v := item.(type) {
	case string:
		// Format: "key=value" or just "key"
		if idx := strings.Index(v, "="); idx > 0 {
			return v[:idx]
		}
		return v
	case map[string]interface{}:
		for k := range v {
			return k
		}
	}
	return ""
}
