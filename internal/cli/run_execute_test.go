package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wesleyorama2/lunge/internal/config"
	lungehttp "github.com/wesleyorama2/lunge/internal/http"
	"github.com/wesleyorama2/lunge/internal/output"
)

// TestExecuteRequestWithContext tests the executeRequestWithContext function
func TestExecuteRequestWithContext(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   1,
				"name": "Test User",
			})
			return
		}

		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if r.URL.Path == "/extract" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"token": "test-token",
				"user": map[string]interface{}{
					"id":   1,
					"name": "Test User",
				},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create a test configuration
	cfg := &config.Config{
		Requests: map[string]config.Request{
			"getUsers": {
				URL:     "/users",
				Method:  "GET",
				Headers: map[string]string{},
			},
			"getError": {
				URL:     "/error",
				Method:  "GET",
				Headers: map[string]string{},
			},
			"getNotFound": {
				URL:     "/not-found",
				Method:  "GET",
				Headers: map[string]string{},
			},
			"extractToken": {
				URL:     "/extract",
				Method:  "GET",
				Headers: map[string]string{},
				Extract: map[string]string{
					"token": "$.token",
				},
			},
			"withValidation": {
				URL:     "/users",
				Method:  "GET",
				Headers: map[string]string{},
				Validate: map[string]interface{}{
					"type":     "object",
					"required": []string{"id", "name"},
					"properties": map[string]interface{}{
						"id":   map[string]interface{}{"type": "number"},
						"name": map[string]interface{}{"type": "string"},
					},
				},
			},
			"withBody": {
				URL:    "/users",
				Method: "POST",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]interface{}{
					"name": "New User",
				},
			},
			"withStringBody": {
				URL:    "/users",
				Method: "POST",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"name": "New User"}`,
			},
			"withQueryParams": {
				URL:     "/users",
				Method:  "GET",
				Headers: map[string]string{},
				QueryParams: map[string]string{
					"id": "1",
				},
			},
		},
	}

	// Create a test environment
	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	// Create a client
	client := lungehttp.NewClient(
		lungehttp.WithBaseURL(server.URL),
	)

	// Create a formatter
	formatter := output.NewFormatter(false, false)

	// Test cases
	tests := []struct {
		name          string
		requestName   string
		envVars       map[string]string
		expectedError bool
	}{
		{
			name:          "Valid request",
			requestName:   "getUsers",
			envVars:       map[string]string{},
			expectedError: false,
		},
		{
			name:          "Error response",
			requestName:   "getError",
			envVars:       map[string]string{},
			expectedError: false, // Not an error from executeRequest's perspective
		},
		{
			name:          "Not found response",
			requestName:   "getNotFound",
			envVars:       map[string]string{},
			expectedError: false, // Not an error from executeRequest's perspective
		},
		{
			name:          "Non-existent request",
			requestName:   "nonExistentRequest",
			envVars:       map[string]string{},
			expectedError: true,
		},
		{
			name:          "Extract token",
			requestName:   "extractToken",
			envVars:       map[string]string{},
			expectedError: false,
		},
		{
			name:          "With validation",
			requestName:   "withValidation",
			envVars:       map[string]string{},
			expectedError: false,
		},
		{
			name:          "With body",
			requestName:   "withBody",
			envVars:       map[string]string{},
			expectedError: false,
		},
		{
			name:          "With string body",
			requestName:   "withStringBody",
			envVars:       map[string]string{},
			expectedError: false,
		},
		{
			name:          "With query params",
			requestName:   "withQueryParams",
			envVars:       map[string]string{},
			expectedError: false,
		},
		{
			name:          "With environment variables",
			requestName:   "getUsers",
			envVars:       map[string]string{"baseUrl": server.URL},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute request
			ctx := context.Background()
			err := executeRequestWithContext(
				ctx,
				cfg,
				tt.requestName,
				env,
				tt.envVars,
				client,
				formatter,
				30*time.Second,
				false,
				false, // Disable output
			)

			// Check error
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}
