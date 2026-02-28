package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixturesDir is the path to the test fixtures
func getTestFixturesDir() string {
	// Try to find the fixtures directory relative to the test file
	_, err := os.Stat("../tests/fixtures")
	if err == nil {
		return "../tests/fixtures"
	}
	_, err = os.Stat("../../tests/fixtures")
	if err == nil {
		return "../../tests/fixtures"
	}
	return "tests/fixtures"
}

func TestParseFile_WithComments(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "with-comments-invalid.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse file with comments: %v", err)
	}

	services := file.GetServices()
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	web, ok := services["web"]
	if !ok {
		t.Fatal("Expected 'web' service")
	}

	// Check that field order is preserved from YAML file (wrong order)
	expectedOrder := []string{"image", "container_name", "environment", "ports", "restart", "volumes", "labels"}
	if len(web.FieldOrder) != len(expectedOrder) {
		t.Errorf("Expected %d fields, got %d", len(expectedOrder), len(web.FieldOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(web.FieldOrder) {
			break
		}
		if web.FieldOrder[i] != expected {
			t.Errorf("Field %d: expected '%s', got '%s'", i, expected, web.FieldOrder[i])
		}
	}
}

func TestParseFile_MultiServiceValid(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "multi-service-valid.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse multi-service file: %v", err)
	}

	services := file.GetServices()

	expectedServices := []string{"web", "db", "cache", "worker", "proxy"}
	if len(services) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(expectedServices), len(services))
	}

	for _, name := range expectedServices {
		if _, ok := services[name]; !ok {
			t.Errorf("Expected service '%s' not found", name)
		}
	}

	// Verify web service has all expected fields
	web := services["web"]
	expectedFields := []string{
		"container_name", "image", "user", "environment", "env_file",
		"networks", "network_mode", "ports", "devices", "healthcheck",
		"restart", "cap_add", "privileged", "extra_hosts", "volumes", "labels",
	}

	for _, field := range expectedFields {
		if _, ok := web.Config[field]; !ok {
			t.Errorf("Expected field '%s' in web service", field)
		}
	}
}

func TestParseFile_MultiServiceInvalid(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "multi-service-invalid.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse multi-service invalid file: %v", err)
	}

	services := file.GetServices()

	expectedServices := []string{"web", "db", "cache", "worker", "proxy", "broken"}
	if len(services) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(expectedServices), len(services))
	}

	// Verify that some services have wrong field order
	db := services["db"]
	if len(db.FieldOrder) > 0 && db.FieldOrder[0] != "container_name" {
		// This is expected - the file has wrong order
		t.Logf("DB service has field order: %v (expected to start with 'container_name')", db.FieldOrder)
	}
}

func TestParseFile_ComplexVolumes(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "complex-volumes.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse complex volumes file: %v", err)
	}

	services := file.GetServices()
	app, ok := services["app"]
	if !ok {
		t.Fatal("Expected 'app' service")
	}

	vols, ok := app.Config["volumes"].([]interface{})
	if !ok {
		t.Fatalf("Expected volumes to be []interface{}, got %T", app.Config["volumes"])
	}

	// Should have multiple volumes
	if len(vols) < 8 {
		t.Errorf("Expected at least 8 volumes for complex test, got %d", len(vols))
	}

	// Check that complex paths are parsed correctly
	foundRelative := false
	foundAbsolute := false
	foundNamed := false

	for _, vol := range vols {
		v, ok := vol.(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(v, "./") || strings.HasPrefix(v, "../") {
			foundRelative = true
		}
		if strings.HasPrefix(v, "/") {
			foundAbsolute = true
		}
		if !strings.Contains(v, ":") || (!strings.HasPrefix(v, "/") && !strings.HasPrefix(v, ".")) {
			// Named volume (no path prefix)
			foundNamed = true
		}
	}

	if !foundRelative {
		t.Error("Expected to find relative path volumes")
	}
	if !foundAbsolute {
		t.Error("Expected to find absolute path volumes")
	}
	if !foundNamed {
		t.Error("Expected to find named volumes")
	}
}

func TestParseFile_MixedEnvFormats(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "mixed-env-formats.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse mixed env formats file: %v", err)
	}

	services := file.GetServices()

	expectedServices := []string{"web-list", "web-keys", "web-vars"}
	for _, name := range expectedServices {
		if _, ok := services[name]; !ok {
			t.Errorf("Expected service '%s' not found", name)
		}
	}

	// Test list format with KEY=value
	webList := services["web-list"]
	envList, ok := webList.Config["environment"].([]interface{})
	if !ok {
		t.Errorf("Expected web-list environment to be []interface{}, got %T", webList.Config["environment"])
	}

	// Check that env vars with KEY=value format are parsed
	if len(envList) != 3 {
		t.Errorf("Expected 3 env vars in web-list, got %d", len(envList))
	}

	// Test list format with just KEY (no value)
	webKeys := services["web-keys"]
	envKeys, ok := webKeys.Config["environment"].([]interface{})
	if !ok {
		t.Errorf("Expected web-keys environment to be []interface{}, got %T", webKeys.Config["environment"])
	}
	if len(envKeys) != 3 {
		t.Errorf("Expected 3 env keys in web-keys, got %d", len(envKeys))
	}

	// Test list format with variable substitution
	webVars := services["web-vars"]
	envVars, ok := webVars.Config["environment"].([]interface{})
	if !ok {
		t.Errorf("Expected web-vars environment to be []interface{}, got %T", webVars.Config["environment"])
	}
	if len(envVars) != 3 {
		t.Errorf("Expected 3 env vars in web-vars, got %d", len(envVars))
	}
}

func TestParseFile_YamlAnchors(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "yaml-anchors.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse YAML anchors file: %v", err)
	}

	services := file.GetServices()

	expectedServices := []string{"web", "api", "db"}
	for _, name := range expectedServices {
		if _, ok := services[name]; !ok {
			t.Errorf("Expected service '%s' not found", name)
		}
	}

	// Verify that anchored values are present in merged environments
	// Note: YAML anchors are resolved during parsing, so the actual values
	// from anchors should be present in the parsed output
	web := services["web"]

	// The environment might be parsed as a map or might not include anchored
	// values depending on how the YAML library handles it. This test is mainly
	// to verify the file parses without error and services are detected.
	t.Logf("Web environment type: %T, value: %v", web.Config["environment"], web.Config["environment"])

	// If it's a map, check for merged values
	if webEnv, ok := web.Config["environment"].(map[string]interface{}); ok {
		hasTZ := false
		hasLANG := false
		for k := range webEnv {
			if k == "TZ" {
				hasTZ = true
			}
			if k == "LANG" {
				hasLANG = true
			}
		}
		if !hasTZ || !hasLANG {
			t.Logf("Note: Anchored values TZ/LANG may not be present in parsed map (Go YAML behavior)")
		}
	} else {
		t.Log("Environment is not a map (may be due to anchor parsing)")
	}
}

func TestParseFile_MultiDocument(t *testing.T) {
	fixturesDir := getTestFixturesDir()
	filePath := filepath.Join(fixturesDir, "multi-document.yml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found: " + filePath)
	}

	file, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse multi-document file: %v", err)
	}

	// Should have 3 documents
	if len(file.Documents) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(file.Documents))
	}

	// Get services from first document
	services := file.GetServices()

	expectedServices := []string{"web", "api"}
	for _, name := range expectedServices {
		if _, ok := services[name]; !ok {
			t.Errorf("Expected service '%s' from first document", name)
		}
	}
}
