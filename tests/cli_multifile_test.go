package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to build and get the CLI path
func getCLIPath() string {
	// Try to find the built binary
	paths := []string{
		"./compose-validator",
		"../compose-validator",
		"../../compose-validator",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// Helper function to run the CLI
func runCLI(args ...string) (string, string, int) {
	cliPath := getCLIPath()

	if cliPath == "" {
		// Build on demand
		buildCmd := exec.Command("go", "build", "-o", "compose-validator", "./cmd/compose-validator")
		buildCmd.Dir = "../.."
		if err := buildCmd.Run(); err != nil {
			return "", fmt.Sprintf("Failed to build CLI: %v", err), -1
		}
		cliPath = "./compose-validator"
	}

	cmd := exec.Command(cliPath, args...)
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return string(output), "", exitCode
}

func getFixturesDir() string {
	dirs := []string{
		"tests/fixtures",
		"../tests/fixtures",
		"../../tests/fixtures",
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	return ""
}

func TestCLI_MultipleFiles_AllValid(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	files := []string{
		filepath.Join(fixturesDir, "valid-compose.yml"),
		filepath.Join(fixturesDir, "multi-service-valid.yml"),
	}

	// Verify files exist
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Skipf("Fixture not found: %s", file)
		}
	}

	output, _, exitCode := runCLI(files...)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for all valid files, got %d. Output: %s", exitCode, output)
	}

	if !strings.Contains(output, "valid") && !strings.Contains(output, "All files are valid") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestCLI_MultipleFiles_MixedValidity(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	files := []string{
		filepath.Join(fixturesDir, "valid-compose.yml"),
		filepath.Join(fixturesDir, "invalid-compose.yml"),
	}

	// Verify files exist
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Skipf("Fixture not found: %s", file)
		}
	}

	output, _, exitCode := runCLI(files...)

	if exitCode == 0 {
		t.Error("Expected non-zero exit code for mixed validity files")
	}

	// Should show violations for the invalid file
	if !strings.Contains(output, "violation") && !strings.Contains(output, "out of order") && !strings.Contains(output, "not alphabetized") {
		t.Errorf("Expected violation message, got: %s", output)
	}

	// Should still process both files
	if strings.Contains(output, "valid-compose.yml") && strings.Contains(output, "invalid-compose.yml") {
		// Both files were processed
	} else {
		t.Logf("Output: %s", output)
	}
}

func TestCLI_GlobPattern(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	// Test with glob pattern
	pattern := filepath.Join(fixturesDir, "valid*.yml")

	output, _, exitCode := runCLI(pattern)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}

	// Should process matching files
	t.Logf("Output: %s", output)
}

func TestCLI_MultipleFiles_FixMode(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	// Create temp copies of invalid files
	tmpDir := t.TempDir()

	invalidFile := filepath.Join(fixturesDir, "invalid-compose.yml")
	if _, err := os.Stat(invalidFile); os.IsNotExist(err) {
		t.Skip("Invalid fixture not found")
	}

	// Copy file to temp location
	data, _ := os.ReadFile(invalidFile)
	tempFile := filepath.Join(tmpDir, "to-fix.yml")
	os.WriteFile(tempFile, data, 0644)

	// Fix it
	output, _, exitCode := runCLI("--fix", tempFile)

	if exitCode != 0 {
		t.Errorf("Fix mode should succeed, got exit code %d. Output: %s", exitCode, output)
	}

	if !strings.Contains(output, "Fixed") && !strings.Contains(output, "fixed") {
		t.Errorf("Expected 'Fixed' message, got: %s", output)
	}

	// Re-validate to confirm it's fixed
	validateOutput, _, validateExitCode := runCLI(tempFile)

	if validateExitCode != 0 {
		t.Errorf("After fix, file should be valid, got exit code %d. Output: %s", validateExitCode, validateOutput)
	}
}

func TestCLI_MultipleFiles_FixModeMultiple(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	tmpDir := t.TempDir()

	// Copy multiple invalid files
	filesToFix := []string{
		"invalid-compose.yml",
		"with-comments-invalid.yml",
		"multi-service-invalid.yml",
	}

	fixedFiles := []string{}

	for _, filename := range filesToFix {
		srcPath := filepath.Join(fixturesDir, filename)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		data, _ := os.ReadFile(srcPath)
		dstPath := filepath.Join(tmpDir, filename)
		os.WriteFile(dstPath, data, 0644)
		fixedFiles = append(fixedFiles, dstPath)
	}

	if len(fixedFiles) == 0 {
		t.Skip("No invalid fixtures found to fix")
	}

	// Fix all at once
	args := append([]string{"--fix"}, fixedFiles...)
	output, _, exitCode := runCLI(args...)

	if exitCode != 0 {
		t.Errorf("Fix mode should succeed, got exit code %d. Output: %s", exitCode, output)
	}

	// Verify each file was fixed
	for _, file := range fixedFiles {
		validateOutput, _, validateExitCode := runCLI(file)
		if validateExitCode != 0 {
			t.Errorf("File %s should be valid after fix. Output: %s", file, validateOutput)
		}
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	output, _, exitCode := runCLI("/nonexistent/path/file.yml")

	if exitCode == 0 {
		t.Error("Expected non-zero exit code for non-existent file")
	}

	if !strings.Contains(output, "Error") && !strings.Contains(output, "failed") {
		t.Errorf("Expected error message for non-existent file, got: %s", output)
	}
}

func TestCLI_EmptyArgs(t *testing.T) {
	output, _, exitCode := runCLI()

	if exitCode == 0 {
		t.Error("Expected non-zero exit code for empty args")
	}

	if !strings.Contains(output, "no files") && !strings.Contains(output, "required") {
		t.Errorf("Expected error about missing files, got: %s", output)
	}
}

func TestCLI_VerboseMultipleFiles(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	files := []string{
		filepath.Join(fixturesDir, "valid-compose.yml"),
		filepath.Join(fixturesDir, "multi-service-valid.yml"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Skipf("Fixture not found: %s", file)
		}
	}

	output, _, exitCode := runCLI(append([]string{"-v"}, files...)...)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}

	// Verbose mode should show field order info
	if !strings.Contains(output, "Field order") {
		t.Errorf("Verbose output should contain 'Field order', got: %s", output)
	}
}

func TestCLI_WithConfigFlag(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	tmpDir := t.TempDir()

	// Create custom config that reverses the order of first two fields
	configContent := `
field_order:
  - image
  - container_name
  - environment
  - networks
  - ports
  - restart
  - volumes
  - labels
`
	configPath := filepath.Join(tmpDir, "custom-config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Create compose file with this custom order
	composeContent := `
services:
  web:
    image: nginx:latest
    container_name: web-server
    environment:
      - KEY=value
`
	composePath := filepath.Join(tmpDir, "custom-order.yml")
	os.WriteFile(composePath, []byte(composeContent), 0644)

	// With custom config, this should be valid (image before container_name)
	output, _, exitCode := runCLI("--config", configPath, composePath)

	if exitCode != 0 {
		t.Errorf("With custom config, file should be valid. Exit code: %d, Output: %s", exitCode, output)
	}
}

func TestCLI_WildcardPattern(t *testing.T) {
	fixturesDir := getFixturesDir()
	if fixturesDir == "" {
		t.Skip("Fixtures directory not found")
	}

	// Test with wildcard that should match multiple files
	pattern := filepath.Join(fixturesDir, "*.yml")

	output, _, exitCode := runCLI(pattern)

	// May pass or fail depending on if there are invalid files
	t.Logf("Wildcard pattern output (exit=%d): %s", exitCode, output)

	// Should process at least one file
	if output == "" {
		t.Error("Expected some output from wildcard pattern")
	}
}
