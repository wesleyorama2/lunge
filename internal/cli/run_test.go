package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wesleyorama2/lunge/internal/config"
	lungehttp "github.com/wesleyorama2/lunge/internal/http"
	"github.com/wesleyorama2/lunge/internal/output"
)

// mockHTTPServer creates a mock HTTP server for testing
func mockHTTPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"users": []map[string]interface{}{
					{
						"id":   1,
						"name": "Test User",
					},
				},
			})
			return
		}

		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestExecuteRequest(t *testing.T) {
	// Create a mock HTTP server
	server := mockHTTPServer()
	defer server.Close()

	// Create a test configuration
	cfg := &config.Config{
		Environments: map[string]config.Environment{
			"test": {
				BaseURL: server.URL,
				Vars:    map[string]string{},
			},
		},
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
		},
	}

	// Create a client
	client := lungehttp.NewClient(
		lungehttp.WithBaseURL(server.URL),
	)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute request
			env := cfg.Environments["test"]
			formatter := output.NewFormatter(false, false)
			timeout := 30 * time.Second

			// We can't easily capture the output or return value of executeRequest,
			// so we're just testing that it doesn't panic
			executeRequest(cfg, tt.requestName, env, tt.envVars, client, formatter, timeout, false)

			// Since executeRequest doesn't return anything, we can only check
			// that it doesn't panic for valid requests
			if tt.expectedError {
				// For requests that should error, we expect executeRequest to call os.Exit(1)
				// which we can't test directly, so we'll skip these tests
				t.Skip("Skipping test that would cause os.Exit(1)")
			}
		})
	}
}

func TestExecuteSuite(t *testing.T) {
	// Create a mock HTTP server
	server := mockHTTPServer()
	defer server.Close()

	// Create a test configuration
	cfg := &config.Config{
		Environments: map[string]config.Environment{
			"test": {
				BaseURL: server.URL,
				Vars:    map[string]string{},
			},
		},
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
		},
		Suites: map[string]config.Suite{
			"validSuite": {
				Requests: []string{"getUsers"},
				Vars:     map[string]string{},
			},
			"errorSuite": {
				Requests: []string{"getError"},
				Vars:     map[string]string{},
			},
			"mixedSuite": {
				Requests: []string{"getUsers", "getError"},
				Vars:     map[string]string{},
			},
			"nonExistentRequestSuite": {
				Requests: []string{"nonExistentRequest"},
				Vars:     map[string]string{},
			},
			"emptySuite": {
				Requests: []string{},
				Vars:     map[string]string{},
			},
		},
	}

	// Test cases
	tests := []struct {
		name          string
		suiteName     string
		expectedError bool
	}{
		{
			name:          "Valid suite",
			suiteName:     "validSuite",
			expectedError: false,
		},
		{
			name:          "Error suite",
			suiteName:     "errorSuite",
			expectedError: false, // Not an error from executeSuite's perspective
		},
		{
			name:          "Mixed suite",
			suiteName:     "mixedSuite",
			expectedError: false,
		},
		{
			name:          "Non-existent request suite",
			suiteName:     "nonExistentRequestSuite",
			expectedError: true,
		},
		{
			name:          "Empty suite",
			suiteName:     "emptySuite",
			expectedError: false,
		},
		{
			name:          "Non-existent suite",
			suiteName:     "nonExistentSuite",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute suite
			env := cfg.Environments["test"]
			envVars := map[string]string{}
			client := lungehttp.NewClient(
				lungehttp.WithBaseURL(server.URL),
			)
			formatter := output.NewFormatter(false, false)
			timeout := 30 * time.Second

			// We can't easily capture the output or return value of executeSuite,
			// so we're just testing that it doesn't panic
			if !tt.expectedError {
				executeSuite(cfg, tt.suiteName, env, envVars, client, formatter, timeout, false)
			} else {
				// For suites that should error, we expect executeSuite to call os.Exit(1)
				// which we can't test directly, so we'll skip these tests
				t.Skip("Skipping test that would cause os.Exit(1)")
			}

			// Since executeSuite doesn't return anything, we can only check
			// that it doesn't panic for valid suites
		})
	}
}
