package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Do(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "GET" {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check request path
		if r.URL.Path != "/test" {
			t.Errorf("Expected path /test, got %s", r.URL.Path)
		}

		// Check request headers
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Errorf("Expected header X-Test-Header: test-value, got %s", r.Header.Get("X-Test-Header"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	// Create client
	client := NewClient(
		WithTimeout(5*time.Second),
		WithHeader("User-Agent", "lunge-test"),
		WithBaseURL(server.URL),
	)

	// Create request
	req := NewRequest("GET", "/test")
	req.WithHeader("X-Test-Header", "test-value")

	// Execute request
	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Error executing request: %v", err)
	}

	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.GetHeader("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", resp.GetHeader("Content-Type"))
	}

	body, err := resp.GetBodyAsString()
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	expectedBody := `{"message":"success"}`
	if body != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, body)
	}
}

func TestClient_WithOptions(t *testing.T) {
	// Test client options
	timeout := 10 * time.Second
	baseURL := "https://example.com"
	headerKey := "X-Test"
	headerValue := "test-value"

	client := NewClient(
		WithTimeout(timeout),
		WithBaseURL(baseURL),
		WithHeader(headerKey, headerValue),
	)

	// Check timeout
	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.httpClient.Timeout)
	}

	// Check base URL
	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}

	// Check headers
	if client.headers[headerKey] != headerValue {
		t.Errorf("Expected header %s: %s, got %s", headerKey, headerValue, client.headers[headerKey])
	}
}
