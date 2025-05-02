package jsonschema

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		schema        string
		json          string
		expectedValid bool
		expectedError bool
	}{
		{
			name: "Valid simple object",
			schema: `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "integer" }
				},
				"required": ["name"]
			}`,
			json: `{
				"name": "John Doe",
				"age": 30
			}`,
			expectedValid: true,
			expectedError: false,
		},
		{
			name: "Invalid - missing required property",
			schema: `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "integer" }
				},
				"required": ["name"]
			}`,
			json: `{
				"age": 30
			}`,
			expectedValid: false,
			expectedError: false,
		},
		{
			name: "Invalid - wrong type",
			schema: `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "integer" }
				}
			}`,
			json: `{
				"name": "John Doe",
				"age": "thirty"
			}`,
			expectedValid: false,
			expectedError: false,
		},
		{
			name: "Valid array",
			schema: `{
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": { "type": "integer" },
						"name": { "type": "string" }
					},
					"required": ["id"]
				}
			}`,
			json: `[
				{ "id": 1, "name": "Item 1" },
				{ "id": 2, "name": "Item 2" }
			]`,
			expectedValid: true,
			expectedError: false,
		},
		{
			name: "Invalid array item",
			schema: `{
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": { "type": "integer" },
						"name": { "type": "string" }
					},
					"required": ["id"]
				}
			}`,
			json: `[
				{ "id": 1, "name": "Item 1" },
				{ "name": "Missing ID" }
			]`,
			expectedValid: false,
			expectedError: false,
		},
		{
			name: "Invalid schema",
			schema: `{
				"type": "invalid-type"
			}`,
			json:          `{}`,
			expectedValid: false,
			expectedError: true,
		},
		{
			name: "Invalid JSON",
			schema: `{
				"type": "object"
			}`,
			json:          `{ invalid json }`,
			expectedValid: false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := Validate(tt.json, tt.schema)

			// Check error
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			// If we expect an error, don't check validity
			if tt.expectedError {
				return
			}

			// Check validity
			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectedValid, valid)
			}
		})
	}
}

func TestValidateWithErrors(t *testing.T) {
	// Test cases that focus on validation errors
	tests := []struct {
		name           string
		schema         string
		json           string
		expectedErrors []string // Substrings that should be in the error message
	}{
		{
			name: "Missing required property",
			schema: `{
				"type": "object",
				"required": ["name"]
			}`,
			json:           `{}`,
			expectedErrors: []string{"name", "missing properties"},
		},
		{
			name: "Wrong type",
			schema: `{
				"type": "object",
				"properties": {
					"age": { "type": "integer" }
				}
			}`,
			json: `{
				"age": "thirty"
			}`,
			expectedErrors: []string{"age", "integer", "string"},
		},
		{
			name: "Multiple errors",
			schema: `{
				"type": "object",
				"properties": {
					"name": { "type": "string", "minLength": 3 },
					"age": { "type": "integer", "minimum": 18 }
				},
				"required": ["name", "age"]
			}`,
			json: `{
				"name": "Jo",
				"age": 16
			}`,
			expectedErrors: []string{"length must be >= 3", "must be >= 18"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := ValidateWithErrors(tt.json, tt.schema)

			// Check that we got errors
			if len(errors) == 0 {
				t.Errorf("Expected validation errors, got none")
				return
			}

			// Check that all expected error substrings are present
			errorStr := errors.Error()
			for _, expectedError := range tt.expectedErrors {
				if !strings.Contains(errorStr, expectedError) {
					t.Errorf("Expected error to contain %q, got %q", expectedError, errorStr)
				}
			}
		})
	}
}
