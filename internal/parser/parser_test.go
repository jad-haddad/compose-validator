package parser

import (
	"testing"
)

func TestParseBytes_ValidSingleDocument(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    container_name: web-server
    image: nginx:latest
    ports:
      - "8080:80"
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if file.Path != "test.yml" {
		t.Errorf("Expected path 'test.yml', got '%s'", file.Path)
	}

	if len(file.Documents) != 1 {
		t.Errorf("Expected 1 document, got %d", len(file.Documents))
	}
}

func TestParseBytes_MultiDocument(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    image: nginx:latest
---
version: '3.8'
services:
  db:
    image: postgres:latest
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(file.Documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(file.Documents))
	}
}

func TestParseBytes_InvalidYAML(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
    ports: [invalid
`

	_, err := ParseBytes("test.yml", []byte(yaml))
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestGetServices_SingleService(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    container_name: web-server
    image: nginx:latest
    ports:
      - "8080:80"
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	web, ok := services["web"]
	if !ok {
		t.Fatal("Expected 'web' service")
	}

	if web.Name != "web" {
		t.Errorf("Expected service name 'web', got '%s'", web.Name)
	}

	if web.Config["container_name"] != "web-server" {
		t.Errorf("Expected container_name 'web-server', got '%v'", web.Config["container_name"])
	}
}

func TestGetServices_MultipleServices(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
  db:
    image: postgres:latest
  cache:
    image: redis:latest
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()

	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}

	expectedServices := []string{"web", "db", "cache"}
	for _, name := range expectedServices {
		if _, ok := services[name]; !ok {
			t.Errorf("Expected service '%s' not found", name)
		}
	}
}

func TestGetServices_NoServices(t *testing.T) {
	yaml := `
version: '3.8'
networks:
  mynet:
    driver: bridge
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()

	if len(services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(services))
	}
}

func TestGetServices_MultiDocument(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
---
services:
  db:
    image: postgres:latest
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()

	if len(services) != 2 {
		t.Errorf("Expected 2 services from multi-document, got %d", len(services))
	}

	if _, ok := services["web"]; !ok {
		t.Error("Expected 'web' service from first document")
	}

	if _, ok := services["db"]; !ok {
		t.Error("Expected 'db' service from second document")
	}
}

func TestGetServices_PreservesFieldOrder(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
    container_name: web-server
    environment:
      - KEY=value
    ports:
      - "8080:80"
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()
	web := services["web"]

	// Field order should match YAML file order
	expectedOrder := []string{"image", "container_name", "environment", "ports"}

	if len(web.FieldOrder) != len(expectedOrder) {
		t.Errorf("Expected %d fields in order, got %d", len(expectedOrder), len(web.FieldOrder))
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

func TestGetServices_ComplexConfig(t *testing.T) {
	yaml := `
services:
  web:
    container_name: web-server
    image: nginx:latest
    environment:
      - TZ=Europe/Paris
      - DEBUG=true
    env_file:
      - .env
    networks:
      - web
    ports:
      - "8080:80"
      - "8443:443"
    volumes:
      - /data:/var/www/html
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.web.rule=Host('example.com')"
    restart: always
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()
	web := services["web"]

	// Check all fields are parsed
	expectedFields := []string{
		"container_name",
		"image",
		"environment",
		"env_file",
		"networks",
		"ports",
		"volumes",
		"labels",
		"restart",
	}

	for _, field := range expectedFields {
		if _, ok := web.Config[field]; !ok {
			t.Errorf("Expected field '%s' not found in config", field)
		}
	}

	// Check arrays are parsed correctly
	env, ok := web.Config["environment"].([]interface{})
	if !ok {
		t.Errorf("Expected environment to be []interface{}, got %T", web.Config["environment"])
	} else if len(env) != 2 {
		t.Errorf("Expected 2 environment variables, got %d", len(env))
	}

	ports, ok := web.Config["ports"].([]interface{})
	if !ok {
		t.Errorf("Expected ports to be []interface{}, got %T", web.Config["ports"])
	} else if len(ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(ports))
	}
}

func TestGetServices_EmptyService(t *testing.T) {
	yaml := `
services:
  empty:
    image: nginx:latest
`

	file, err := ParseBytes("test.yml", []byte(yaml))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	services := file.GetServices()

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	empty := services["empty"]
	if len(empty.FieldOrder) != 1 {
		t.Errorf("Expected 1 field in order, got %v", empty.FieldOrder)
	}
}
