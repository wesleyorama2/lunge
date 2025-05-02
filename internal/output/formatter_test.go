package output

import (
	"io"
	"strings"
	"testing"
	"time"

	nethttp "net/http"

	"github.com/wesleyorama2/lunge/internal/http"
)

func TestFormatter_FormatRequest(t *testing.T) {
	// Create formatter
	formatter := NewFormatter(true, true) // verbose, no color

	// Create request
	req := http.NewRequest("GET", "/users")
	req.WithHeader("Accept", "application/json")
	req.WithHeader("Authorization", "Bearer token123")
	req.WithQueryParam("page", "1")
	req.WithQueryParam("limit", "10")

	// Format request
	baseURL := "https://api.example.com"
	output := formatter.FormatRequest(req, baseURL)

	// Check output
	expectedParts := []string{
		"REQUEST: GET https://api.example.com/users",
		"Headers:",
		"Accept: application/json",
		"Authorization: Bearer token123",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain '%s', but it doesn't", part)
		}
	}

	// Check query parameters
	if !strings.Contains(output, "?") ||
		!strings.Contains(output, "page=1") ||
		!strings.Contains(output, "limit=10") {
		t.Errorf("Expected output to contain query parameters, got: %s", output)
	}
}

func TestFormatter_FormatRequestWithBody(t *testing.T) {
	// Create formatter
	formatter := NewFormatter(true, true) // verbose, no color

	// Create request with body
	req := http.NewRequest("POST", "/users")
	req.WithHeader("Content-Type", "application/json")
	req.WithBody(map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	})

	// Format request
	baseURL := "https://api.example.com"
	output := formatter.FormatRequest(req, baseURL)

	// Check output
	expectedParts := []string{
		"REQUEST: POST https://api.example.com/users",
		"Headers:",
		"Content-Type: application/json",
		"Body:",
		"name",
		"John Doe",
		"email",
		"john@example.com",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain '%s', but it doesn't", part)
		}
	}
}

func TestFormatter_FormatResponse(t *testing.T) {
	// Create formatter
	formatter := NewFormatter(true, true) // verbose, no color

	// Create headers
	headers := make(nethttp.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Rate-Limit", "100")

	// Create a response with a body
	jsonBody := `{"id":1,"name":"John Doe","email":"john@example.com"}`
	resp := &http.Response{
		StatusCode:   200,
		Status:       "200 OK",
		Headers:      headers,
		Body:         io.NopCloser(strings.NewReader(jsonBody)),
		ResponseTime: 123 * time.Millisecond,
	}

	// Read the body to ensure it's parsed
	_, err := resp.GetBodyAsString()
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	// Format response
	output := formatter.FormatResponse(resp)

	// Check output
	expectedParts := []string{
		"RESPONSE: 200 OK (123ms)",
		"Headers:",
		"Content-Type: application/json",
		"X-Rate-Limit: 100",
		"Body:",
		"id",
		"name",
		"John Doe",
		"email",
		"john@example.com",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain '%s', but it doesn't", part)
		}
	}
}

func TestFormatter_FormatResponseWithDifferentStatus(t *testing.T) {
	// Create formatter
	formatter := NewFormatter(false, true) // not verbose, no color

	// Test different status codes
	statusTests := []struct {
		statusCode int
		status     string
	}{
		{200, "200 OK"},
		{201, "201 Created"},
		{301, "301 Moved Permanently"},
		{400, "400 Bad Request"},
		{404, "404 Not Found"},
		{500, "500 Internal Server Error"},
	}

	for _, tt := range statusTests {
		t.Run(tt.status, func(t *testing.T) {
			// Create response
			resp := &http.Response{
				StatusCode:   tt.statusCode,
				Status:       tt.status,
				Headers:      make(nethttp.Header),
				Body:         nethttp.NoBody,
				ResponseTime: 100 * time.Millisecond,
			}

			// Format response
			output := formatter.FormatResponse(resp)

			// Check output
			expectedStatus := "RESPONSE: " + tt.status
			if !strings.Contains(output, expectedStatus) {
				t.Errorf("Expected output to contain '%s', but it doesn't", expectedStatus)
			}
		})
	}
}
