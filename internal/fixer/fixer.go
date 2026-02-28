package fixer

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/yourusername/compose-validator/internal/config"
	"github.com/yourusername/compose-validator/internal/parser"
)

// FixResult represents the result of a fix operation
type FixResult struct {
	File    string
	Fixed   bool
	Changes []string
	Error   error
}

// Fix repairs violations in a Docker Compose file
func Fix(file *parser.ComposeFile, cfg *config.Config) (*FixResult, error) {
	result := &FixResult{
		File:    file.Path,
		Changes: make([]string, 0),
	}

	// Parse the file content
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file.Path, err)
	}

	// Parse into generic structure
	var composeContent map[string]interface{}
	if err := yaml.Unmarshal(data, &composeContent); err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", file.Path, err)
	}

	// Get services
	services, ok := composeContent["services"].(map[string]interface{})
	if !ok {
		// No services to fix
		result.Fixed = false
		return result, nil
	}

	fixed := false
	for serviceName, svc := range services {
		if svcMap, ok := svc.(map[string]interface{}); ok {
			fieldOrder := cfg.GetFieldOrder(serviceName)
			svcFixed, svcChanges := fixService(serviceName, svcMap, fieldOrder, cfg)
			if svcFixed {
				fixed = true
				result.Changes = append(result.Changes, svcChanges...)
				services[serviceName] = svcMap
			}
		}
	}

	if fixed {
		result.Fixed = true

		// Marshal back to YAML with comment preservation
		// For now, we'll use standard marshaling
		output, err := yaml.Marshal(composeContent)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal fixed content: %w", err)
		}

		// Write back to file
		if err := os.WriteFile(file.Path, output, 0644); err != nil {
			return nil, fmt.Errorf("failed to write fixed file %s: %w", file.Path, err)
		}
	}

	return result, nil
}

// FixBytes fixes violations in YAML bytes
func FixBytes(data []byte, cfg *config.Config) ([]byte, []string, error) {
	// Parse into generic structure
	var composeContent map[string]interface{}
	if err := yaml.Unmarshal(data, &composeContent); err != nil {
		return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Get services
	services, ok := composeContent["services"].(map[string]interface{})
	if !ok {
		// No services to fix
		return data, nil, nil
	}

	changes := make([]string, 0)
	fixed := false

	for serviceName, svc := range services {
		if svcMap, ok := svc.(map[string]interface{}); ok {
			fieldOrder := cfg.GetFieldOrder(serviceName)
			svcFixed, svcChanges := fixService(serviceName, svcMap, fieldOrder, cfg)
			if svcFixed {
				fixed = true
				changes = append(changes, svcChanges...)
				services[serviceName] = svcMap
			}
		}
	}

	if !fixed {
		return data, nil, nil
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(composeContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal fixed content: %w", err)
	}

	return output, changes, nil
}

// fixService repairs a single service configuration
func fixService(name string, svc map[string]interface{}, fieldOrder []string, cfg *config.Config) (bool, []string) {
	changes := make([]string, 0)
	fixed := false

	// Create ordered map
	ordered := make(map[string]interface{}, len(svc))

	// Add fields in correct order
	for _, field := range fieldOrder {
		if value, ok := svc[field]; ok {
			// Alphabetize if needed
			alphabetizedValue, alphaFixed := alphabetizeField(field, value, cfg)
			if alphaFixed {
				changes = append(changes, fmt.Sprintf("service '%s': alphabetized '%s'", name, field))
				fixed = true
			}

			ordered[field] = alphabetizedValue
			delete(svc, field)
		}
	}

	// Add any remaining fields (not in field order) at the end
	for field, value := range svc {
		alphabetizedValue, alphaFixed := alphabetizeField(field, value, cfg)
		if alphaFixed {
			changes = append(changes, fmt.Sprintf("service '%s': alphabetized '%s'", name, field))
			fixed = true
		}
		ordered[field] = alphabetizedValue
	}

	// Check if field order was wrong
	if !isFieldOrderCorrect(svc, fieldOrder) {
		changes = append(changes, fmt.Sprintf("service '%s': reordered fields", name))
		fixed = true
	}

	// Copy ordered back to svc
	for field, value := range ordered {
		svc[field] = value
	}

	return fixed, changes
}

// isFieldOrderCorrect checks if fields are in the expected order
func isFieldOrderCorrect(svc map[string]interface{}, fieldOrder []string) bool {
	lastIndex := -1
	for field := range svc {
		idx := getFieldIndex(field, fieldOrder)
		if idx >= 0 {
			if idx < lastIndex {
				return false
			}
			lastIndex = idx
		}
	}
	return true
}

// getFieldIndex returns the index of a field in the field order
func getFieldIndex(field string, fieldOrder []string) int {
	for i, f := range fieldOrder {
		if f == field {
			return i
		}
	}
	return -1
}

// alphabetizeField alphabetizes a field's value if needed
func alphabetizeField(field string, value interface{}, cfg *config.Config) (interface{}, bool) {
	if !cfg.ShouldAlphabetize(field) {
		return value, false
	}

	switch field {
	case "environment":
		return alphabetizeEnvironment(value)
	case "volumes":
		return alphabetizeVolumes(value)
	case "labels":
		return alphabetizeLabels(value)
	}

	return value, false
}

// alphabetizeEnvironment alphabetizes environment variables
func alphabetizeEnvironment(value interface{}) (interface{}, bool) {
	switch v := value.(type) {
	case []interface{}:
		if len(v) < 2 {
			return value, false
		}

		sorted := make([]interface{}, len(v))
		copy(sorted, v)

		sort.Slice(sorted, func(i, j int) bool {
			keyI := extractEnvKey(sorted[i])
			keyJ := extractEnvKey(sorted[j])
			return strings.ToLower(keyI) < strings.ToLower(keyJ)
		})

		// Check if sorting changed anything
		for i := range v {
			if v[i] != sorted[i] {
				return sorted, true
			}
		}
		return value, false

	case map[string]interface{}:
		if len(v) < 2 {
			return value, false
		}

		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}

		sortedKeys := make([]string, len(keys))
		copy(sortedKeys, keys)
		sort.Slice(sortedKeys, func(i, j int) bool {
			return strings.ToLower(sortedKeys[i]) < strings.ToLower(sortedKeys[j])
		})

		// Check if keys were already sorted
		sorted := true
		for i, key := range keys {
			if key != sortedKeys[i] {
				sorted = false
				break
			}
		}

		if sorted {
			return value, false
		}

		// Create new ordered map
		ordered := make(map[string]interface{}, len(v))
		for _, key := range sortedKeys {
			ordered[key] = v[key]
		}
		return ordered, true
	}

	return value, false
}

// alphabetizeVolumes alphabetizes volumes by source path
func alphabetizeVolumes(value interface{}) (interface{}, bool) {
	switch v := value.(type) {
	case []interface{}:
		if len(v) < 2 {
			return value, false
		}

		sorted := make([]interface{}, len(v))
		copy(sorted, v)

		sort.Slice(sorted, func(i, j int) bool {
			sourceI := extractVolumeKey(sorted[i])
			sourceJ := extractVolumeKey(sorted[j])
			return strings.ToLower(sourceI) < strings.ToLower(sourceJ)
		})

		// Check if sorting changed anything
		for i := range v {
			if v[i] != sorted[i] {
				return sorted, true
			}
		}
		return value, false
	}

	return value, false
}

// alphabetizeLabels alphabetizes labels by key
func alphabetizeLabels(value interface{}) (interface{}, bool) {
	switch v := value.(type) {
	case []interface{}:
		if len(v) < 2 {
			return value, false
		}

		sorted := make([]interface{}, len(v))
		copy(sorted, v)

		sort.Slice(sorted, func(i, j int) bool {
			keyI := extractLabelKey(sorted[i])
			keyJ := extractLabelKey(sorted[j])
			return strings.ToLower(keyI) < strings.ToLower(keyJ)
		})

		// Check if sorting changed anything
		for i := range v {
			if v[i] != sorted[i] {
				return sorted, true
			}
		}
		return value, false

	case map[string]interface{}:
		// Similar to environment map handling
		if len(v) < 2 {
			return value, false
		}

		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}

		sortedKeys := make([]string, len(keys))
		copy(sortedKeys, keys)
		sort.Slice(sortedKeys, func(i, j int) bool {
			return strings.ToLower(sortedKeys[i]) < strings.ToLower(sortedKeys[j])
		})

		sorted := true
		for i, key := range keys {
			if key != sortedKeys[i] {
				sorted = false
				break
			}
		}

		if sorted {
			return value, false
		}

		ordered := make(map[string]interface{}, len(v))
		for _, key := range sortedKeys {
			ordered[key] = v[key]
		}
		return ordered, true
	}

	return value, false
}

// extractEnvKey extracts the key from an environment variable entry
func extractEnvKey(item interface{}) string {
	switch v := item.(type) {
	case string:
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

// extractVolumeKey extracts the source path from a volume entry
func extractVolumeKey(item interface{}) string {
	switch v := item.(type) {
	case string:
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
