package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

func TestGenerateHTMLString(t *testing.T) {
	// Create a sample test result
	result := createSampleTestResult()

	// Generate HTML
	html, err := GenerateHTMLString(result)
	if err != nil {
		t.Fatalf("GenerateHTMLString failed: %v", err)
	}

	// Verify HTML contains expected content
	expectedContents := []string{
		"<!DOCTYPE html>",
		"<title>Sample Load Test - Performance Test Report</title>",
		"Sample Load Test",
		"✓ PASSED",
		"Total Requests",
		"Throughput",
		"P95 Latency",
		"chart.js",
		"rpsChart",
		"latencyChart",
		"vusChart",
		"errorChart",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(html, expected) {
			t.Errorf("HTML does not contain expected content: %s", expected)
		}
	}

	// Verify JSON time series data is included
	if !strings.Contains(html, `timeSeriesData`) {
		t.Error("HTML does not contain time series data")
	}
}

func TestGenerateHTMLStringNilResult(t *testing.T) {
	_, err := GenerateHTMLString(nil)
	if err == nil {
		t.Error("Expected error for nil result, got nil")
	}
}

func TestGenerateHTML(t *testing.T) {
	// Create a sample test result
	result := createSampleTestResult()

	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-report.html")

	// Generate HTML file
	err := GenerateHTML(result, outputPath)
	if err != nil {
		t.Fatalf("GenerateHTML failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("HTML file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if !strings.Contains(string(content), "<!DOCTYPE html>") {
		t.Error("Generated file does not contain valid HTML")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{500 * time.Microsecond, "500µs"},
		{1500 * time.Microsecond, "1ms"},
		{150 * time.Millisecond, "150ms"},
		{1500 * time.Millisecond, "1.5s"},
		{65 * time.Second, "1m 5s"},
		{2 * time.Hour, "2h"},
		{90 * time.Minute, "1h 30m"},
	}

	for _, tc := range tests {
		result := formatDuration(tc.input)
		if result != tc.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{-1234, "-1,234"},
	}

	for _, tc := range tests {
		result := formatNumber(tc.input)
		if result != tc.expected {
			t.Errorf("formatNumber(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestFormatLatency(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0"},
		{500 * time.Nanosecond, "500ns"},
		{50 * time.Microsecond, "50.0µs"},
		{500 * time.Microsecond, "500µs"},
		{5 * time.Millisecond, "5.00ms"},
		{50 * time.Millisecond, "50.0ms"},
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tc := range tests {
		result := formatLatency(tc.input)
		if result != tc.expected {
			t.Errorf("formatLatency(%v) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tc := range tests {
		result := formatBytes(tc.input)
		if result != tc.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestSuccessRate(t *testing.T) {
	tests := []struct {
		snapshot *metrics.Snapshot
		expected float64
	}{
		{nil, 0},
		{&metrics.Snapshot{TotalRequests: 0}, 0},
		{&metrics.Snapshot{TotalRequests: 100, SuccessRequests: 100}, 100},
		{&metrics.Snapshot{TotalRequests: 100, SuccessRequests: 95}, 95},
		{&metrics.Snapshot{TotalRequests: 100, SuccessRequests: 0}, 0},
	}

	for _, tc := range tests {
		result := successRate(tc.snapshot)
		if result != tc.expected {
			t.Errorf("successRate() = %f, expected %f", result, tc.expected)
		}
	}
}

func TestHasRequestStats(t *testing.T) {
	scenarios := map[string]*engine.ScenarioResult{
		"test1": {
			RequestStats: map[string]engine.RequestStats{},
		},
	}
	if hasRequestStats(scenarios) {
		t.Error("Expected false for empty request stats")
	}

	scenarios["test1"].RequestStats["req1"] = engine.RequestStats{}
	if !hasRequestStats(scenarios) {
		t.Error("Expected true for non-empty request stats")
	}
}

// createSampleTestResult creates a sample TestResult for testing
func createSampleTestResult() *engine.TestResult {
	now := time.Now()

	return &engine.TestResult{
		Name:        "Sample Load Test",
		Description: "A sample test for report generation",
		StartTime:   now.Add(-30 * time.Second),
		EndTime:     now,
		Duration:    30 * time.Second,
		Passed:      true,
		Metrics: &metrics.Snapshot{
			TotalRequests:   1000,
			SuccessRequests: 990,
			FailedRequests:  10,
			TotalBytes:      1048576,
			RPS:             33.33,
			ErrorRate:       0.01,
			Latency: metrics.LatencyStats{
				Min:    10 * time.Millisecond,
				Max:    500 * time.Millisecond,
				Mean:   50 * time.Millisecond,
				StdDev: 20 * time.Millisecond,
				P50:    45 * time.Millisecond,
				P90:    100 * time.Millisecond,
				P95:    150 * time.Millisecond,
				P99:    300 * time.Millisecond,
				Count:  1000,
			},
		},
		TimeSeries: createSampleTimeSeries(30),
		Scenarios: map[string]*engine.ScenarioResult{
			"default": {
				Name:       "default",
				Executor:   "constant-vus",
				Duration:   30 * time.Second,
				Iterations: 1000,
				ActiveVUs:  10,
				Metrics: &metrics.Snapshot{
					TotalRequests:   1000,
					SuccessRequests: 990,
					FailedRequests:  10,
					RPS:             33.33,
					ErrorRate:       0.01,
					Latency: metrics.LatencyStats{
						Mean: 50 * time.Millisecond,
						P95:  150 * time.Millisecond,
					},
				},
				RequestStats: map[string]engine.RequestStats{
					"GET /api/users": {
						Name:  "GET /api/users",
						Count: 1000,
						Latency: metrics.LatencyStats{
							Min:  10 * time.Millisecond,
							Max:  500 * time.Millisecond,
							Mean: 50 * time.Millisecond,
							P50:  45 * time.Millisecond,
							P95:  150 * time.Millisecond,
							P99:  300 * time.Millisecond,
						},
					},
				},
			},
		},
		Thresholds: []engine.ThresholdResult{
			{
				Metric:     "http_req_duration",
				Expression: "p95 < 200ms",
				Passed:     true,
				Value:      "150ms",
			},
			{
				Metric:     "http_req_failed",
				Expression: "rate < 0.05",
				Passed:     true,
				Value:      "0.01",
			},
		},
	}
}

// createSampleTimeSeries creates sample time series data
func createSampleTimeSeries(seconds int) []*metrics.TimeBucket {
	buckets := make([]*metrics.TimeBucket, seconds)
	baseTime := time.Now().Add(-time.Duration(seconds) * time.Second)

	for i := 0; i < seconds; i++ {
		phase := metrics.PhaseSteady
		if i < 5 {
			phase = metrics.PhaseRampUp
		} else if i >= seconds-5 {
			phase = metrics.PhaseRampDown
		}

		buckets[i] = &metrics.TimeBucket{
			Timestamp:         baseTime.Add(time.Duration(i) * time.Second),
			TotalRequests:     int64(i * 33),
			TotalSuccesses:    int64(i * 32),
			TotalFailures:     int64(i),
			TotalBytes:        int64(i * 35000),
			IntervalRequests:  33,
			IntervalRPS:       33.0,
			LatencyMin:        10 * time.Millisecond,
			LatencyMax:        500 * time.Millisecond,
			LatencyP50:        45 * time.Millisecond,
			LatencyP90:        100 * time.Millisecond,
			LatencyP95:        150 * time.Millisecond,
			LatencyP99:        300 * time.Millisecond,
			ActiveVUs:         10,
			Phase:             phase,
			IntervalErrorRate: 0.01,
		}
	}

	return buckets
}
