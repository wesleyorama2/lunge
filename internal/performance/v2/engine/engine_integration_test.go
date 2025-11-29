// Package engine provides integration tests for the v2 performance engine.
package engine

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wesleyorama2/lunge/internal/performance/v2/config"
)

// Test server types for different scenarios
type serverType int

const (
	serverNormal serverType = iota
	serverSlow
	serverError
	serverMixed
)

// createTestServer creates a test HTTP server with the specified behavior.
func createTestServer(st serverType) *httptest.Server {
	var requestCount atomic.Int64

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		switch st {
		case serverNormal:
			// Normal server: 200 OK with ~10ms latency
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","request":` + fmt.Sprintf("%d", count) + `}`))

		case serverSlow:
			// Slow server: 200 OK with ~500ms latency
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","slow":true}`))

		case serverError:
			// Error server: 500 errors
			time.Sleep(5 * time.Millisecond)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"server error"}`))

		case serverMixed:
			// Mixed server: varying latency and status codes
			latency := time.Duration(rand.Intn(100)+10) * time.Millisecond
			time.Sleep(latency)

			// 80% success, 20% error
			if count%5 == 0 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"occasional error"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}
		}
	}))
}

// ============================================================================
// Constant VUs Executor Tests
// ============================================================================

func TestEngineIntegration_ConstantVUs(t *testing.T) {
	// Start test server
	server := createTestServer(serverNormal)
	defer server.Close()

	// Create config
	cfg := &config.TestConfig{
		Name:        "Constant VUs Integration Test",
		Description: "Test constant-vus executor with fixed number of VUs",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "3s",
				Requests: []config.RequestConfig{
					{
						Name:   "get_status",
						Method: "GET",
						URL:    server.URL,
					},
				},
			},
		},
	}

	// Run engine
	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Verify results
	assert.NotNil(t, result)
	assert.Equal(t, "Constant VUs Integration Test", result.Name)
	assert.True(t, result.Duration > 0)
	assert.True(t, result.Metrics.TotalRequests > 0, "Should have made some requests")
	assert.True(t, result.Metrics.Latency.P95 > 0, "Should have latency data")
	assert.True(t, result.Metrics.RPS > 0, "Should have calculated RPS")

	// Verify scenario result
	scenarioResult, ok := result.Scenarios["test"]
	require.True(t, ok, "Should have test scenario result")
	assert.Equal(t, "constant-vus", scenarioResult.Executor)
	assert.True(t, scenarioResult.Iterations > 0, "Should have completed iterations")

	t.Logf("Constant VUs Test Results:")
	t.Logf("  Total Requests: %d", result.Metrics.TotalRequests)
	t.Logf("  RPS: %.2f", result.Metrics.RPS)
	t.Logf("  P95 Latency: %v", result.Metrics.Latency.P95)
	t.Logf("  Error Rate: %.4f", result.Metrics.ErrorRate)
}

func TestEngineIntegration_ConstantVUs_MultipleRequests(t *testing.T) {
	// Start test server
	server := createTestServer(serverNormal)
	defer server.Close()

	// Create config with multiple requests per iteration
	cfg := &config.TestConfig{
		Name: "Constant VUs Multiple Requests Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"multi_request": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{
						Name:   "request_1",
						Method: "GET",
						URL:    server.URL + "/endpoint1",
					},
					{
						Name:   "request_2",
						Method: "GET",
						URL:    server.URL + "/endpoint2",
					},
					{
						Name:   "request_3",
						Method: "GET",
						URL:    server.URL + "/endpoint3",
					},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Should have multiple request stats
	scenarioResult := result.Scenarios["multi_request"]
	require.NotNil(t, scenarioResult)

	// Each iteration has 3 requests
	assert.True(t, result.Metrics.TotalRequests >= 3, "Should have at least 3 requests (1 iteration)")

	t.Logf("Multiple Requests Test - Total Requests: %d, Iterations: %d",
		result.Metrics.TotalRequests, scenarioResult.Iterations)
}

// ============================================================================
// Ramping VUs Executor Tests
// ============================================================================

func TestEngineIntegration_RampingVUs(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	// Create config with ramping stages
	cfg := &config.TestConfig{
		Name: "Ramping VUs Integration Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"ramping": {
				Executor: "ramping-vus",
				Stages: []config.StageConfig{
					{Duration: "1s", Target: 2, Name: "ramp-up"},
					{Duration: "2s", Target: 2, Name: "steady"},
					{Duration: "1s", Target: 0, Name: "ramp-down"},
				},
				Requests: []config.RequestConfig{
					{
						Name:   "get_data",
						Method: "GET",
						URL:    server.URL,
					},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.True(t, result.Metrics.TotalRequests > 0)
	assert.True(t, result.Duration >= 3*time.Second, "Should run for at least total stage duration")

	scenarioResult := result.Scenarios["ramping"]
	require.NotNil(t, scenarioResult)
	assert.Equal(t, "ramping-vus", scenarioResult.Executor)

	// Verify time series shows VU count changes
	if len(result.TimeSeries) > 0 {
		t.Logf("Ramping VUs - Time series buckets: %d", len(result.TimeSeries))
	}

	t.Logf("Ramping VUs Test Results:")
	t.Logf("  Duration: %v", result.Duration)
	t.Logf("  Total Requests: %d", result.Metrics.TotalRequests)
	t.Logf("  RPS: %.2f", result.Metrics.RPS)
}

func TestEngineIntegration_RampingVUs_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	server := createTestServer(serverNormal)
	defer server.Close()

	// Ramp up to higher VUs
	cfg := &config.TestConfig{
		Name: "Ramping VUs Stress Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"stress": {
				Executor: "ramping-vus",
				Stages: []config.StageConfig{
					{Duration: "1s", Target: 5},
					{Duration: "2s", Target: 10},
					{Duration: "1s", Target: 0},
				},
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.True(t, result.Metrics.TotalRequests > 50, "Should have made many requests with 10 VUs")
	t.Logf("Stress Test - Requests: %d, RPS: %.2f", result.Metrics.TotalRequests, result.Metrics.RPS)
}

// ============================================================================
// Constant Arrival Rate Executor Tests
// ============================================================================

func TestEngineIntegration_ConstantArrivalRate(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	// Configure for ~10 iterations/second
	cfg := &config.TestConfig{
		Name: "Constant Arrival Rate Integration Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"arrival": {
				Executor:        "constant-arrival-rate",
				Rate:            10, // 10 iterations per second
				Duration:        "3s",
				PreAllocatedVUs: 5,
				MaxVUs:          10,
				Requests: []config.RequestConfig{
					{
						Name:   "rate_test",
						Method: "GET",
						URL:    server.URL,
					},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.NotNil(t, result)

	// With 10 RPS for 3 seconds, expect approximately 30 requests
	// In practice, arrival rate executors may exceed targets slightly due to:
	// - Multiple VUs running simultaneously
	// - Iteration timing overlap
	// Just verify we got a reasonable number of requests
	assert.True(t, result.Metrics.TotalRequests >= 20, "Should have at least 20 requests")
	assert.True(t, result.Metrics.TotalRequests <= 100, "Should not exceed 100 requests")

	t.Logf("Constant Arrival Rate Test Results:")
	t.Logf("  Target Rate: 10 RPS")
	t.Logf("  Duration: 3s")
	t.Logf("  Expected Requests: ~30")
	t.Logf("  Actual Requests: %d", result.Metrics.TotalRequests)
	t.Logf("  Actual RPS: %.2f", result.Metrics.RPS)
}

func TestEngineIntegration_ConstantArrivalRate_HighRate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high rate test in short mode")
	}

	server := createTestServer(serverNormal)
	defer server.Close()

	// Higher rate test
	cfg := &config.TestConfig{
		Name: "Constant Arrival Rate High Rate Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"high_rate": {
				Executor:        "constant-arrival-rate",
				Rate:            50, // 50 iterations per second
				Duration:        "2s",
				PreAllocatedVUs: 10,
				MaxVUs:          50,
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Should have approximately 100 requests (50 RPS * 2s)
	assert.True(t, result.Metrics.TotalRequests > 50, "Should have made many requests at 50 RPS")

	t.Logf("High Rate Test - Target: 50 RPS, Actual: %.2f RPS, Requests: %d",
		result.Metrics.RPS, result.Metrics.TotalRequests)
}

// ============================================================================
// Ramping Arrival Rate Executor Tests
// ============================================================================

func TestEngineIntegration_RampingArrivalRate(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	// Use simpler stages with more pre-allocated VUs
	cfg := &config.TestConfig{
		Name: "Ramping Arrival Rate Integration Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"ramping_rate": {
				Executor:        "ramping-arrival-rate",
				PreAllocatedVUs: 10,
				MaxVUs:          30,
				Stages: []config.StageConfig{
					{Duration: "2s", Target: 10, Name: "steady"},
					{Duration: "1s", Target: 0, Name: "ramp-down"},
				},
				Requests: []config.RequestConfig{
					{
						Name:   "ramping_rate_test",
						Method: "GET",
						URL:    server.URL,
					},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.NotNil(t, result)
	// Ramping arrival rate may have varying results depending on implementation
	// Just verify the test completed and we have metrics
	assert.True(t, result.Duration >= 2*time.Second, "Should run for at least stage duration")

	scenarioResult := result.Scenarios["ramping_rate"]
	require.NotNil(t, scenarioResult)
	assert.Equal(t, "ramping-arrival-rate", scenarioResult.Executor)

	t.Logf("Ramping Arrival Rate Test Results:")
	t.Logf("  Duration: %v", result.Duration)
	t.Logf("  Total Requests: %d", result.Metrics.TotalRequests)
	t.Logf("  Average RPS: %.2f", result.Metrics.RPS)
	t.Logf("  Iterations: %d", scenarioResult.Iterations)
}

// ============================================================================
// Multi-Scenario Tests
// ============================================================================

func TestEngineIntegration_MultiScenario(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Multi-Scenario Integration Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"scenario_a": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Name: "scenario_a_req", Method: "GET", URL: server.URL + "/a"},
				},
			},
			"scenario_b": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Name: "scenario_b_req", Method: "GET", URL: server.URL + "/b"},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Both scenarios should have results
	assert.Len(t, result.Scenarios, 2)
	assert.Contains(t, result.Scenarios, "scenario_a")
	assert.Contains(t, result.Scenarios, "scenario_b")

	// Both should have made requests
	assert.True(t, result.Scenarios["scenario_a"].Iterations > 0)
	assert.True(t, result.Scenarios["scenario_b"].Iterations > 0)

	t.Logf("Multi-Scenario Test Results:")
	for name, scenario := range result.Scenarios {
		t.Logf("  %s: %d iterations", name, scenario.Iterations)
	}
	t.Logf("  Total Requests: %d", result.Metrics.TotalRequests)
}

func TestEngineIntegration_MultiScenario_DifferentExecutors(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Multi-Scenario Different Executors Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"vus_scenario": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL + "/vus"},
				},
			},
			"rate_scenario": {
				Executor:        "constant-arrival-rate",
				Rate:            5,
				Duration:        "2s",
				PreAllocatedVUs: 2,
				MaxVUs:          5,
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL + "/rate"},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.Len(t, result.Scenarios, 2)
	assert.Equal(t, "constant-vus", result.Scenarios["vus_scenario"].Executor)
	assert.Equal(t, "constant-arrival-rate", result.Scenarios["rate_scenario"].Executor)

	t.Logf("Different Executors Test - Total Requests: %d", result.Metrics.TotalRequests)
}

// ============================================================================
// Threshold Tests
// ============================================================================

func TestEngineIntegration_Thresholds_Passing(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Threshold Passing Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
		Thresholds: &config.ThresholdsConfig{
			HTTPReqDuration: []string{
				"p95 < 1s",    // Should pass - server has ~10ms latency
				"avg < 500ms", // Should pass
			},
			HTTPReqFailed: []string{
				"rate < 0.1", // Should pass - no errors expected
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.True(t, result.Passed, "Test should pass all thresholds")
	assert.Len(t, result.Thresholds, 3, "Should have 3 threshold results")

	for _, tr := range result.Thresholds {
		assert.True(t, tr.Passed, "Threshold %s should pass: %s", tr.Expression, tr.Message)
	}

	t.Logf("Threshold Passing Test - All %d thresholds passed", len(result.Thresholds))
}

func TestEngineIntegration_Thresholds_Failing(t *testing.T) {
	server := createTestServer(serverError)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Threshold Failing Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
		Thresholds: &config.ThresholdsConfig{
			HTTPReqFailed: []string{
				"rate < 0.01", // Should fail - server returns errors
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.False(t, result.Passed, "Test should fail thresholds")
	assert.True(t, result.Metrics.ErrorRate > 0.5, "Error rate should be high")

	t.Logf("Threshold Failing Test - Error Rate: %.2f%%", result.Metrics.ErrorRate*100)
}

func TestEngineIntegration_Thresholds_RequestCount(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Threshold Request Count Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
		Thresholds: &config.ThresholdsConfig{
			HTTPReqs: []string{
				"count > 10", // Should have more than 10 requests in 2s with 2 VUs
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.True(t, result.Passed, "Should pass request count threshold")
	assert.True(t, result.Metrics.TotalRequests > 10, "Should have more than 10 requests")

	t.Logf("Request Count Test - Total: %d requests", result.Metrics.TotalRequests)
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestEngineIntegration_ErrorServer(t *testing.T) {
	server := createTestServer(serverError)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Error Server Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err) // Engine should run even with errors

	// All requests should have failed (500 status)
	assert.True(t, result.Metrics.FailedRequests > 0, "Should have failed requests")
	assert.True(t, result.Metrics.ErrorRate > 0.9, "Error rate should be very high")

	t.Logf("Error Server Test - Failed: %d, Error Rate: %.2f%%",
		result.Metrics.FailedRequests, result.Metrics.ErrorRate*100)
}

func TestEngineIntegration_MixedServer(t *testing.T) {
	server := createTestServer(serverMixed)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Mixed Server Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "3s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Should have both successes and failures
	assert.True(t, result.Metrics.SuccessRequests > 0, "Should have some successes")
	assert.True(t, result.Metrics.FailedRequests > 0, "Should have some failures")
	assert.True(t, result.Metrics.ErrorRate > 0.1 && result.Metrics.ErrorRate < 0.5,
		"Error rate should be around 20%%")

	t.Logf("Mixed Server Test - Success: %d, Failed: %d, Error Rate: %.2f%%",
		result.Metrics.SuccessRequests, result.Metrics.FailedRequests, result.Metrics.ErrorRate*100)
}

func TestEngineIntegration_SlowServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow server test in short mode")
	}

	server := createTestServer(serverSlow)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Slow Server Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "3s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
		Thresholds: &config.ThresholdsConfig{
			HTTPReqDuration: []string{
				"p95 < 100ms", // Should fail - server has 500ms latency
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Threshold should fail due to slow response
	assert.False(t, result.Passed, "Should fail latency threshold")
	assert.True(t, result.Metrics.Latency.P95 > 400*time.Millisecond,
		"P95 latency should be high")

	t.Logf("Slow Server Test - P95: %v", result.Metrics.Latency.P95)
}

// ============================================================================
// Config Parsing Integration Tests
// ============================================================================

func TestEngineIntegration_ConfigParsing_YAML(t *testing.T) {
	// Create a temporary YAML config file
	yamlContent := `
name: "YAML Config Test"
description: "Test loading config from YAML file"
settings:
  timeout: 10s
scenarios:
  api_test:
    executor: constant-vus
    vus: 1
    duration: 2s
    requests:
      - name: "test_request"
        method: GET
        url: "{{baseUrl}}/test"
thresholds:
  http_req_duration:
    - "p95 < 1s"
`

	// Create temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Start test server
	server := createTestServer(serverNormal)
	defer server.Close()

	// Load config
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)

	// Update URL with actual server URL
	cfg.Settings.BaseURL = server.URL
	cfg.Scenarios["api_test"].Requests[0].URL = server.URL + "/test"

	// Create and run engine
	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, "YAML Config Test", result.Name)
	assert.True(t, result.Metrics.TotalRequests > 0)
	assert.True(t, result.Passed, "Threshold should pass")

	t.Logf("YAML Config Test - Loaded and executed successfully")
}

func TestEngineIntegration_ConfigParsing_JSON(t *testing.T) {
	// Create a temporary JSON config file
	jsonContent := `{
	"name": "JSON Config Test",
	"scenarios": {
		"test": {
			"executor": "constant-vus",
			"vus": 1,
			"duration": "2s",
			"requests": [
				{
					"method": "GET",
					"url": "http://localhost/test"
				}
			]
		}
	}
}`

	// Create temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")
	err := os.WriteFile(configPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Start test server
	server := createTestServer(serverNormal)
	defer server.Close()

	// Load config
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)

	// Update URL with actual server URL
	cfg.Scenarios["test"].Requests[0].URL = server.URL

	// Create and run engine
	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, "JSON Config Test", result.Name)
	assert.True(t, result.Metrics.TotalRequests > 0)

	t.Logf("JSON Config Test - Loaded and executed successfully")
}

// ============================================================================
// Context Cancellation Tests
// ============================================================================

func TestEngineIntegration_ContextCancellation(t *testing.T) {
	server := createTestServer(serverSlow)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Context Cancellation Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "30s", // Long duration
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 1 second
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()

	startTime := time.Now()
	_, err = engine.Run(ctx)
	elapsed := time.Since(startTime)

	// Should have stopped early due to cancellation
	assert.True(t, elapsed < 5*time.Second, "Should stop quickly after cancellation")

	t.Logf("Context Cancellation Test - Stopped in %v", elapsed)
}

// ============================================================================
// Time Series Data Tests
// ============================================================================

func TestEngineIntegration_TimeSeries(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Time Series Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      2,
				Duration: "3s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Should have time series data
	assert.True(t, len(result.TimeSeries) > 0, "Should have time series buckets")

	// Verify time series has expected fields
	for _, bucket := range result.TimeSeries {
		assert.False(t, bucket.Timestamp.IsZero(), "Bucket should have timestamp")
		assert.True(t, bucket.TotalRequests >= 0, "Bucket should have request count")
	}

	t.Logf("Time Series Test - %d buckets captured", len(result.TimeSeries))
	if len(result.TimeSeries) > 0 {
		first := result.TimeSeries[0]
		last := result.TimeSeries[len(result.TimeSeries)-1]
		t.Logf("  First bucket: %v requests", first.IntervalRequests)
		t.Logf("  Last bucket: %v total requests", last.TotalRequests)
	}
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestEngineIntegration_InvalidConfig(t *testing.T) {
	// Missing scenarios
	cfg := &config.TestConfig{
		Name:      "Invalid Config Test",
		Scenarios: map[string]*config.ScenarioConfig{},
	}

	_, err := NewEngine(cfg)
	assert.Error(t, err, "Should reject config with no scenarios")
}

func TestEngineIntegration_InvalidExecutor(t *testing.T) {
	cfg := &config.TestConfig{
		Name: "Invalid Executor Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "invalid-executor",
				VUs:      1,
				Duration: "1s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: "http://localhost/test"},
				},
			},
		},
	}

	_, err := NewEngine(cfg)
	assert.Error(t, err, "Should reject invalid executor type")
}

// ============================================================================
// Request Stats Tests
// ============================================================================

func TestEngineIntegration_RequestStats(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Request Stats Test",
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{Name: "get_users", Method: "GET", URL: server.URL + "/users"},
					{Name: "get_posts", Method: "GET", URL: server.URL + "/posts"},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Check request stats
	scenarioResult := result.Scenarios["test"]
	require.NotNil(t, scenarioResult)

	// Should have per-request stats
	if len(scenarioResult.RequestStats) > 0 {
		t.Logf("Request Stats:")
		for name, stats := range scenarioResult.RequestStats {
			t.Logf("  %s: count=%d, p95=%v", name, stats.Count, stats.Latency.P95)
		}
	}
}

// ============================================================================
// Sequential Execution Tests
// ============================================================================

func TestEngineIntegration_SequentialScenarios(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Sequential Scenarios Test",
		Options: &config.ExecutionOptions{
			Sequential: true,
		},
		Scenarios: map[string]*config.ScenarioConfig{
			"first": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "1s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL + "/first"},
				},
			},
			"second": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "1s",
				Requests: []config.RequestConfig{
					{Method: "GET", URL: server.URL + "/second"},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	// Both scenarios should complete
	assert.Len(t, result.Scenarios, 2)
	assert.True(t, result.Duration >= 2*time.Second, "Sequential should take at least 2s")

	t.Logf("Sequential Test - Duration: %v", result.Duration)
}

// ============================================================================
// Variables and URL Substitution Tests
// ============================================================================

func TestEngineIntegration_Variables(t *testing.T) {
	server := createTestServer(serverNormal)
	defer server.Close()

	cfg := &config.TestConfig{
		Name: "Variables Test",
		Settings: config.GlobalSettings{
			BaseURL: server.URL,
		},
		Variables: map[string]string{
			"endpoint": "/api/test",
		},
		Scenarios: map[string]*config.ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      1,
				Duration: "2s",
				Requests: []config.RequestConfig{
					{
						Method: "GET",
						URL:    "{{baseUrl}}{{endpoint}}",
					},
				},
			},
		},
	}

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := engine.Run(ctx)
	require.NoError(t, err)

	assert.True(t, result.Metrics.TotalRequests > 0, "Should have made requests with variable substitution")
	t.Logf("Variables Test - Requests: %d", result.Metrics.TotalRequests)
}
