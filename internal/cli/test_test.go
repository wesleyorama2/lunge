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

// TestRunTestWithContext tests the runTestWithContext function
func TestRunTestWithContext(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   1,
			"name": "Test User",
		})
	}))
	defer server.Close()

	// Create a test configuration
	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:     "/test",
				Method:  "GET",
				Headers: map[string]string{},
			},
		},
	}

	// Create a test environment
	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	// Create a test
	test := config.Test{
		Name:    "Test Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{
				"status": float64(200),
			},
		},
	}

	// Create a client
	client := lungehttp.NewClient(
		lungehttp.WithBaseURL(server.URL),
	)

	// Create a formatter
	formatter := output.NewFormatter(false, false)

	// Run the test
	results := runTestWithContext(
		context.Background(),
		1,
		test,
		cfg,
		env,
		map[string]string{},
		client,
		formatter,
		30*time.Second,
		false,
		false, // Disable output
	)

	// Check the results
	if !results.passed {
		t.Errorf("Test should have passed")
	}
	if results.totalAssertions != 1 {
		t.Errorf("Expected 1 assertion, got %d", results.totalAssertions)
	}
	if results.passedAssertions != 1 {
		t.Errorf("Expected 1 passed assertion, got %d", results.passedAssertions)
	}
	if results.failedAssertions != 0 {
		t.Errorf("Expected 0 failed assertions, got %d", results.failedAssertions)
	}

	// Test with a failing assertion
	test.Assertions = []map[string]interface{}{
		{
			"status": float64(404), // This will fail
		},
	}

	// Run the test
	results = runTestWithContext(
		context.Background(),
		1,
		test,
		cfg,
		env,
		map[string]string{},
		client,
		formatter,
		30*time.Second,
		false,
		false, // Disable output
	)

	// Check the results
	if results.passed {
		t.Errorf("Test should have failed")
	}
	if results.totalAssertions != 1 {
		t.Errorf("Expected 1 assertion, got %d", results.totalAssertions)
	}
	if results.passedAssertions != 0 {
		t.Errorf("Expected 0 passed assertions, got %d", results.passedAssertions)
	}
	if results.failedAssertions != 1 {
		t.Errorf("Expected 1 failed assertion, got %d", results.failedAssertions)
	}
}
