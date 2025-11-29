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

// TestRunTestWithContext_JSONFormatter tests runTestWithContext with JSON format
func TestRunTestWithContext_JSONFormatter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": 1, "name": "User 1"},
				{"id": 2, "name": "User 2"},
			},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:    "JSON Format Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{"status": float64(200)},
		},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatterWithFormat(output.FormatJSON, false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_YAMLFormatter tests runTestWithContext with YAML format
func TestRunTestWithContext_YAMLFormatter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:    "YAML Format Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{"status": float64(200)},
		},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatterWithFormat(output.FormatYAML, false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_JUnitFormatter tests runTestWithContext with JUnit format
func TestRunTestWithContext_JUnitFormatter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"result": "success"})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:    "JUnit Format Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{"status": float64(200)},
		},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := &output.JUnitFormatter{
		Verbose:   false,
		SuiteName: "TestSuite",
	}

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_WithOutput tests runTestWithContext with output enabled
func TestRunTestWithContext_WithOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"test": "data"})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:    "Output Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{"status": float64(200)},
		},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

	// Run with output enabled to cover print paths
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
		true, // Enable output
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_FailedWithOutput tests runTestWithContext with output enabled for failing test
func TestRunTestWithContext_FailedWithOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"test": "data"})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:    "Failed Output Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{"status": float64(404)}, // Will fail
		},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

	// Run with output enabled to cover print paths for failed test
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
		true, // Enable output
	)

	if results.passed {
		t.Errorf("Test should have failed")
	}
}

// TestRunTestWithContext_MultipleAssertions tests multiple assertions
func TestRunTestWithContext_MultipleAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": 1, "name": "User 1"},
			},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:    "Multiple Assertions Test",
		Request: "testRequest",
		Assertions: []map[string]interface{}{
			{"status": float64(200)},
			{"header": "Content-Type", "equals": "application/json"},
			{"path": "$.users", "isArray": true},
		},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
	if results.totalAssertions != 3 {
		t.Errorf("Expected 3 assertions, got %d", results.totalAssertions)
	}
}

// TestRunAssertion_ResponseTime tests response time assertions
func TestRunAssertion_ResponseTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

	tests := []struct {
		name       string
		assertion  map[string]interface{}
		shouldPass bool
	}{
		{
			name:       "Response time less than",
			assertion:  map[string]interface{}{"responseTime": "<5000"},
			shouldPass: true,
		},
		{
			name:       "Response time exact match - high value",
			assertion:  map[string]interface{}{"responseTime": "5000"},
			shouldPass: false, // exact match won't match 0ms response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := config.Test{
				Name:       tt.name,
				Request:    "testRequest",
				Assertions: []map[string]interface{}{tt.assertion},
			}

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
				false,
			)

			if results.passed != tt.shouldPass {
				t.Errorf("Expected passed=%v, got passed=%v", tt.shouldPass, results.passed)
			}
		})
	}
}

// TestRunAssertion_HeaderAssertions tests header assertions
func TestRunAssertion_HeaderAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

	tests := []struct {
		name       string
		assertion  map[string]interface{}
		shouldPass bool
	}{
		{
			name:       "Header exists - true",
			assertion:  map[string]interface{}{"header": "Content-Type", "exists": true},
			shouldPass: true,
		},
		{
			name:       "Header exists - false",
			assertion:  map[string]interface{}{"header": "X-Non-Existent", "exists": false},
			shouldPass: true,
		},
		{
			name:       "Header contains",
			assertion:  map[string]interface{}{"header": "Content-Type", "contains": "application/json"},
			shouldPass: true,
		},
		{
			name:       "Header matches",
			assertion:  map[string]interface{}{"header": "Content-Type", "matches": "application/.*"},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := config.Test{
				Name:       tt.name,
				Request:    "testRequest",
				Assertions: []map[string]interface{}{tt.assertion},
			}

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
				false,
			)

			if results.passed != tt.shouldPass {
				t.Errorf("Expected passed=%v, got passed=%v", tt.shouldPass, results.passed)
			}
		})
	}
}

// TestRunAssertion_PathAssertions tests JSON path assertions
func TestRunAssertion_PathAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"id":   123,
				"name": "test-item",
			},
			"items": []string{"item1", "item2", "item3"},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

	tests := []struct {
		name       string
		assertion  map[string]interface{}
		shouldPass bool
	}{
		{
			name:       "Path exists - true",
			assertion:  map[string]interface{}{"path": "$.status", "exists": true},
			shouldPass: true,
		},
		{
			name:       "Path exists - false",
			assertion:  map[string]interface{}{"path": "$.nonexistent", "exists": false},
			shouldPass: true,
		},
		{
			name:       "Path equals",
			assertion:  map[string]interface{}{"path": "$.status", "equals": "success"},
			shouldPass: true,
		},
		{
			name:       "Path is array",
			assertion:  map[string]interface{}{"path": "$.items", "isArray": true},
			shouldPass: true,
		},
		{
			name:       "Path min length",
			assertion:  map[string]interface{}{"path": "$.items", "minLength": 2},
			shouldPass: true,
		},
		{
			name:       "Path contains",
			assertion:  map[string]interface{}{"path": "$.status", "contains": "succ"},
			shouldPass: true,
		},
		{
			name:       "Path matches",
			assertion:  map[string]interface{}{"path": "$.status", "matches": "^success$"},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := config.Test{
				Name:       tt.name,
				Request:    "testRequest",
				Assertions: []map[string]interface{}{tt.assertion},
			}

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
				false,
			)

			if results.passed != tt.shouldPass {
				t.Errorf("Expected passed=%v, got passed=%v", tt.shouldPass, results.passed)
			}
		})
	}
}

// TestRunTestWithContext_WithAbsoluteURL tests with absolute URL in request
func TestRunTestWithContext_WithAbsoluteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    server.URL + "/test",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: "https://unused.example.com",
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:       "Absolute URL Test",
		Request:    "testRequest",
		Assertions: []map[string]interface{}{{"status": float64(200)}},
	}

	client := lungehttp.NewClient()
	formatter := output.NewFormatter(false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_WithQueryParams tests with query parameters
func TestRunTestWithContext_WithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query param was sent
		if r.URL.Query().Get("key") != "value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:         "/test",
				Method:      "GET",
				QueryParams: map[string]string{"key": "value"},
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:       "Query Params Test",
		Request:    "testRequest",
		Assertions: []map[string]interface{}{{"status": float64(200)}},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_WithBody tests with request body
func TestRunTestWithContext_WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "/test",
				Method: "POST",
				Body: map[string]interface{}{
					"name":  "test",
					"value": 123,
				},
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:       "Body Test",
		Request:    "testRequest",
		Assertions: []map[string]interface{}{{"status": float64(200)}},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}

// TestRunTestWithContext_WithEmptyURL tests with empty URL (uses baseURL)
func TestRunTestWithContext_WithEmptyURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Requests: map[string]config.Request{
			"testRequest": {
				URL:    "",
				Method: "GET",
			},
		},
	}

	env := config.Environment{
		BaseURL: server.URL,
		Vars:    map[string]string{},
	}

	test := config.Test{
		Name:       "Empty URL Test",
		Request:    "testRequest",
		Assertions: []map[string]interface{}{{"status": float64(200)}},
	}

	client := lungehttp.NewClient(lungehttp.WithBaseURL(server.URL))
	formatter := output.NewFormatter(false, false)

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
		false,
	)

	if !results.passed {
		t.Errorf("Test should have passed")
	}
}
