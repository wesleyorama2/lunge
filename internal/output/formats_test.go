package output

import (
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	nethttp "net/http"

	"github.com/wesleyorama2/lunge/internal/http"
)

// setupTestRequest creates a test request with headers, query params, and body
func setupTestRequest() (*http.Request, string) {
	req := http.NewRequest("POST", "/users")
	req.WithHeader("Content-Type", "application/json")
	req.WithHeader("Authorization", "Bearer token123")
	req.WithQueryParam("page", "1")
	req.WithQueryParam("limit", "10")
	req.WithBody(map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	})

	baseURL := "https://api.example.com"
	return req, baseURL
}

// setupTestResponse creates a test response with headers and body
func setupTestResponse() *http.Response {
	headers := make(nethttp.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Rate-Limit", "100")

	jsonBody := `{"id":1,"name":"John Doe","email":"john@example.com"}`
	resp := &http.Response{
		StatusCode:   200,
		Status:       "200 OK",
		Headers:      headers,
		Body:         io.NopCloser(strings.NewReader(jsonBody)),
		ResponseTime: 123 * time.Millisecond,
	}

	// Read the body to ensure it's parsed
	resp.GetBodyAsString()

	return resp
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		name     string
		format   OutputFormat
		verbose  bool
		noColor  bool
		expected string
	}{
		{
			name:     "Default text formatter",
			format:   FormatText,
			verbose:  false,
			noColor:  false,
			expected: "*output.Formatter",
		},
		{
			name:     "JSON formatter",
			format:   FormatJSON,
			verbose:  false,
			noColor:  false,
			expected: "*output.JSONFormatter",
		},
		{
			name:     "YAML formatter",
			format:   FormatYAML,
			verbose:  false,
			noColor:  false,
			expected: "*output.YAMLFormatter",
		},
		{
			name:     "JUnit formatter",
			format:   FormatJUnit,
			verbose:  false,
			noColor:  false,
			expected: "*output.JUnitFormatter",
		},
		{
			name:     "Unknown format defaults to text",
			format:   "unknown",
			verbose:  false,
			noColor:  false,
			expected: "*output.Formatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := GetFormatter(tt.format, tt.verbose, tt.noColor)
			// Get the type name using reflection
			typeName := reflect.TypeOf(formatter).String()
			if typeName != tt.expected {
				t.Errorf("GetFormatter() returned %s, expected %s", typeName, tt.expected)
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := &JSONFormatter{Verbose: true, Pretty: true}
	req, baseURL := setupTestRequest()
	resp := setupTestResponse()

	// Test request formatting
	reqOutput := formatter.FormatRequest(req, baseURL)
	var reqData map[string]interface{}
	if err := json.Unmarshal([]byte(reqOutput), &reqData); err != nil {
		t.Errorf("FormatRequest() did not return valid JSON: %v", err)
	}

	// Check required fields
	requiredFields := []string{"method", "url", "headers", "body", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := reqData[field]; !ok {
			t.Errorf("FormatRequest() JSON missing required field: %s", field)
		}
	}

	// Test response formatting
	respOutput := formatter.FormatResponse(resp)
	var respData map[string]interface{}
	if err := json.Unmarshal([]byte(respOutput), &respData); err != nil {
		t.Errorf("FormatResponse() did not return valid JSON: %v", err)
	}

	// Check required fields
	requiredRespFields := []string{"statusCode", "status", "headers", "body", "responseTimeMs", "timestamp"}
	for _, field := range requiredRespFields {
		if _, ok := respData[field]; !ok {
			t.Errorf("FormatResponse() JSON missing required field: %s", field)
		}
	}
}

func TestYAMLFormatter(t *testing.T) {
	formatter := &YAMLFormatter{Verbose: true}
	req, baseURL := setupTestRequest()
	resp := setupTestResponse()

	// Test request formatting
	reqOutput := formatter.FormatRequest(req, baseURL)
	if !strings.Contains(reqOutput, "method: POST") {
		t.Errorf("FormatRequest() YAML does not contain expected content")
	}

	// Test response formatting
	respOutput := formatter.FormatResponse(resp)
	if !strings.Contains(respOutput, "statusCode: 200") {
		t.Errorf("FormatResponse() YAML does not contain expected content")
	}
}

func TestJUnitFormatter(t *testing.T) {
	formatter := &JUnitFormatter{
		Verbose:   true,
		TestName:  "TestRequest",
		SuiteName: "TestSuite",
	}
	req, baseURL := setupTestRequest()
	resp := setupTestResponse()

	// Test request formatting
	reqOutput := formatter.FormatRequest(req, baseURL)
	if !strings.Contains(reqOutput, "<!-- Request:") {
		t.Errorf("FormatRequest() JUnit XML does not contain expected content")
	}

	// Test response formatting for first request
	respOutput1 := formatter.FormatResponse(resp)
	if !strings.Contains(respOutput1, "<testsuite") || !strings.Contains(respOutput1, "<testcase") {
		t.Errorf("FormatResponse() JUnit XML does not contain expected content")
	}

	// Verify test count is 1 for the first response
	if !strings.Contains(respOutput1, `tests="1"`) {
		t.Errorf("First response should have tests=\"1\", got: %s", respOutput1)
	}

	// Add a second test case
	formatter.TestName = "SecondTest"
	resp2 := setupTestResponse()

	// Test response formatting for second request
	respOutput2 := formatter.FormatResponse(resp2)

	// Verify test count is 2 for the second response
	if !strings.Contains(respOutput2, `tests="2"`) {
		t.Errorf("Second response should have tests=\"2\", got: %s", respOutput2)
	}

	// Verify both test cases are included
	if !strings.Contains(respOutput2, "TestRequest") || !strings.Contains(respOutput2, "SecondTest") {
		t.Errorf("Second response should contain both test names")
	}
}
