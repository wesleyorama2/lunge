package jsonschema

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestValidateWithSchema(t *testing.T) {
	// Create a temporary schema file for testing
	tempSchemaContent := []byte(`{
		"type": "object",
		"required": ["id", "name"],
		"properties": {
			"id": { "type": "integer" },
			"name": { "type": "string" }
		}
	}`)

	tempFile, err := os.CreateTemp("", "schema-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(tempSchemaContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test cases
	tests := []struct {
		name           string
		json           string
		schemaURL      string
		expectedValid  bool
		expectedErrors bool
	}{
		{
			name:           "Valid JSON with local schema",
			json:           `{"id": 1, "name": "Test User"}`,
			schemaURL:      tempFile.Name(),
			expectedValid:  true,
			expectedErrors: false,
		},
		{
			name:           "Invalid JSON with local schema",
			json:           `{"id": "not-an-integer", "name": "Test User"}`,
			schemaURL:      tempFile.Name(),
			expectedValid:  false,
			expectedErrors: true,
		},
		{
			name:           "Non-existent schema file",
			json:           `{"id": 1, "name": "Test User"}`,
			schemaURL:      "non-existent-schema.json",
			expectedValid:  false,
			expectedErrors: true,
		},
		{
			name:           "Invalid JSON syntax",
			json:           `{"id": 1, "name": "Test User"`,
			schemaURL:      tempFile.Name(),
			expectedValid:  false,
			expectedErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate with schema
			valid, errors := ValidateWithSchema(tt.json, tt.schemaURL)

			// Check validity
			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectedValid, valid)
			}

			// Check errors
			if (len(errors) > 0) != tt.expectedErrors {
				t.Errorf("Expected errors=%v, got %v", tt.expectedErrors, len(errors) > 0)
			}
		})
	}
}

func TestExtractValidationErrors(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		json          string
		schema        string
		expectedCount int
	}{
		{
			name: "Multiple validation errors",
			schema: `{
				"type": "object",
				"required": ["id", "name", "email"],
				"properties": {
					"id": { "type": "integer" },
					"name": { "type": "string" },
					"email": { "type": "string" }
				}
			}`,
			json:          `{"id": "not-an-integer"}`,
			expectedCount: 3, // Missing name and email, id is not an integer, and schema validation error
		},
		{
			name: "No validation errors",
			schema: `{
				"type": "object",
				"properties": {
					"id": { "type": "integer" },
					"name": { "type": "string" }
				}
			}`,
			json:          `{"id": 1, "name": "Test User"}`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate with errors
			_, errors := ValidateWithErrors(tt.json, tt.schema)

			// Check error count
			if len(errors) != tt.expectedCount {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedCount, len(errors), errors)
			}
		})
	}
}

func TestRegisterFormats(t *testing.T) {
	// Test cases for format validation
	tests := []struct {
		name          string
		schema        string
		json          string
		expectedValid bool
	}{
		{
			name: "Valid email format",
			schema: `{
				"type": "object",
				"properties": {
					"email": { "type": "string", "format": "email" }
				}
			}`,
			json:          `{"email": "user@example.com"}`,
			expectedValid: true,
		},
		{
			name: "Invalid email format",
			schema: `{
				"type": "object",
				"properties": {
					"email": { "type": "string", "format": "email" }
				},
				"required": ["email"]
			}`,
			json:          `{"email": "not-an-email"}`,
			expectedValid: false, // The jsonschema library validates email format
		},
		{
			name: "Valid date format",
			schema: `{
				"type": "object",
				"properties": {
					"date": { "type": "string", "format": "date" }
				}
			}`,
			json:          `{"date": "2023-01-01"}`,
			expectedValid: true,
		},
		{
			name: "Invalid date format",
			schema: `{
				"type": "object",
				"properties": {
					"date": { "type": "string", "format": "date" }
				},
				"required": ["date"]
			}`,
			json:          `{"date": "01/01/2023"}`,
			expectedValid: false,
		},
		{
			name: "Valid URI format",
			schema: `{
				"type": "object",
				"properties": {
					"uri": { "type": "string", "format": "uri" }
				}
			}`,
			json:          `{"uri": "https://example.com"}`,
			expectedValid: true,
		},
		{
			name: "Invalid URI format",
			schema: `{
				"type": "object",
				"properties": {
					"uri": { "type": "string", "format": "uri" }
				},
				"required": ["uri"]
			}`,
			json:          `{"uri": "not-a-uri"}`,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a compiler and register formats
			compiler := jsonschema.NewCompiler()
			registerFormats(compiler)

			// Add the schema to the compiler
			if err := compiler.AddResource("schema.json", strings.NewReader(tt.schema)); err != nil {
				t.Fatalf("Failed to add schema: %v", err)
			}

			// Compile the schema
			schema, err := compiler.Compile("schema.json")
			if err != nil {
				t.Fatalf("Failed to compile schema: %v", err)
			}

			// Parse the JSON
			var jsonData interface{}
			if err := json.Unmarshal([]byte(tt.json), &jsonData); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate
			err = schema.Validate(jsonData)
			valid := err == nil

			// Check validity
			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectedValid, valid)
			}
		})
	}
}
