package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{1 * time.Second, "1.0s"},
		{1*time.Minute + 30*time.Second, "1m 30s"},
		{1*time.Hour + 2*time.Minute + 3*time.Second, "1h 02m 03s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0ms"},
		{500 * time.Microsecond, "500µs"},
		{50 * time.Millisecond, "50ms"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDurationShort(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDurationShort(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		number   int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatNumber(tt.number)
			if result != tt.expected {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.number, result, tt.expected)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\033[32mgreen\033[0m", "green"},
		{"\033[1m\033[34mbold blue\033[0m", "bold blue"},
		{"no \033[31mcolors\033[0m here", "no colors here"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := stripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConsoleOutputCreation(t *testing.T) {
	var buf bytes.Buffer

	output := NewConsoleOutput(ConsoleOutputConfig{
		TestName:       "Test Name",
		ExecutorType:   "constant-vus",
		TotalDuration:  time.Minute,
		UpdateInterval: time.Second,
		Writer:         &buf,
		Quiet:          false,
	})

	if output == nil {
		t.Fatal("NewConsoleOutput returned nil")
	}

	if output.testName != "Test Name" {
		t.Errorf("testName = %q, want %q", output.testName, "Test Name")
	}

	// Should not be TTY when writing to buffer
	if output.IsTTY() {
		t.Error("Expected non-TTY when writing to buffer")
	}
}

func TestProgressBar(t *testing.T) {
	var buf bytes.Buffer

	output := NewConsoleOutput(ConsoleOutputConfig{
		TestName: "Test",
		Writer:   &buf,
	})

	tests := []struct {
		progress float64
		width    int
	}{
		{0.0, 20},
		{0.5, 20},
		{1.0, 20},
	}

	for _, tt := range tests {
		result := output.renderProgressBar(tt.progress, tt.width)

		// Should have brackets
		if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
			t.Errorf("Progress bar should be wrapped in brackets: %q", result)
		}

		// Count runes (not bytes) because we use multi-byte Unicode characters
		runeCount := len([]rune(result))

		// Should be correct length in runes (width + 2 for brackets)
		if runeCount != tt.width+2 {
			t.Errorf("Progress bar rune count = %d, want %d", runeCount, tt.width+2)
		}
	}
}

func TestPrintSummary(t *testing.T) {
	var buf bytes.Buffer

	output := NewConsoleOutput(ConsoleOutputConfig{
		TestName: "Test",
		Writer:   &buf,
		Quiet:    false,
	})

	result := &engine.TestResult{
		Name:     "Test Result",
		Duration: 30 * time.Second,
		Passed:   true,
		Metrics: &metrics.Snapshot{
			TotalRequests:   1000,
			SuccessRequests: 990,
			FailedRequests:  10,
			ErrorRate:       0.01,
			RPS:             33.33,
			Latency: metrics.LatencyStats{
				Min:  10 * time.Millisecond,
				Max:  100 * time.Millisecond,
				Mean: 30 * time.Millisecond,
				P50:  25 * time.Millisecond,
				P90:  50 * time.Millisecond,
				P95:  60 * time.Millisecond,
				P99:  80 * time.Millisecond,
			},
		},
		Thresholds: []engine.ThresholdResult{
			{
				Metric:     "http_req_duration",
				Expression: "p95 < 100ms",
				Passed:     true,
				Value:      "60ms",
			},
		},
	}

	output.PrintSummary(result)

	summary := buf.String()

	// Check that key information is present
	if !strings.Contains(summary, "Test Result") {
		t.Error("Summary should contain test name")
	}
	if !strings.Contains(summary, "Completed ✓") {
		t.Error("Summary should show completion status")
	}
	if !strings.Contains(summary, "1,000") {
		t.Error("Summary should show total requests")
	}
}

func TestStatsFromMetrics(t *testing.T) {
	snapshot := &metrics.Snapshot{
		TotalRequests:   500,
		SuccessRequests: 490,
		FailedRequests:  10,
		ErrorRate:       0.02,
		RPS:             50.0,
		ActiveVUs:       10,
		CurrentPhase:    metrics.PhaseSteady,
		Elapsed:         30 * time.Second,
		Latency: metrics.LatencyStats{
			Mean: 20 * time.Millisecond,
			P95:  50 * time.Millisecond,
		},
	}

	stats := StatsFromMetrics(snapshot, 0.5, time.Minute, 20, 2, 3)

	if stats.Progress != 0.5 {
		t.Errorf("Progress = %f, want 0.5", stats.Progress)
	}
	if stats.ActiveVUs != 10 {
		t.Errorf("ActiveVUs = %d, want 10", stats.ActiveVUs)
	}
	if stats.TargetVUs != 20 {
		t.Errorf("TargetVUs = %d, want 20", stats.TargetVUs)
	}
	if stats.CurrentRPS != 50.0 {
		t.Errorf("CurrentRPS = %f, want 50.0", stats.CurrentRPS)
	}
	if stats.CurrentStage != 2 {
		t.Errorf("CurrentStage = %d, want 2", stats.CurrentStage)
	}
	if stats.TotalStages != 3 {
		t.Errorf("TotalStages = %d, want 3", stats.TotalStages)
	}
}

func TestQuietMode(t *testing.T) {
	var buf bytes.Buffer

	output := NewConsoleOutput(ConsoleOutputConfig{
		TestName: "Test",
		Writer:   &buf,
		Quiet:    true,
	})

	// PrintHeader should not output in quiet mode
	output.PrintHeader()
	if buf.Len() != 0 {
		t.Error("PrintHeader should not output in quiet mode")
	}

	// Update should not output in quiet mode
	stats := &LiveStats{
		Progress:  0.5,
		ActiveVUs: 10,
		TargetVUs: 10,
	}
	output.Update(stats)
	if buf.Len() != 0 {
		t.Error("Update should not output in quiet mode")
	}

	// PrintSummary should still output pass/fail status in quiet mode
	buf.Reset()
	result := &engine.TestResult{
		Name:   "Test",
		Passed: true,
	}
	output.PrintSummary(result)
	if !strings.Contains(buf.String(), "PASSED") {
		t.Error("PrintSummary should output PASSED in quiet mode")
	}
}
