package cli

import (
	"lunge/internal/config"
	"reflect"
	"testing"
)

// TestRequestBodyProcessing tests the processing of request bodies with variable substitution
func TestRequestBodyProcessing(t *testing.T) {
	// Setup environment variables
	envVars := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"userId":   "123",
		"city":     "TestCity",
	}

	// Test cases
	tests := []struct {
		name     string
		body     interface{}
		expected interface{}
	}{
		{
			name: "Map with string values",
			body: map[string]interface{}{
				"title":  "Post by {{username}}",
				"body":   "This is a post by {{username}} ({{email}}) from {{city}}",
				"userId": "{{userId}}",
			},
			expected: map[string]interface{}{
				"title":  "Post by testuser",
				"body":   "This is a post by testuser (test@example.com) from TestCity",
				"userId": "123",
			},
		},
		{
			name:     "String body",
			body:     "Hello {{username}}!",
			expected: "Hello testuser!",
		},
		{
			name: "Map with mixed value types",
			body: map[string]interface{}{
				"title":    "Post by {{username}}",
				"userId":   "{{userId}}",
				"verified": true,
				"count":    42,
			},
			expected: map[string]interface{}{
				"title":    "Post by testuser",
				"userId":   "123",
				"verified": true,
				"count":    42,
			},
		},
		{
			name:     "Non-string body",
			body:     123,
			expected: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}

			// This is the same logic used in the executeRequest function
			switch body := tt.body.(type) {
			case map[string]interface{}:
				// Process each field in the map
				processedBody := make(map[string]interface{})
				for k, v := range body {
					if strValue, ok := v.(string); ok {
						// Process string values for variable substitution
						processedBody[k] = config.ProcessEnvironment(strValue, envVars)
					} else {
						// Keep non-string values as is
						processedBody[k] = v
					}
				}
				result = processedBody
			case string:
				// Process string body
				result = config.ProcessEnvironment(body, envVars)
			default:
				// Use body as is for other types
				result = tt.body
			}

			// Check result
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
