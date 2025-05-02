package jsonschema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ValidationErrors represents a collection of validation errors
type ValidationErrors []error

// Error implements the error interface for ValidationErrors
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, err := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// Validate validates a JSON string against a JSON Schema
// Returns true if the JSON is valid, false otherwise
// If there's an error in the schema or JSON parsing, it returns an error
func Validate(jsonStr, schemaStr string) (bool, error) {
	// Parse the schema
	compiler := jsonschema.NewCompiler()

	// Add the schema to the compiler
	if err := compiler.AddResource("schema.json", strings.NewReader(schemaStr)); err != nil {
		return false, fmt.Errorf("invalid schema: %w", err)
	}

	// Compile the schema
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return false, fmt.Errorf("invalid schema: %w", err)
	}

	// Parse the JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return false, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate the JSON against the schema
	err = schema.Validate(jsonData)
	if err != nil {
		// JSON is invalid according to the schema
		return false, nil
	}

	// JSON is valid
	return true, nil
}

// ValidateWithErrors validates a JSON string against a JSON Schema
// Returns true if the JSON is valid, false otherwise
// Also returns a list of validation errors if the JSON is invalid
func ValidateWithErrors(jsonStr, schemaStr string) (bool, ValidationErrors) {
	// Parse the schema
	compiler := jsonschema.NewCompiler()

	// Add the schema to the compiler
	if err := compiler.AddResource("schema.json", strings.NewReader(schemaStr)); err != nil {
		return false, ValidationErrors{fmt.Errorf("invalid schema: %w", err)}
	}

	// Compile the schema
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return false, ValidationErrors{fmt.Errorf("invalid schema: %w", err)}
	}

	// Parse the JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return false, ValidationErrors{fmt.Errorf("invalid JSON: %w", err)}
	}

	// Validate the JSON against the schema
	err = schema.Validate(jsonData)
	if err != nil {
		// JSON is invalid according to the schema
		if validationErr, ok := err.(*jsonschema.ValidationError); ok {
			// Extract all validation errors
			errors := extractValidationErrors(validationErr)
			return false, errors
		}
		return false, ValidationErrors{err}
	}

	// JSON is valid
	return true, nil
}

// extractValidationErrors extracts all validation errors from a jsonschema.ValidationError
func extractValidationErrors(err *jsonschema.ValidationError) ValidationErrors {
	var errors ValidationErrors

	// Add the current error
	if err.Message != "" {
		errors = append(errors, fmt.Errorf("validation error at %s: %s", err.InstanceLocation, err.Message))
	}

	// Add all child errors
	for _, childErr := range err.Causes {
		errors = append(errors, extractValidationErrors(childErr)...)
	}

	return errors
}
