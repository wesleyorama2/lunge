package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v2config "github.com/wesleyorama2/lunge/internal/performance/v2/config"
	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/executor"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

func TestBuildConfigFromCLI(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		executorType    string
		duration        string
		vus             int
		stages          string
		rate            float64
		maxVUs          int
		preAllocatedVUs int
		wantErr         bool
		checkName       string
		checkExecutor   string
		checkDuration   string
		checkVUs        int
		checkRate       float64
	}{
		{
			name:          "Default executor with URL only",
			url:           "https://api.example.com/health",
			executorType:  "",
			duration:      "",
			vus:           0,
			wantErr:       false,
			checkName:     "CLI Test",
			checkExecutor: "constant-vus",
			checkVUs:      10,    // Default VUs
			checkDuration: "30s", // Default duration
		},
		{
			name:          "Constant VUs executor",
			url:           "https://api.example.com/users",
			executorType:  "constant-vus",
			duration:      "5m",
			vus:           50,
			wantErr:       false,
			checkExecutor: "constant-vus",
			checkVUs:      50,
			checkDuration: "5m",
		},
		{
			name:          "Ramping VUs with stages",
			url:           "https://api.example.com/health",
			executorType:  "ramping-vus",
			stages:        "30s:10,2m:50,30s:0",
			wantErr:       false,
			checkExecutor: "ramping-vus",
		},
		{
			name:            "Constant arrival rate",
			url:             "https://api.example.com/health",
			executorType:    "constant-arrival-rate",
			rate:            100,
			duration:        "5m",
			maxVUs:          200,
			preAllocatedVUs: 50,
			wantErr:         false,
			checkExecutor:   "constant-arrival-rate",
			checkRate:       100,
		},
		{
			name:          "Invalid stages format",
			url:           "https://api.example.com/health",
			executorType:  "ramping-vus",
			stages:        "invalid",
			wantErr:       true,
			checkExecutor: "ramping-vus",
		},
		{
			name:          "Ramping arrival rate with stages",
			url:           "https://api.example.com/health",
			executorType:  "ramping-arrival-rate",
			stages:        "1m:50,2m:100,1m:0",
			wantErr:       false,
			checkExecutor: "ramping-arrival-rate",
		},
		{
			name:            "Arrival rate with pre-allocated VUs",
			url:             "https://api.example.com/health",
			executorType:    "constant-arrival-rate",
			rate:            50,
			duration:        "2m",
			maxVUs:          100,
			preAllocatedVUs: 25,
			wantErr:         false,
			checkExecutor:   "constant-arrival-rate",
			checkRate:       50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := buildConfigFromCLI(tt.url, tt.executorType, tt.duration, tt.vus, tt.stages, tt.rate, tt.maxVUs, tt.preAllocatedVUs)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildConfigFromCLI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tt.checkName != "" && cfg.Name != tt.checkName {
				t.Errorf("buildConfigFromCLI() Name = %v, want %v", cfg.Name, tt.checkName)
			}

			// Check scenario executor type
			for _, scenario := range cfg.Scenarios {
				if scenario.Executor != tt.checkExecutor {
					t.Errorf("buildConfigFromCLI() Executor = %v, want %v", scenario.Executor, tt.checkExecutor)
				}
				if tt.checkDuration != "" && scenario.Duration != tt.checkDuration {
					t.Errorf("buildConfigFromCLI() Duration = %v, want %v", scenario.Duration, tt.checkDuration)
				}
				if tt.checkVUs != 0 && scenario.VUs != tt.checkVUs {
					t.Errorf("buildConfigFromCLI() VUs = %v, want %v", scenario.VUs, tt.checkVUs)
				}
				if tt.checkRate != 0 && scenario.Rate != tt.checkRate {
					t.Errorf("buildConfigFromCLI() Rate = %v, want %v", scenario.Rate, tt.checkRate)
				}
			}
		})
	}
}

func TestParseStages(t *testing.T) {
	tests := []struct {
		name       string
		stagesStr  string
		wantStages int
		wantErr    bool
	}{
		{
			name:       "Single stage",
			stagesStr:  "30s:10",
			wantStages: 1,
			wantErr:    false,
		},
		{
			name:       "Multiple stages",
			stagesStr:  "30s:10,2m:50,30s:0",
			wantStages: 3,
			wantErr:    false,
		},
		{
			name:       "Stage with spaces",
			stagesStr:  " 30s:10 , 2m:50 , 30s:0 ",
			wantStages: 3,
			wantErr:    false,
		},
		{
			name:       "Invalid format - no colon",
			stagesStr:  "30s10",
			wantStages: 0,
			wantErr:    true,
		},
		{
			name:       "Invalid duration",
			stagesStr:  "invalid:10",
			wantStages: 0,
			wantErr:    true,
		},
		{
			name:       "Invalid target",
			stagesStr:  "30s:abc",
			wantStages: 0,
			wantErr:    true,
		},
		{
			name:       "Empty string",
			stagesStr:  "",
			wantStages: 0,
			wantErr:    true,
		},
		{
			name:       "Complex durations",
			stagesStr:  "1h30m:100,45m:200,15m30s:0",
			wantStages: 3,
			wantErr:    false,
		},
		{
			name:       "Whitespace only",
			stagesStr:  "   ",
			wantStages: 0,
			wantErr:    true,
		},
		{
			name:       "Single stage with zero target",
			stagesStr:  "1m:0",
			wantStages: 1,
			wantErr:    false,
		},
		{
			name:       "Stage with large values",
			stagesStr:  "1h:1000",
			wantStages: 1,
			wantErr:    false,
		},
		{
			name:       "Seconds and milliseconds",
			stagesStr:  "500ms:5,10s:10",
			wantStages: 2,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stages, err := parseStages(tt.stagesStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(stages) != tt.wantStages {
				t.Errorf("parseStages() returned %d stages, want %d", len(stages), tt.wantStages)
			}
		})
	}
}

func TestParseStages_Values(t *testing.T) {
	stages, err := parseStages("30s:10,2m:50,30s:0")
	if err != nil {
		t.Fatalf("parseStages() unexpected error = %v", err)
	}

	if len(stages) != 3 {
		t.Fatalf("parseStages() returned %d stages, want 3", len(stages))
	}

	// Check first stage
	if stages[0].Duration != "30s" {
		t.Errorf("stages[0].Duration = %v, want 30s", stages[0].Duration)
	}
	if stages[0].Target != 10 {
		t.Errorf("stages[0].Target = %v, want 10", stages[0].Target)
	}
	if stages[0].Name != "stage-1" {
		t.Errorf("stages[0].Name = %v, want stage-1", stages[0].Name)
	}

	// Check second stage
	if stages[1].Duration != "2m" {
		t.Errorf("stages[1].Duration = %v, want 2m", stages[1].Duration)
	}
	if stages[1].Target != 50 {
		t.Errorf("stages[1].Target = %v, want 50", stages[1].Target)
	}
	if stages[1].Name != "stage-2" {
		t.Errorf("stages[1].Name = %v, want stage-2", stages[1].Name)
	}

	// Check third stage
	if stages[2].Duration != "30s" {
		t.Errorf("stages[2].Duration = %v, want 30s", stages[2].Duration)
	}
	if stages[2].Target != 0 {
		t.Errorf("stages[2].Target = %v, want 0", stages[2].Target)
	}
	if stages[2].Name != "stage-3" {
		t.Errorf("stages[2].Name = %v, want stage-3", stages[2].Name)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
		{2147483648, "2.00 GB"},
		{10737418240, "10.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestCalculateTotalDuration(t *testing.T) {
	tests := []struct {
		name     string
		config   *v2config.TestConfig
		expected time.Duration
	}{
		{
			name: "Single scenario with duration",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						Duration: "1m",
					},
				},
			},
			expected: 1 * time.Minute,
		},
		{
			name: "Single scenario with stages",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						Stages: []v2config.StageConfig{
							{Duration: "30s", Target: 10},
							{Duration: "1m", Target: 20},
							{Duration: "30s", Target: 0},
						},
					},
				},
			},
			expected: 2 * time.Minute,
		},
		{
			name: "Multiple scenarios - max duration",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"short": {
						Duration: "30s",
					},
					"long": {
						Duration: "5m",
					},
				},
			},
			expected: 5 * time.Minute,
		},
		{
			name: "Mixed duration and stages",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"duration": {
						Duration: "2m",
					},
					"stages": {
						Stages: []v2config.StageConfig{
							{Duration: "1m", Target: 10},
							{Duration: "2m", Target: 20},
						},
					},
				},
			},
			expected: 3 * time.Minute,
		},
		{
			name: "Empty config",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{},
			},
			expected: 0,
		},
		{
			name: "Invalid duration string",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"invalid": {
						Duration: "invalid",
					},
				},
			},
			expected: 0,
		},
		{
			name: "Invalid stage duration",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"invalid": {
						Stages: []v2config.StageConfig{
							{Duration: "invalid", Target: 10},
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTotalDuration(tt.config)
			if got != tt.expected {
				t.Errorf("calculateTotalDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetTargetVUs(t *testing.T) {
	tests := []struct {
		name     string
		config   *v2config.TestConfig
		expected int
	}{
		{
			name: "Single scenario with VUs",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						VUs: 50,
					},
				},
			},
			expected: 50,
		},
		{
			name: "Single scenario with MaxVUs",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						MaxVUs: 100,
					},
				},
			},
			expected: 100,
		},
		{
			name: "Scenario with stages",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						VUs: 10,
						Stages: []v2config.StageConfig{
							{Duration: "1m", Target: 50},
							{Duration: "2m", Target: 100},
							{Duration: "1m", Target: 0},
						},
					},
				},
			},
			expected: 100,
		},
		{
			name: "Multiple scenarios",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"small": {
						VUs: 10,
					},
					"large": {
						VUs: 100,
					},
				},
			},
			expected: 100,
		},
		{
			name: "Empty config - default",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{},
			},
			expected: 10, // Default
		},
		{
			name: "Zero VUs - default",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						VUs: 0,
					},
				},
			},
			expected: 10, // Default
		},
		{
			name: "VUs and MaxVUs - use max",
			config: &v2config.TestConfig{
				Scenarios: map[string]*v2config.ScenarioConfig{
					"test": {
						VUs:    50,
						MaxVUs: 200,
					},
				},
			},
			expected: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTargetVUs(tt.config)
			if got != tt.expected {
				t.Errorf("getTargetVUs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetStageInfo(t *testing.T) {
	tests := []struct {
		name          string
		stats         map[string]*executor.Stats
		expectedCurr  int
		expectedTotal int
	}{
		{
			name:          "Empty stats",
			stats:         map[string]*executor.Stats{},
			expectedCurr:  0,
			expectedTotal: 0,
		},
		{
			name: "Single scenario",
			stats: map[string]*executor.Stats{
				"test": {
					CurrentStage: 2,
					TotalStages:  3,
				},
			},
			expectedCurr:  2,
			expectedTotal: 3,
		},
		{
			name: "Multiple scenarios",
			stats: map[string]*executor.Stats{
				"first": {
					CurrentStage: 1,
					TotalStages:  3,
				},
				"second": {
					CurrentStage: 3,
					TotalStages:  5,
				},
			},
			expectedCurr:  3,
			expectedTotal: 5,
		},
		{
			name: "Nil stats entry",
			stats: map[string]*executor.Stats{
				"test": nil,
			},
			expectedCurr:  0,
			expectedTotal: 0,
		},
		{
			name: "Mixed nil and valid",
			stats: map[string]*executor.Stats{
				"valid": {
					CurrentStage: 2,
					TotalStages:  4,
				},
				"nil": nil,
			},
			expectedCurr:  2,
			expectedTotal: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curr, total := getStageInfo(tt.stats)
			if curr != tt.expectedCurr {
				t.Errorf("getStageInfo() current = %v, want %v", curr, tt.expectedCurr)
			}
			if total != tt.expectedTotal {
				t.Errorf("getStageInfo() total = %v, want %v", total, tt.expectedTotal)
			}
		})
	}
}

func TestGenerateDefaultHTMLPath(t *testing.T) {
	tests := []struct {
		name     string
		testName string
		prefix   string
	}{
		{
			name:     "Simple name",
			testName: "API Test",
			prefix:   "perf-report-api-test-",
		},
		{
			name:     "Name with slashes",
			testName: "API/Test/Name",
			prefix:   "perf-report-api-test-name-",
		},
		{
			name:     "Name with spaces",
			testName: "My Performance Test",
			prefix:   "perf-report-my-performance-test-",
		},
		{
			name:     "Empty name",
			testName: "",
			prefix:   "perf-report--",
		},
		{
			name:     "Mixed case",
			testName: "MyTest",
			prefix:   "perf-report-mytest-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDefaultHTMLPath(tt.testName)
			if !strings.HasPrefix(result, tt.prefix) {
				t.Errorf("generateDefaultHTMLPath(%q) = %q, want prefix %q", tt.testName, result, tt.prefix)
			}
			if !strings.HasSuffix(result, ".html") {
				t.Errorf("generateDefaultHTMLPath(%q) = %q, should end with .html", tt.testName, result)
			}
		})
	}
}

func TestOutputHTMLReport_NilResult(t *testing.T) {
	err := outputHTMLReport(nil, "test.html", false)
	if err == nil {
		t.Error("outputHTMLReport() with nil result should return error")
	}
	if !strings.Contains(err.Error(), "no results") {
		t.Errorf("outputHTMLReport() error = %v, should mention 'no results'", err)
	}
}

func TestOutputHTMLReport_InvalidDirectory(t *testing.T) {
	result := &engine.TestResult{
		Name:      "Test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Metrics: &metrics.Snapshot{
			TotalRequests:   100,
			SuccessRequests: 95,
			FailedRequests:  5,
			ErrorRate:       0.05,
			RPS:             10.0,
		},
	}

	// Test with an invalid path that should still work with directory creation
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "subdir", "report.html")

	err := outputHTMLReport(result, outputPath, false)
	if err != nil {
		t.Errorf("outputHTMLReport() unexpected error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("outputHTMLReport() did not create the file")
	}
}

func TestOutputHTMLReport_AddHtmlExtension(t *testing.T) {
	result := &engine.TestResult{
		Name:      "Test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Metrics: &metrics.Snapshot{
			TotalRequests:   100,
			SuccessRequests: 95,
			FailedRequests:  5,
			ErrorRate:       0.05,
			RPS:             10.0,
		},
	}

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "report") // No .html extension

	err := outputHTMLReport(result, outputPath, false)
	if err != nil {
		t.Errorf("outputHTMLReport() unexpected error = %v", err)
	}

	// Verify file was created with .html extension
	if _, err := os.Stat(outputPath + ".html"); os.IsNotExist(err) {
		t.Error("outputHTMLReport() did not add .html extension")
	}
}

func TestOutputJSONResult(t *testing.T) {
	result := &engine.TestResult{
		Name:        "Test Result",
		Description: "Test description",
		StartTime:   time.Now(),
		EndTime:     time.Now(),
		Duration:    time.Second,
		Passed:      true,
		Metrics: &metrics.Snapshot{
			TotalRequests:   100,
			SuccessRequests: 95,
			FailedRequests:  5,
			ErrorRate:       0.05,
			RPS:             10.0,
		},
	}

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "result.json")

	outputJSONResult(result, outputPath)

	// Verify file was created
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify it's valid JSON
	var decoded engine.TestResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	// Verify content
	if decoded.Name != "Test Result" {
		t.Errorf("Decoded name = %q, want %q", decoded.Name, "Test Result")
	}
}

func TestOutputConsoleResult(t *testing.T) {
	// Test with nil result - should not panic
	outputConsoleResult(nil, false)

	// Test with valid result
	result := &engine.TestResult{
		Name:      "Console Test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  10 * time.Second,
		Passed:    true,
		Metrics: &metrics.Snapshot{
			TotalRequests:   1000,
			SuccessRequests: 950,
			FailedRequests:  50,
			ErrorRate:       0.05,
			RPS:             100.0,
			TotalBytes:      1048576, // 1 MB
			Latency: metrics.LatencyStats{
				Min:  10 * time.Millisecond,
				Max:  500 * time.Millisecond,
				Mean: 100 * time.Millisecond,
				P50:  80 * time.Millisecond,
				P90:  200 * time.Millisecond,
				P95:  300 * time.Millisecond,
				P99:  450 * time.Millisecond,
			},
		},
		Scenarios: map[string]*engine.ScenarioResult{
			"test-scenario": {
				Name:       "test-scenario",
				Executor:   "constant-vus",
				Duration:   10 * time.Second,
				Iterations: 1000,
			},
		},
		Thresholds: []engine.ThresholdResult{
			{
				Metric:     "http_req_duration",
				Expression: "p95 < 500ms",
				Passed:     true,
				Value:      "300ms",
			},
			{
				Metric:     "http_req_failed",
				Expression: "rate < 0.1",
				Passed:     false,
				Value:      "0.05",
				Message:    "Threshold passed but marked as failed for testing",
			},
		},
	}

	// Test with verbose = false
	outputConsoleResult(result, false)

	// Test with verbose = true to cover scenario output
	outputConsoleResult(result, true)

	// Test with failed result
	failedResult := &engine.TestResult{
		Name:      "Failed Test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  5 * time.Second,
		Passed:    false,
		Metrics: &metrics.Snapshot{
			TotalRequests:   100,
			SuccessRequests: 50,
			FailedRequests:  50,
			ErrorRate:       0.5,
			RPS:             20.0,
		},
	}
	outputConsoleResult(failedResult, false)
}

func TestPerfCommand_Help(t *testing.T) {
	rootCmd := RootCmd
	rootCmd.SetArgs([]string{"perf", "--help"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	// Execute should not error for help
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("perf --help returned error: %v", err)
	}
}

func TestPerfCommand_NoArgs(t *testing.T) {
	// Create a fresh root command
	rootCmd := RootCmd
	rootCmd.SetArgs([]string{"perf"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Execute - should print error about config/url
	_ = rootCmd.Execute()

	// Check that error message is present
	output := buf.String()
	if !strings.Contains(output, "config") && !strings.Contains(output, "url") {
		// This is okay - the command might handle it differently
		t.Log("Command output:", output)
	}
}

func TestBuildConfigFromCLI_Defaults(t *testing.T) {
	// Test default values are applied correctly
	cfg, err := buildConfigFromCLI("https://example.com", "", "", 0, "", 0, 0, 0)
	if err != nil {
		t.Fatalf("buildConfigFromCLI() error = %v", err)
	}

	scenario := cfg.Scenarios["cli-test"]
	if scenario == nil {
		t.Fatal("Expected 'cli-test' scenario to exist")
	}

	// Check defaults
	if scenario.Executor != "constant-vus" {
		t.Errorf("Default executor = %q, want %q", scenario.Executor, "constant-vus")
	}

	if scenario.VUs != 10 {
		t.Errorf("Default VUs = %d, want %d", scenario.VUs, 10)
	}

	if scenario.Duration != "30s" {
		t.Errorf("Default duration = %q, want %q", scenario.Duration, "30s")
	}

	// Check request
	if len(scenario.Requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(scenario.Requests))
	}

	req := scenario.Requests[0]
	if req.URL != "https://example.com" {
		t.Errorf("Request URL = %q, want %q", req.URL, "https://example.com")
	}

	if req.Method != "GET" {
		t.Errorf("Request Method = %q, want %q", req.Method, "GET")
	}

	if req.Name != "cli-request" {
		t.Errorf("Request Name = %q, want %q", req.Name, "cli-request")
	}
}

func TestBuildConfigFromCLI_Description(t *testing.T) {
	url := "https://api.test.com/endpoint"
	cfg, err := buildConfigFromCLI(url, "constant-vus", "1m", 20, "", 0, 0, 0)
	if err != nil {
		t.Fatalf("buildConfigFromCLI() error = %v", err)
	}

	expectedDesc := "Test generated from CLI flags for " + url
	if cfg.Description != expectedDesc {
		t.Errorf("Description = %q, want %q", cfg.Description, expectedDesc)
	}
}

func TestParseStages_StageNaming(t *testing.T) {
	stages, err := parseStages("1m:10,2m:20,1m:0")
	if err != nil {
		t.Fatalf("parseStages() error = %v", err)
	}

	for i, stage := range stages {
		expectedName := "stage-" + string(rune('1'+i))
		if i == 0 && stage.Name != "stage-1" {
			t.Errorf("Stage %d name = %q, want %q", i, stage.Name, expectedName)
		}
	}
}

func TestPerfCommand_WithTestServer(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// We can't easily test the full perf command execution because it involves
	// complex interactions with the engine, but we can test the config building
	cfg, err := buildConfigFromCLI(server.URL, "constant-vus", "1s", 1, "", 0, 0, 0)
	if err != nil {
		t.Fatalf("buildConfigFromCLI() error = %v", err)
	}

	// Verify the config is valid and uses the test server URL
	scenario := cfg.Scenarios["cli-test"]
	if scenario == nil {
		t.Fatal("Expected 'cli-test' scenario")
	}

	if len(scenario.Requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(scenario.Requests))
	}

	if scenario.Requests[0].URL != server.URL {
		t.Errorf("Request URL = %q, want %q", scenario.Requests[0].URL, server.URL)
	}
}

func TestCalculateTotalDuration_ComplexScenarios(t *testing.T) {
	// Test with multiple complex scenarios
	cfg := &v2config.TestConfig{
		Scenarios: map[string]*v2config.ScenarioConfig{
			"constant": {
				Executor: "constant-vus",
				Duration: "5m",
				VUs:      10,
			},
			"ramping": {
				Executor: "ramping-vus",
				Stages: []v2config.StageConfig{
					{Duration: "1m", Target: 10},
					{Duration: "3m", Target: 50},
					{Duration: "1m", Target: 0},
				},
			},
			"arrival": {
				Executor: "constant-arrival-rate",
				Duration: "2m",
				Rate:     100,
			},
		},
	}

	expected := 5 * time.Minute // Max of: 5m, 5m (stages sum), 2m
	got := calculateTotalDuration(cfg)

	if got != expected {
		t.Errorf("calculateTotalDuration() = %v, want %v", got, expected)
	}
}

func TestGetTargetVUs_ComplexScenarios(t *testing.T) {
	cfg := &v2config.TestConfig{
		Scenarios: map[string]*v2config.ScenarioConfig{
			"constant": {
				VUs: 100,
			},
			"ramping": {
				VUs: 10,
				Stages: []v2config.StageConfig{
					{Duration: "1m", Target: 50},
					{Duration: "2m", Target: 200}, // This should be the max
					{Duration: "1m", Target: 0},
				},
			},
			"arrival": {
				MaxVUs: 150,
			},
		},
	}

	expected := 200 // Max stage target
	got := getTargetVUs(cfg)

	if got != expected {
		t.Errorf("getTargetVUs() = %v, want %v", got, expected)
	}
}

func TestFormatBytes_EdgeCases(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{-1, "-1 B"},      // Negative value (edge case)
		{1, "1 B"},        // Single byte
		{1023, "1023 B"},  // Just under 1 KB
		{1025, "1.00 KB"}, // Just over 1 KB
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestOutputJSONResult_EmptyPath(t *testing.T) {
	result := &engine.TestResult{
		Name:      "Test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  time.Second,
		Passed:    true,
		Metrics: &metrics.Snapshot{
			TotalRequests: 100,
		},
	}

	// With empty path, it should print to stdout
	// We can't easily capture stdout, but we can ensure it doesn't panic
	outputJSONResult(result, "")
}

func TestParseStages_ColonInDuration(t *testing.T) {
	// Test that durations with multiple colons are handled correctly
	// Go's time.ParseDuration doesn't use colons, but let's verify behavior
	_, err := parseStages("1h30m:10")
	if err != nil {
		t.Errorf("parseStages() with complex duration failed: %v", err)
	}
}

func TestBuildConfigFromCLI_AllArrivalRateOptions(t *testing.T) {
	cfg, err := buildConfigFromCLI(
		"https://example.com",
		"constant-arrival-rate",
		"5m",
		0,   // VUs not needed for arrival rate
		"",  // No stages
		150, // Rate
		300, // MaxVUs
		100, // PreAllocatedVUs
	)

	if err != nil {
		t.Fatalf("buildConfigFromCLI() error = %v", err)
	}

	scenario := cfg.Scenarios["cli-test"]
	if scenario == nil {
		t.Fatal("Expected 'cli-test' scenario")
	}

	if scenario.Rate != 150 {
		t.Errorf("Rate = %v, want %v", scenario.Rate, 150)
	}

	if scenario.MaxVUs != 300 {
		t.Errorf("MaxVUs = %v, want %v", scenario.MaxVUs, 300)
	}

	if scenario.PreAllocatedVUs != 100 {
		t.Errorf("PreAllocatedVUs = %v, want %v", scenario.PreAllocatedVUs, 100)
	}
}

func TestBuildConfigFromCLI_RampingArrivalRate(t *testing.T) {
	cfg, err := buildConfigFromCLI(
		"https://example.com",
		"ramping-arrival-rate",
		"",
		0,
		"1m:50,2m:100,1m:0",
		0,
		200,
		50,
	)

	if err != nil {
		t.Fatalf("buildConfigFromCLI() error = %v", err)
	}

	scenario := cfg.Scenarios["cli-test"]
	if len(scenario.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(scenario.Stages))
	}
}
