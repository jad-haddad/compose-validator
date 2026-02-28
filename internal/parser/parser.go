package parser

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// ComposeFile represents a parsed Docker Compose file
type ComposeFile struct {
	Path      string
	AST       *ast.File
	Documents []*ast.DocumentNode
	RawData   []byte
}

// Service represents a Docker Compose service
type Service struct {
	Name       string
	Config     map[string]interface{}
	FieldOrder []string // Original field order from YAML
	// Position information
	Line   int
	Column int
}

// ParseFile parses a Docker Compose YAML file
func ParseFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return ParseBytes(path, data)
}

// ParseBytes parses Docker Compose YAML from bytes
func ParseBytes(path string, data []byte) (*ComposeFile, error) {
	file, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	composeFile := &ComposeFile{
		Path:      path,
		AST:       file,
		Documents: file.Docs,
		RawData:   data,
	}

	return composeFile, nil
}

// GetServices extracts services from all documents
func (cf *ComposeFile) GetServices() map[string]Service {
	services := make(map[string]Service)

	for _, doc := range cf.Documents {
		if doc == nil || doc.Body == nil {
			continue
		}

		// Get the document as a mapping node
		mapping, ok := doc.Body.(*ast.MappingNode)
		if !ok {
			continue
		}

		// Find the services key
		var servicesNode *ast.MappingNode
		for _, val := range mapping.Values {
			if val.Key.String() == "services" {
				if svcMap, ok := val.Value.(*ast.MappingNode); ok {
					servicesNode = svcMap
					break
				}
			}
		}

		if servicesNode == nil {
			continue
		}

		// Extract each service
		for _, svcVal := range servicesNode.Values {
			svcName := svcVal.Key.String()
			svcMapping, ok := svcVal.Value.(*ast.MappingNode)
			if !ok {
				continue
			}

			// Extract field order from AST
			fieldOrder := make([]string, 0, len(svcMapping.Values))
			svcConfig := make(map[string]interface{})

			for _, field := range svcMapping.Values {
				fieldName := field.Key.String()
				fieldOrder = append(fieldOrder, fieldName)

				// Decode the value using yaml.Unmarshal
				var value interface{}
				if err := yaml.Unmarshal([]byte(field.Value.String()), &value); err == nil {
					svcConfig[fieldName] = value
				} else {
					// Fallback: try to decode as string
					svcConfig[fieldName] = field.Value.String()
				}
			}

			services[svcName] = Service{
				Name:       svcName,
				Config:     svcConfig,
				FieldOrder: fieldOrder,
				Line:       svcVal.Key.GetToken().Position.Line,
				Column:     svcVal.Key.GetToken().Position.Column,
			}
		}
	}

	return services
}

// GetDocumentContent returns the content of a document as a map
func (cf *ComposeFile) GetDocumentContent(doc *ast.DocumentNode) (map[string]interface{}, error) {
	if doc == nil || doc.Body == nil {
		return nil, fmt.Errorf("empty document")
	}

	var content map[string]interface{}
	if err := yaml.Unmarshal([]byte(doc.String()), &content); err != nil {
		return nil, err
	}

	return content, nil
}
