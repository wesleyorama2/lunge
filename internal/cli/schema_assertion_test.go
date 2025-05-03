package cli

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wesleyorama2/lunge/internal/config"
	"github.com/wesleyorama2/lunge/internal/http"
)

func TestSchemaAssertion(t *testing.T) {
	// Create a test configuration with schemas
	validSchemaJSON := json.RawMessage(`{
		"type": "object",
		"required": ["id", "name"],
		"properties": {
			"id": { "type": "integer" },
			"name": { "type": "string" }
		}
	}`)

	invalidSchemaJSON := json.RawMessage(`{
		"type": "object",
		"required": ["nonExistentField"],
		"properties": {
			"nonExistentField": { "type": "string" }
		}
	}`)

	cfg := &config.Config{
		Schemas: map[string]json.RawMessage{
			"validSchema":   validSchemaJSON,
			"invalidSchema": invalidSchemaJSON,
		},
	}

	// Test cases
	tests := []struct {
		name            string
		assertion       map[string]interface{}
		responseBody    string
		responseStatus  int
		responseHeaders map[string][]string
		expectedResult  bool
	}{
		// Schema validation tests
		{
			name: "Valid schema validation",
			assertion: map[string]interface{}{
				"schema": "validSchema",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Invalid schema validation - missing required field",
			assertion: map[string]interface{}{
				"schema": "validSchema",
			},
			responseBody:    `{"id": 1}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Invalid schema validation - wrong type",
			assertion: map[string]interface{}{
				"schema": "validSchema",
			},
			responseBody:    `{"id": "not-an-integer", "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Schema that doesn't match response",
			assertion: map[string]interface{}{
				"schema": "invalidSchema",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Non-existent schema",
			assertion: map[string]interface{}{
				"schema": "nonExistentSchema",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},

		// Status code assertion tests
		{
			name: "Status code assertion - valid",
			assertion: map[string]interface{}{
				"status": float64(200),
			},
			responseBody:    `{}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Status code assertion - invalid",
			assertion: map[string]interface{}{
				"status": float64(201),
			},
			responseBody:    `{}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},

		// Header assertion tests
		{
			name: "Header exists assertion - valid",
			assertion: map[string]interface{}{
				"header": "Content-Type",
				"exists": true,
			},
			responseBody:   `{}`,
			responseStatus: 200,
			responseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedResult: true,
		},
		{
			name: "Header exists assertion - invalid",
			assertion: map[string]interface{}{
				"header": "X-Custom-Header",
				"exists": true,
			},
			responseBody:    `{}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Header equals assertion - valid",
			assertion: map[string]interface{}{
				"header": "Content-Type",
				"equals": "application/json",
			},
			responseBody:   `{}`,
			responseStatus: 200,
			responseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedResult: true,
		},
		{
			name: "Header equals assertion - invalid",
			assertion: map[string]interface{}{
				"header": "Content-Type",
				"equals": "text/plain",
			},
			responseBody:   `{}`,
			responseStatus: 200,
			responseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedResult: false,
		},

		// Path assertion tests
		{
			name: "Path exists assertion - valid",
			assertion: map[string]interface{}{
				"path":   "$.name",
				"exists": true,
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Path exists assertion - invalid",
			assertion: map[string]interface{}{
				"path":   "$.age",
				"exists": true,
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Path equals assertion - valid",
			assertion: map[string]interface{}{
				"path":   "$.name",
				"equals": "Test User",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Path equals assertion - invalid",
			assertion: map[string]interface{}{
				"path":   "$.name",
				"equals": "Wrong Name",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Path isArray assertion - valid",
			assertion: map[string]interface{}{
				"path":    "$.tags",
				"isArray": true,
			},
			responseBody:    `{"id": 1, "name": "Test User", "tags": ["tag1", "tag2"]}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Path isArray assertion - invalid",
			assertion: map[string]interface{}{
				"path":    "$.name",
				"isArray": true,
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Path minLength assertion - valid",
			assertion: map[string]interface{}{
				"path":      "$.tags",
				"minLength": float64(2),
			},
			responseBody:    `{"id": 1, "name": "Test User", "tags": ["tag1", "tag2"]}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Path minLength assertion - invalid",
			assertion: map[string]interface{}{
				"path":      "$.tags",
				"minLength": float64(3),
			},
			responseBody:    `{"id": 1, "name": "Test User", "tags": ["tag1", "tag2"]}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Path contains assertion - valid",
			assertion: map[string]interface{}{
				"path":     "$.name",
				"contains": "Test",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Path contains assertion - invalid",
			assertion: map[string]interface{}{
				"path":     "$.name",
				"contains": "Admin",
			},
			responseBody:    `{"id": 1, "name": "Test User"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Path matches assertion - valid",
			assertion: map[string]interface{}{
				"path":    "$.email",
				"matches": ".*@example\\.com",
			},
			responseBody:    `{"id": 1, "name": "Test User", "email": "test@example.com"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  true,
		},
		{
			name: "Path matches assertion - invalid",
			assertion: map[string]interface{}{
				"path":    "$.email",
				"matches": ".*@example\\.com",
			},
			responseBody:    `{"id": 1, "name": "Test User", "email": "test@gmail.com"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},

		// Unknown assertion test
		{
			name: "Unknown assertion",
			assertion: map[string]interface{}{
				"unknown": "value",
			},
			responseBody:    `{}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Path matches assertion - invalid regex",
			assertion: map[string]interface{}{
				"path":    "$.email",
				"matches": "[",
			},
			responseBody:    `{"id": 1, "name": "Test User", "email": "test@example.com"}`,
			responseStatus:  200,
			responseHeaders: map[string][]string{},
			expectedResult:  false,
		},
		{
			name: "Header contains assertion - valid",
			assertion: map[string]interface{}{
				"header":   "Content-Type",
				"contains": "json",
			},
			responseBody:   `{}`,
			responseStatus: 200,
			responseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedResult: true,
		},
		{
			name: "Header contains assertion - invalid",
			assertion: map[string]interface{}{
				"header":   "Content-Type",
				"contains": "xml",
			},
			responseBody:   `{}`,
			responseStatus: 200,
			responseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock response
			resp := &http.Response{
				StatusCode: tt.responseStatus,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				Headers:    tt.responseHeaders,
			}

			// For response time assertions, we need to use a fixed time
			var startTime time.Time
			if _, ok := tt.assertion["responseTime"]; ok {
				// Set startTime to 100ms ago to simulate a 100ms response time
				startTime = time.Now().Add(-100 * time.Millisecond)
			} else {
				startTime = time.Now()
			}

			result, _ := runAssertion(tt.assertion, resp, nil, startTime, cfg)

			// Check the result
			if result != tt.expectedResult {
				t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestGetSchemaFromConfig(t *testing.T) {
	// Create a test configuration with schemas
	testSchemaJSON := json.RawMessage(`{"type": "object"}`)

	cfg := &config.Config{
		Schemas: map[string]json.RawMessage{
			"testSchema": testSchemaJSON,
		},
	}

	// Test cases
	tests := []struct {
		name          string
		schemaName    string
		cfg           *config.Config
		expectedError bool
	}{
		{
			name:          "Existing schema",
			schemaName:    "testSchema",
			cfg:           cfg,
			expectedError: false,
		},
		{
			name:          "Non-existent schema",
			schemaName:    "nonExistentSchema",
			cfg:           cfg,
			expectedError: true,
		},
		{
			name:          "Nil schemas",
			schemaName:    "testSchema",
			cfg:           &config.Config{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get schema from config
			schema, err := getSchemaFromConfig(tt.schemaName, tt.cfg)

			// Check error
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			// Check schema
			if !tt.expectedError && schema == "" {
				t.Errorf("Expected schema, got empty string")
			}
		})
	}
}
