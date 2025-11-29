package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDurationString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "standard seconds",
			input:    "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "standard minutes",
			input:    "2m",
			expected: 2 * time.Minute,
		},
		{
			name:     "standard hours",
			input:    "1h",
			expected: time.Hour,
		},
		{
			name:     "milliseconds",
			input:    "500ms",
			expected: 500 * time.Millisecond,
		},
		{
			name:     "combined duration",
			input:    "1h30m",
			expected: 90 * time.Minute,
		},
		{
			name:     "integer as seconds",
			input:    "30",
			expected: 30 * time.Second,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:    "invalid format",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDurationString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDurationString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseDurationString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseConfig_YAML(t *testing.T) {
	yamlConfig := `
name: "Test Config"
description: "A test configuration"
settings:
  baseUrl: "https://api.example.com"
  timeout: 30s
variables:
  apiKey: "test-key"
scenarios:
  browse:
    executor: constant-vus
    vus: 10
    duration: 30s
    requests:
      - name: "Get Users"
        method: GET
        url: "{{baseUrl}}/api/users"
thresholds:
  http_req_duration:
    - "p95 < 500ms"
`
	config, err := ParseConfig([]byte(yamlConfig), "test.yaml")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if config.Name != "Test Config" {
		t.Errorf("Name = %v, want %v", config.Name, "Test Config")
	}

	if config.Description != "A test configuration" {
		t.Errorf("Description = %v, want %v", config.Description, "A test configuration")
	}

	if config.Settings.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %v, want %v", config.Settings.BaseURL, "https://api.example.com")
	}

	if config.Variables["apiKey"] != "test-key" {
		t.Errorf("Variables[apiKey] = %v, want %v", config.Variables["apiKey"], "test-key")
	}

	browse, ok := config.Scenarios["browse"]
	if !ok {
		t.Fatal("Scenario 'browse' not found")
	}

	if browse.Executor != "constant-vus" {
		t.Errorf("Executor = %v, want %v", browse.Executor, "constant-vus")
	}

	if browse.VUs != 10 {
		t.Errorf("VUs = %v, want %v", browse.VUs, 10)
	}

	if browse.Duration != "30s" {
		t.Errorf("Duration = %v, want %v", browse.Duration, "30s")
	}

	if len(browse.Requests) != 1 {
		t.Fatalf("len(Requests) = %v, want %v", len(browse.Requests), 1)
	}

	if browse.Requests[0].Name != "Get Users" {
		t.Errorf("Request.Name = %v, want %v", browse.Requests[0].Name, "Get Users")
	}

	if browse.Requests[0].Method != "GET" {
		t.Errorf("Request.Method = %v, want %v", browse.Requests[0].Method, "GET")
	}
}

func TestParseConfig_JSON(t *testing.T) {
	jsonConfig := `{
		"name": "JSON Test Config",
		"settings": {
			"baseUrl": "https://api.example.com"
		},
		"scenarios": {
			"test": {
				"executor": "ramping-vus",
				"stages": [
					{"duration": "30s", "target": 10},
					{"duration": "1m", "target": 10},
					{"duration": "30s", "target": 0}
				],
				"requests": [
					{"method": "GET", "url": "/api/health"}
				]
			}
		}
	}`

	config, err := ParseConfig([]byte(jsonConfig), "test.json")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if config.Name != "JSON Test Config" {
		t.Errorf("Name = %v, want %v", config.Name, "JSON Test Config")
	}

	test, ok := config.Scenarios["test"]
	if !ok {
		t.Fatal("Scenario 'test' not found")
	}

	if test.Executor != "ramping-vus" {
		t.Errorf("Executor = %v, want %v", test.Executor, "ramping-vus")
	}

	if len(test.Stages) != 3 {
		t.Fatalf("len(Stages) = %v, want %v", len(test.Stages), 3)
	}

	if test.Stages[0].Duration != "30s" {
		t.Errorf("Stages[0].Duration = %v, want %v", test.Stages[0].Duration, "30s")
	}

	if test.Stages[0].Target != 10 {
		t.Errorf("Stages[0].Target = %v, want %v", test.Stages[0].Target, 10)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-config.yaml")

	yamlContent := `
name: "File Test"
scenarios:
  test:
    executor: constant-vus
    vus: 5
    duration: 10s
    requests:
      - method: GET
        url: "/test"
`
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.Name != "File Test" {
		t.Errorf("Name = %v, want %v", config.Name, "File Test")
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadConfig() should return error for nonexistent file")
	}
}

func TestParseScenarioDuration(t *testing.T) {
	tests := []struct {
		name     string
		scenario *ScenarioConfig
		expected time.Duration
		wantErr  bool
	}{
		{
			name: "explicit duration",
			scenario: &ScenarioConfig{
				Duration: "1m30s",
			},
			expected: 90 * time.Second,
		},
		{
			name: "stages sum",
			scenario: &ScenarioConfig{
				Stages: []StageConfig{
					{Duration: "30s", Target: 10},
					{Duration: "1m", Target: 10},
					{Duration: "30s", Target: 0},
				},
			},
			expected: 2 * time.Minute,
		},
		{
			name: "explicit duration overrides stages",
			scenario: &ScenarioConfig{
				Duration: "5m",
				Stages: []StageConfig{
					{Duration: "30s", Target: 10},
				},
			},
			expected: 5 * time.Minute,
		},
		{
			name: "no duration or stages",
			scenario: &ScenarioConfig{
				VUs: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseScenarioDuration(tt.scenario)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScenarioDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseScenarioDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResolveVariables(t *testing.T) {
	globals := map[string]string{
		"apiKey": "secret-key",
		"host":   "example.com",
	}

	settings := &GlobalSettings{
		BaseURL: "https://api.example.com",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "global variable",
			input:    "Bearer {{apiKey}}",
			expected: "Bearer secret-key",
		},
		{
			name:     "baseUrl variable",
			input:    "{{baseUrl}}/api/users",
			expected: "https://api.example.com/api/users",
		},
		{
			name:     "multiple variables",
			input:    "https://{{host}}/api?key={{apiKey}}",
			expected: "https://example.com/api?key=secret-key",
		},
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "unresolved variable",
			input:    "{{unknown}}",
			expected: "{{unknown}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveVariables(tt.input, globals, settings)
			if got != tt.expected {
				t.Errorf("ResolveVariables() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMergeVariables(t *testing.T) {
	first := map[string]string{
		"a": "1",
		"b": "2",
	}

	second := map[string]string{
		"b": "3", // Override
		"c": "4",
	}

	result := MergeVariables(first, second)

	if result["a"] != "1" {
		t.Errorf("result[a] = %v, want %v", result["a"], "1")
	}

	if result["b"] != "3" {
		t.Errorf("result[b] = %v, want %v (should be overridden)", result["b"], "3")
	}

	if result["c"] != "4" {
		t.Errorf("result[c] = %v, want %v", result["c"], "4")
	}
}

func TestConvertToExecutorConfig(t *testing.T) {
	scenario := &ScenarioConfig{
		Executor:        "constant-arrival-rate",
		Rate:            100,
		Duration:        "1m",
		PreAllocatedVUs: 10,
		MaxVUs:          50,
		GracefulStop:    "30s",
		Pacing: &PacingConfig{
			Type:     "constant",
			Duration: "100ms",
		},
	}

	execConfig, err := ConvertToExecutorConfig("test", scenario)
	if err != nil {
		t.Fatalf("ConvertToExecutorConfig() error = %v", err)
	}

	if execConfig.Name != "test" {
		t.Errorf("Name = %v, want %v", execConfig.Name, "test")
	}

	if execConfig.Type != "constant-arrival-rate" {
		t.Errorf("Type = %v, want %v", execConfig.Type, "constant-arrival-rate")
	}

	if execConfig.Rate != 100 {
		t.Errorf("Rate = %v, want %v", execConfig.Rate, 100)
	}

	if execConfig.Duration != time.Minute {
		t.Errorf("Duration = %v, want %v", execConfig.Duration, time.Minute)
	}

	if execConfig.PreAllocatedVUs != 10 {
		t.Errorf("PreAllocatedVUs = %v, want %v", execConfig.PreAllocatedVUs, 10)
	}

	if execConfig.MaxVUs != 50 {
		t.Errorf("MaxVUs = %v, want %v", execConfig.MaxVUs, 50)
	}

	if execConfig.GracefulStop != 30*time.Second {
		t.Errorf("GracefulStop = %v, want %v", execConfig.GracefulStop, 30*time.Second)
	}

	if execConfig.Pacing == nil {
		t.Fatal("Pacing should not be nil")
	}

	if execConfig.Pacing.Type != "constant" {
		t.Errorf("Pacing.Type = %v, want %v", execConfig.Pacing.Type, "constant")
	}

	if execConfig.Pacing.Duration != 100*time.Millisecond {
		t.Errorf("Pacing.Duration = %v, want %v", execConfig.Pacing.Duration, 100*time.Millisecond)
	}
}

func TestApplyDefaults(t *testing.T) {
	config := &TestConfig{
		Name: "Test",
		Scenarios: map[string]*ScenarioConfig{
			"test1": {
				Executor: "constant-vus",
				VUs:      0, // Should remain 0, not defaulted as value is 0
				Duration: "30s",
				Requests: []RequestConfig{
					{Method: "", URL: "/test"}, // Method should default to GET
				},
			},
			"test2": {
				Executor: "", // Should default to constant-vus
				Duration: "30s",
				Requests: []RequestConfig{
					{Name: "", Method: "POST", URL: "/test"}, // Name should be auto-generated
				},
			},
		},
	}

	ApplyDefaults(config)

	// Check global settings defaults
	if config.Settings.Timeout == 0 {
		t.Error("Timeout should have a default value")
	}

	if config.Settings.UserAgent != "lunge/2.0" {
		t.Errorf("UserAgent = %v, want %v", config.Settings.UserAgent, "lunge/2.0")
	}

	// Check executor defaults
	if config.Scenarios["test2"].Executor != "constant-vus" {
		t.Errorf("Executor = %v, want %v", config.Scenarios["test2"].Executor, "constant-vus")
	}

	// Check request defaults
	if config.Scenarios["test1"].Requests[0].Method != "GET" {
		t.Errorf("Method = %v, want %v", config.Scenarios["test1"].Requests[0].Method, "GET")
	}

	// Check auto-generated name
	if config.Scenarios["test2"].Requests[0].Name == "" {
		t.Error("Request name should be auto-generated")
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Duration
		wantErr  bool
	}{
		{
			name:     "quoted duration",
			input:    `"30s"`,
			expected: Duration(30 * time.Second),
		},
		{
			name:     "unquoted null",
			input:    `null`,
			expected: Duration(0),
		},
		{
			name:     "quoted empty",
			input:    `""`,
			expected: Duration(0),
		},
		{
			name:    "invalid duration",
			input:   `"invalid"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if d != tt.expected {
				t.Errorf("UnmarshalJSON() = %v, want %v", d, tt.expected)
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	d := Duration(90 * time.Second)
	got, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	expected := `"1m30s"`
	if string(got) != expected {
		t.Errorf("MarshalJSON() = %v, want %v", string(got), expected)
	}
}
