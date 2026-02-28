package fixer

import (
	"testing"

	"github.com/yourusername/compose-validator/internal/config"
)

func TestAlphabetizeEnvironment_List(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		changed  bool
	}{
		{
			name: "already alphabetized",
			input: []interface{}{
				"AAA=value",
				"BBB=value",
				"CCC=value",
			},
			expected: []interface{}{
				"AAA=value",
				"BBB=value",
				"CCC=value",
			},
			changed: false,
		},
		{
			name: "needs sorting",
			input: []interface{}{
				"ZZZ=value",
				"AAA=value",
				"MMM=value",
			},
			expected: []interface{}{
				"AAA=value",
				"MMM=value",
				"ZZZ=value",
			},
			changed: true,
		},
		{
			name: "case insensitive sorting",
			input: []interface{}{
				"zzz=value",
				"AAA=value",
				"BBB=value",
			},
			expected: []interface{}{
				"AAA=value",
				"BBB=value",
				"zzz=value",
			},
			changed: true,
		},
		{
			name:     "empty list",
			input:    []interface{}{},
			expected: []interface{}{},
			changed:  false,
		},
		{
			name:     "single item",
			input:    []interface{}{"KEY=value"},
			expected: []interface{}{"KEY=value"},
			changed:  false,
		},
		{
			name: "with env var format",
			input: []interface{}{
				"${VAR3}=value3",
				"${VAR1}=value1",
				"${VAR2}=value2",
			},
			expected: []interface{}{
				"${VAR1}=value1",
				"${VAR2}=value2",
				"${VAR3}=value3",
			},
			changed: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, changed := alphabetizeEnvironment(test.input)

			if changed != test.changed {
				t.Errorf("Expected changed=%v, got %v", test.changed, changed)
			}

			if changed {
				resultSlice, ok := result.([]interface{})
				if !ok {
					t.Fatalf("Expected []interface{}, got %T", result)
				}

				expectedSlice, ok := test.expected.([]interface{})
				if !ok {
					t.Fatalf("Expected []interface{} in test data, got %T", test.expected)
				}

				if len(resultSlice) != len(expectedSlice) {
					t.Errorf("Expected %d items, got %d", len(expectedSlice), len(resultSlice))
				}

				for i := range expectedSlice {
					if resultSlice[i] != expectedSlice[i] {
						t.Errorf("Item %d: expected '%v', got '%v'", i, expectedSlice[i], resultSlice[i])
					}
				}
			}
		})
	}
}

func TestAlphabetizeEnvironment_Map(t *testing.T) {
	input := map[string]interface{}{
		"ZZZ": "value3",
		"AAA": "value1",
		"MMM": "value2",
	}

	result, changed := alphabetizeEnvironment(input)

	if !changed {
		t.Error("Expected changed=true for unsorted map")
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", result)
	}

	// Check that keys are in order (Go 1.12+ preserves map iteration order)
	expectedOrder := []string{"AAA", "MMM", "ZZZ"}
	i := 0
	for key := range resultMap {
		if i >= len(expectedOrder) {
			t.Error("More keys than expected")
			break
		}
		if key != expectedOrder[i] {
			t.Errorf("Key %d: expected '%s', got '%s'", i, expectedOrder[i], key)
		}
		i++
	}
}

func TestAlphabetizeVolumes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
		changed  bool
	}{
		{
			name: "already alphabetized",
			input: []interface{}{
				"/aaa:/container/aaa",
				"/bbb:/container/bbb",
				"/ccc:/container/ccc",
			},
			expected: []string{
				"/aaa:/container/aaa",
				"/bbb:/container/bbb",
				"/ccc:/container/ccc",
			},
			changed: false,
		},
		{
			name: "needs sorting by source",
			input: []interface{}{
				"/zzz:/container/zzz",
				"/aaa:/container/aaa",
				"/mmm:/container/mmm",
			},
			expected: []string{
				"/aaa:/container/aaa",
				"/mmm:/container/mmm",
				"/zzz:/container/zzz",
			},
			changed: true,
		},
		{
			name: "complex paths",
			input: []interface{}{
				"/var/log:/container/log",
				"/etc/config:/container/config",
				"/home/data:/container/data",
			},
			expected: []string{
				"/etc/config:/container/config",
				"/home/data:/container/data",
				"/var/log:/container/log",
			},
			changed: true,
		},
		{
			name:     "empty volumes",
			input:    []interface{}{},
			expected: []string{},
			changed:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, changed := alphabetizeVolumes(test.input)

			if changed != test.changed {
				t.Errorf("Expected changed=%v, got %v", test.changed, changed)
			}

			if changed {
				resultSlice, ok := result.([]interface{})
				if !ok {
					t.Fatalf("Expected []interface{}, got %T", result)
				}

				if len(resultSlice) != len(test.expected) {
					t.Errorf("Expected %d items, got %d", len(test.expected), len(resultSlice))
				}

				for i, expected := range test.expected {
					if resultSlice[i] != expected {
						t.Errorf("Item %d: expected '%v', got '%v'", i, expected, resultSlice[i])
					}
				}
			}
		})
	}
}

func TestAlphabetizeLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
		changed  bool
	}{
		{
			name: "already alphabetized",
			input: []interface{}{
				"traefik.enable=true",
				"traefik.http.routers.app.rule=Host(`example.com`)",
				"wud.watch=true",
			},
			expected: []string{
				"traefik.enable=true",
				"traefik.http.routers.app.rule=Host(`example.com`)",
				"wud.watch=true",
			},
			changed: false,
		},
		{
			name: "needs sorting",
			input: []interface{}{
				"wud.watch=true",
				"traefik.enable=true",
				"com.example.label=value",
			},
			expected: []string{
				"com.example.label=value",
				"traefik.enable=true",
				"wud.watch=true",
			},
			changed: true,
		},
		{
			name: "without values",
			input: []interface{}{
				"zzz.label",
				"aaa.label",
				"mmm.label",
			},
			expected: []string{
				"aaa.label",
				"mmm.label",
				"zzz.label",
			},
			changed: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, changed := alphabetizeLabels(test.input)

			if changed != test.changed {
				t.Errorf("Expected changed=%v, got %v", test.changed, changed)
			}

			if changed {
				resultSlice, ok := result.([]interface{})
				if !ok {
					t.Fatalf("Expected []interface{}, got %T", result)
				}

				if len(resultSlice) != len(test.expected) {
					t.Errorf("Expected %d items, got %d", len(test.expected), len(resultSlice))
				}

				for i, expected := range test.expected {
					if resultSlice[i] != expected {
						t.Errorf("Item %d: expected '%v', got '%v'", i, expected, resultSlice[i])
					}
				}
			}
		})
	}
}

func TestExtractEnvKey(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"KEY=value", "KEY"},
		{"KEY", "KEY"},
		{"${VAR}=value", "${VAR}"},
		{map[string]interface{}{"KEY": "value"}, "KEY"},
		{123, ""},
	}

	for _, test := range tests {
		result := extractEnvKey(test.input)
		if result != test.expected {
			t.Errorf("extractEnvKey(%v) = '%s', expected '%s'", test.input, result, test.expected)
		}
	}
}

func TestExtractVolumeKey(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"/host:/container", "/host"},
		{"/host:/container:ro", "/host"},
		{"/host", "/host"},
		{"named_volume:/container", "named_volume"},
		{123, ""},
	}

	for _, test := range tests {
		result := extractVolumeKey(test.input)
		if result != test.expected {
			t.Errorf("extractVolumeKey(%v) = '%s', expected '%s'", test.input, result, test.expected)
		}
	}
}

func TestExtractLabelKey(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"key=value", "key"},
		{"key", "key"},
		{"traefik.http.routers.app.rule=Host(`example.com`)", "traefik.http.routers.app.rule"},
		{map[string]interface{}{"key": "value"}, "key"},
		{123, ""},
	}

	for _, test := range tests {
		result := extractLabelKey(test.input)
		if result != test.expected {
			t.Errorf("extractLabelKey(%v) = '%s', expected '%s'", test.input, result, test.expected)
		}
	}
}

func TestIsFieldOrderCorrect(t *testing.T) {
	cfg := config.NewDefaultConfig()

	tests := []struct {
		name     string
		svc      map[string]interface{}
		expected bool
	}{
		{
			name: "correct order",
			svc: map[string]interface{}{
				"container_name": "app",
				"image":          "nginx",
				"environment":    []interface{}{},
			},
			expected: true,
		},
		{
			name: "wrong order - image before container_name",
			svc: map[string]interface{}{
				"image":          "nginx",
				"container_name": "app",
				"environment":    []interface{}{},
			},
			expected: false,
		},
		{
			name: "missing fields",
			svc: map[string]interface{}{
				"container_name": "app",
				"image":          "nginx",
			},
			expected: true, // Still correct since we only check present fields
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isFieldOrderCorrect(test.svc, cfg.FieldOrder)
			if result != test.expected {
				t.Errorf("isFieldOrderCorrect() = %v, expected %v", result, test.expected)
			}
		})
	}
}
