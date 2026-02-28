//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/compose-validator", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "compose-validator") {
		t.Errorf("Expected version output to contain 'compose-validator', got: %s", outputStr)
	}
}

func TestCLI_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/compose-validator", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	expectedFlags := []string{"--fix", "--verbose", "--config"}
	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Expected help output to contain '%s'", flag)
		}
	}
}

func TestCLI_ValidFile(t *testing.T) {
	// Create temp valid compose file
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "valid.yml")
	content := `
services:
  web:
    container_name: web-server
    image: nginx:latest
    environment:
      - AAA=value
      - BBB=value
    ports:
      - "8080:80"
    restart: always
`
	if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/compose-validator", validFile)
	output, err := cmd.CombinedOutput()

	// Command should succeed with exit code 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() != 0 {
			t.Errorf("Expected exit code 0 for valid file, got output: %s", output)
		}
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "valid") && !strings.Contains(outputStr, "All files are valid") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
}

func TestCLI_InvalidFile(t *testing.T) {
	// Create temp invalid compose file
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yml")
	content := `
services:
  web:
    image: nginx:latest
    container_name: web-server
    environment:
      - ZZZ=value
      - AAA=value
`
	if err := os.WriteFile(invalidFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/compose-validator", invalidFile)
	output, err := cmd.CombinedOutput()

	// Command should fail with non-zero exit code
	if err == nil {
		t.Error("Expected non-zero exit code for invalid file")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Error("Expected ExitError with non-zero code")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "out of order") && !strings.Contains(outputStr, "not alphabetized") {
		t.Errorf("Expected error message about violations, got: %s", outputStr)
	}
}

func TestCLI_FixMode(t *testing.T) {
	// Create temp invalid compose file
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "to-fix.yml")
	content := `
services:
  web:
    image: nginx:latest
    container_name: web-server
    environment:
      - ZZZ=value
      - AAA=value
`
	if err := os.WriteFile(invalidFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/compose-validator", "--fix", invalidFile)
	output, err := cmd.CombinedOutput()

	// Fix mode should succeed
	if err != nil {
		t.Errorf("Fix mode should succeed, got error: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Fixed") {
		t.Errorf("Expected 'Fixed' message, got: %s", outputStr)
	}

	// Verify the file was fixed
	fixedContent, err := os.ReadFile(invalidFile)
	if err != nil {
		t.Fatalf("Failed to read fixed file: %v", err)
	}

	// Check that container_name comes before image now
	contentStr := string(fixedContent)
	containerNameIdx := strings.Index(contentStr, "container_name:")
	imageIdx := strings.Index(contentStr, "image:")

	if containerNameIdx == -1 || imageIdx == -1 {
		t.Error("Fixed file should contain both container_name and image")
	}

	if containerNameIdx > imageIdx {
		t.Error("container_name should come before image in fixed file")
	}

	// Verify environment is alphabetized
	envZZZIdx := strings.Index(contentStr, "ZZZ=value")
	envAAAIdx := strings.Index(contentStr, "AAA=value")

	if envAAAIdx > envZZZIdx {
		t.Error("AAA should come before ZZZ in fixed file")
	}
}

func TestCLI_NoArgs(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/compose-validator")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected error when no files provided")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "no files") && !strings.Contains(outputStr, "required") {
		t.Errorf("Expected error message about missing files, got: %s", outputStr)
	}
}

func TestCLI_Verbose(t *testing.T) {
	// Create temp valid compose file
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "verbose-test.yml")
	content := `
services:
  web:
    container_name: web-server
    image: nginx:latest
`
	if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/compose-validator", "-v", validFile)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Verbose mode should succeed, got error: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	// In verbose mode, we should see configuration info
	if !strings.Contains(outputStr, "Field order") && !strings.Contains(outputStr, "valid") {
		t.Errorf("Expected verbose output with field order info, got: %s", outputStr)
	}
}

func TestCLI_ConfigFlag(t *testing.T) {
	// Create temp directory with custom config
	tmpDir := t.TempDir()

	// Create custom config
	configFile := filepath.Join(tmpDir, "custom-config.yaml")
	configContent := `
field_order:
  - image
  - container_name
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create compose file with custom order
	composeFile := filepath.Join(tmpDir, "compose.yml")
	composeContent := `
services:
  web:
    image: nginx:latest
    container_name: web-server
`
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to create compose file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/compose-validator", "--config", configFile, composeFile)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("With custom config (image before container_name), this should be valid, got error: %v\nOutput: %s", err, output)
	}
}

func TestCLI_GlobPattern(t *testing.T) {
	// Create temp directory with multiple files
	tmpDir := t.TempDir()

	for i := 1; i <= 3; i++ {
		file := filepath.Join(tmpDir, fmt.Sprintf("docker-compose-%d.yml", i))
		content := fmt.Sprintf(`
services:
  app%d:
    container_name: app%d
    image: nginx:latest
`, i, i)
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	cmd := exec.Command("go", "run", "./cmd/compose-validator", filepath.Join(tmpDir, "docker-compose-*.yml"))
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Glob pattern should match valid files, got error: %v\nOutput: %s", err, output)
	}

	// All 3 files should be processed
	outputStr := string(output)
	if !strings.Contains(outputStr, "app1") || !strings.Contains(outputStr, "app2") || !strings.Contains(outputStr, "app3") {
		if !strings.Contains(outputStr, "All files are valid") {
			t.Errorf("Expected all 3 files to be processed, got: %s", outputStr)
		}
	}
}
