package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"environments": {
			"dev": {
				"baseUrl": "https://api-dev.example.com",
				"variables": {
					"userId": "1"
				}
			},
			"prod": {
				"baseUrl": "https://api.example.com",
				"variables": {
					"userId": "2"
				}
			}
		},
		"requests": {
			"getUser": {
				"url": "/users/{{userId}}",
				"method": "GET",
				"headers": {
					"Accept": "application/json"
				}
			},
			"getPosts": {
				"url": "/posts",
				"method": "GET",
				"queryParams": {
					"userId": "{{userId}}"
				}
			}
		},
		"suites": {
			"userFlow": {
				"requests": ["getUser", "getPosts"],
				"variables": {
					"userId": "1"
				},
				"tests": [
					{
						"name": "User exists",
						"request": "getUser",
						"assertions": [
							{ "status": 200 },
							{ "path": "$.email", "exists": true }
						]
					}
				]
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Error creating test config file: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Error loading config: %v", err)
	}

	// Check environments
	if len(config.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(config.Environments))
	}

	devEnv, ok := config.Environments["dev"]
	if !ok {
		t.Errorf("Expected dev environment to exist")
	} else {
		if devEnv.BaseURL != "https://api-dev.example.com" {
			t.Errorf("Expected dev baseUrl to be https://api-dev.example.com, got %s", devEnv.BaseURL)
		}
		if devEnv.Vars["userId"] != "1" {
			t.Errorf("Expected dev userId to be 1, got %s", devEnv.Vars["userId"])
		}
	}

	// Check requests
	if len(config.Requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(config.Requests))
	}

	getUserReq, ok := config.Requests["getUser"]
	if !ok {
		t.Errorf("Expected getUser request to exist")
	} else {
		if getUserReq.URL != "/users/{{userId}}" {
			t.Errorf("Expected getUser URL to be /users/{{userId}}, got %s", getUserReq.URL)
		}
		if getUserReq.Method != "GET" {
			t.Errorf("Expected getUser method to be GET, got %s", getUserReq.Method)
		}
	}

	// Check suites
	if len(config.Suites) != 1 {
		t.Errorf("Expected 1 suite, got %d", len(config.Suites))
	}

	userFlowSuite, ok := config.Suites["userFlow"]
	if !ok {
		t.Errorf("Expected userFlow suite to exist")
	} else {
		if len(userFlowSuite.Requests) != 2 {
			t.Errorf("Expected userFlow to have 2 requests, got %d", len(userFlowSuite.Requests))
		}
		if userFlowSuite.Vars["userId"] != "1" {
			t.Errorf("Expected userFlow userId to be 1, got %s", userFlowSuite.Vars["userId"])
		}
		if len(userFlowSuite.Tests) != 1 {
			t.Errorf("Expected userFlow to have 1 test, got %d", len(userFlowSuite.Tests))
		}
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("non-existent-file.json")
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")

	invalidContent := `{ this is not valid json }`
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Error creating test file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got nil")
	}
}

func TestLoadConfig_WithPerformance(t *testing.T) {
	// Create a temporary config file with performance configuration
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"environments": {
			"dev": {
				"baseUrl": "https://api-dev.example.com"
			}
		},
		"requests": {
			"getUser": {
				"url": "/users/1",
				"method": "GET"
			}
		},
		"suites": {},
		"performance": {
			"loadTest": {
				"name": "Load Test",
				"request": "getUser",
				"load": {
					"concurrency": 10,
					"duration": "30s",
					"warmup": {
						"duration": "5s"
					}
				}
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Error creating test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Error loading config: %v", err)
	}

	if len(config.Performance) != 1 {
		t.Errorf("Expected 1 performance test, got %d", len(config.Performance))
	}
}

func TestLoadConfig_InvalidPerformance(t *testing.T) {
	// Create a temporary config file with invalid performance configuration
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"environments": {
			"dev": {
				"baseUrl": "https://api-dev.example.com"
			}
		},
		"requests": {
			"getUser": {
				"url": "/users/1",
				"method": "GET"
			}
		},
		"suites": {},
		"performance": {
			"loadTest": {
				"name": "Load Test",
				"request": "getUser",
				"load": {
					"concurrency": 0
				}
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Error creating test config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Errorf("Expected error for invalid performance config, got nil")
	}
}

func TestProcessEnvironment(t *testing.T) {
	env := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "123",
		"token":   "abc123",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No variables",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "Single variable",
			input:    "{{baseUrl}}/users",
			expected: "https://api.example.com/users",
		},
		{
			name:     "Multiple variables",
			input:    "{{baseUrl}}/users/{{userId}}?token={{token}}",
			expected: "https://api.example.com/users/123?token=abc123",
		},
		{
			name:     "Unknown variable",
			input:    "{{baseUrl}}/users/{{unknown}}",
			expected: "https://api.example.com/users/{{unknown}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessEnvironment(tt.input, env)
			if result != tt.expected {
				t.Errorf("ProcessEnvironment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestProcessEnvironmentInMap(t *testing.T) {
	env := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "123",
	}

	input := map[string]string{
		"url":   "{{baseUrl}}/users/{{userId}}",
		"token": "Bearer {{userId}}-token",
		"plain": "No variables here",
	}

	expected := map[string]string{
		"url":   "https://api.example.com/users/123",
		"token": "Bearer 123-token",
		"plain": "No variables here",
	}

	result := ProcessEnvironmentInMap(input, env)

	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("ProcessEnvironmentInMap()[%s] = %v, want %v", key, result[key], expectedValue)
		}
	}
}

func TestMergeEnvironments(t *testing.T) {
	base := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "123",
		"token":   "abc",
	}

	override := map[string]string{
		"userId": "456",
		"newVar": "xyz",
	}

	expected := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "456",
		"token":   "abc",
		"newVar":  "xyz",
	}

	result := MergeEnvironments(base, override)

	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("MergeEnvironments()[%s] = %v, want %v", key, result[key], expectedValue)
		}
	}
}

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		configPath string
		expected   string
	}{
		{
			configPath: "/path/to/config.json",
			expected:   "/path/to",
		},
		{
			configPath: "config.json",
			expected:   ".",
		},
		{
			configPath: "./config.json",
			expected:   ".",
		},
		{
			configPath: "../config.json",
			expected:   "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.configPath, func(t *testing.T) {
			result := GetConfigDir(tt.configPath)
			// Normalize path separators for cross-platform compatibility
			result = filepath.ToSlash(result)
			if result != tt.expected {
				t.Errorf("GetConfigDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test ValidatePerformanceConfigurations
func TestValidatePerformanceConfigurations(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil performance",
			config:      &Config{Performance: nil},
			expectError: false,
		},
		{
			name: "valid performance",
			config: &Config{
				Performance: map[string]PerformanceTest{
					"test1": {
						Name:    "Test 1",
						Request: "getUser",
						Load: PerformanceLoadConfig{
							Concurrency: 10,
							Duration:    "30s",
							Warmup: WarmupConfig{
								Duration: "5s",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid performance - missing name",
			config: &Config{
				Performance: map[string]PerformanceTest{
					"test1": {
						Name:    "",
						Request: "getUser",
						Load: PerformanceLoadConfig{
							Concurrency: 10,
							Duration:    "30s",
							Warmup: WarmupConfig{
								Duration: "5s",
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePerformanceConfigurations(tt.config)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test ValidatePerformanceTest
func TestValidatePerformanceTest(t *testing.T) {
	tests := []struct {
		name        string
		perfTest    *PerformanceTest
		expectError bool
	}{
		{
			name:        "nil performance test",
			perfTest:    nil,
			expectError: true,
		},
		{
			name: "valid performance test",
			perfTest: &PerformanceTest{
				Name:    "Load Test",
				Request: "getUser",
				Load: PerformanceLoadConfig{
					Concurrency: 10,
					Duration:    "30s",
					Warmup: WarmupConfig{
						Duration: "5s",
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty name",
			perfTest: &PerformanceTest{
				Name:    "",
				Request: "getUser",
				Load: PerformanceLoadConfig{
					Concurrency: 10,
					Duration:    "30s",
					Warmup: WarmupConfig{
						Duration: "5s",
					},
				},
			},
			expectError: true,
		},
		{
			name: "empty request",
			perfTest: &PerformanceTest{
				Name:    "Load Test",
				Request: "",
				Load: PerformanceLoadConfig{
					Concurrency: 10,
					Duration:    "30s",
					Warmup: WarmupConfig{
						Duration: "5s",
					},
				},
			},
			expectError: true,
		},
		{
			name: "with valid thresholds",
			perfTest: &PerformanceTest{
				Name:    "Load Test",
				Request: "getUser",
				Load: PerformanceLoadConfig{
					Concurrency: 10,
					Duration:    "30s",
					Warmup: WarmupConfig{
						Duration: "5s",
					},
				},
				Thresholds: ThresholdConfig{
					MaxResponseTime: "1s",
					MaxErrorRate:    0.05,
					MinThroughput:   100,
				},
			},
			expectError: false,
		},
		{
			name: "with valid monitoring",
			perfTest: &PerformanceTest{
				Name:    "Load Test",
				Request: "getUser",
				Load: PerformanceLoadConfig{
					Concurrency: 10,
					Duration:    "30s",
					Warmup: WarmupConfig{
						Duration: "5s",
					},
				},
				Monitoring: MonitoringConfig{
					RealTime: true,
					Interval: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "with valid reporting",
			perfTest: &PerformanceTest{
				Name:    "Load Test",
				Request: "getUser",
				Load: PerformanceLoadConfig{
					Concurrency: 10,
					Duration:    "30s",
					Warmup: WarmupConfig{
						Duration: "5s",
					},
				},
				Reporting: ReportingConfig{
					Format: "json",
					Output: "report.json",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePerformanceTest(tt.perfTest)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test ValidatePerformanceLoadConfig
func TestValidatePerformanceLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		load        *PerformanceLoadConfig
		expectError bool
	}{
		{
			name:        "nil load config",
			load:        nil,
			expectError: true,
		},
		{
			name: "valid with duration",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "valid with iterations",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Iterations:  100,
				Warmup: WarmupConfig{
					Iterations: 5,
				},
			},
			expectError: false,
		},
		{
			name: "concurrency too low",
			load: &PerformanceLoadConfig{
				Concurrency: 0,
				Duration:    "30s",
			},
			expectError: true,
		},
		{
			name: "concurrency too high",
			load: &PerformanceLoadConfig{
				Concurrency: 1001,
				Duration:    "30s",
			},
			expectError: true,
		},
		{
			name: "no duration or iterations",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
			},
			expectError: true,
		},
		{
			name: "both duration and iterations",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Iterations:  100,
			},
			expectError: true,
		},
		{
			name: "invalid duration format",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "invalid",
			},
			expectError: true,
		},
		{
			name: "negative RPS",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				RPS:         -1,
			},
			expectError: true,
		},
		{
			name: "valid RPS",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				RPS:         100,
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "valid ramp-up",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				RampUp:      "10s",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "invalid ramp-up",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				RampUp:      "invalid",
			},
			expectError: true,
		},
		{
			name: "valid ramp-down",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				RampDown:    "5s",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "invalid ramp-down",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				RampDown:    "invalid",
			},
			expectError: true,
		},
		{
			name: "valid pattern - constant",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Pattern:     "constant",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "valid pattern - linear",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Pattern:     "linear",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "valid pattern - step",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Pattern:     "step",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
		{
			name: "invalid pattern",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Pattern:     "invalid",
			},
			expectError: true,
		},
		{
			name: "valid warmup",
			load: &PerformanceLoadConfig{
				Concurrency: 10,
				Duration:    "30s",
				Warmup: WarmupConfig{
					Duration: "5s",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePerformanceLoadConfig(tt.load)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test ValidateWarmupConfig
func TestValidateWarmupConfig(t *testing.T) {
	tests := []struct {
		name        string
		warmup      *WarmupConfig
		expectError bool
	}{
		{
			name:        "nil warmup",
			warmup:      nil,
			expectError: false,
		},
		{
			name: "valid with duration",
			warmup: &WarmupConfig{
				Duration: "10s",
			},
			expectError: false,
		},
		{
			name: "valid with iterations",
			warmup: &WarmupConfig{
				Iterations: 10,
			},
			expectError: false,
		},
		{
			name: "invalid duration format",
			warmup: &WarmupConfig{
				Duration: "invalid",
			},
			expectError: true,
		},
		{
			name: "no duration or iterations",
			warmup: &WarmupConfig{
				RPS: 10,
			},
			expectError: true,
		},
		{
			name: "negative RPS",
			warmup: &WarmupConfig{
				Duration: "10s",
				RPS:      -1,
			},
			expectError: true,
		},
		{
			name: "valid RPS",
			warmup: &WarmupConfig{
				Duration: "10s",
				RPS:      50,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWarmupConfig(tt.warmup)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test ValidatePerformanceThresholds
func TestValidatePerformanceThresholds(t *testing.T) {
	tests := []struct {
		name        string
		thresholds  *ThresholdConfig
		expectError bool
	}{
		{
			name:        "nil thresholds",
			thresholds:  nil,
			expectError: false,
		},
		{
			name: "valid thresholds",
			thresholds: &ThresholdConfig{
				MaxResponseTime: "1s",
				MaxErrorRate:    0.05,
				MinThroughput:   100,
			},
			expectError: false,
		},
		{
			name: "invalid max response time",
			thresholds: &ThresholdConfig{
				MaxResponseTime: "invalid",
			},
			expectError: true,
		},
		{
			name: "negative error rate",
			thresholds: &ThresholdConfig{
				MaxErrorRate: -0.1,
			},
			expectError: true,
		},
		{
			name: "error rate over 1",
			thresholds: &ThresholdConfig{
				MaxErrorRate: 1.5,
			},
			expectError: true,
		},
		{
			name: "negative throughput",
			thresholds: &ThresholdConfig{
				MinThroughput: -10,
			},
			expectError: true,
		},
		{
			name: "zero error rate - valid",
			thresholds: &ThresholdConfig{
				MaxErrorRate: 0,
			},
			expectError: false,
		},
		{
			name: "max error rate 1 - valid",
			thresholds: &ThresholdConfig{
				MaxErrorRate: 1.0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePerformanceThresholds(tt.thresholds)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test ValidatePerformanceMonitoring
func TestValidatePerformanceMonitoring(t *testing.T) {
	tests := []struct {
		name        string
		monitoring  *MonitoringConfig
		expectError bool
	}{
		{
			name:        "nil monitoring",
			monitoring:  nil,
			expectError: false,
		},
		{
			name: "valid monitoring",
			monitoring: &MonitoringConfig{
				RealTime:  true,
				Interval:  "5s",
				Resources: true,
				Alerts:    true,
			},
			expectError: false,
		},
		{
			name: "invalid interval",
			monitoring: &MonitoringConfig{
				Interval: "invalid",
			},
			expectError: true,
		},
		{
			name: "empty interval - valid",
			monitoring: &MonitoringConfig{
				RealTime: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePerformanceMonitoring(tt.monitoring)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test ValidatePerformanceReporting
func TestValidatePerformanceReporting(t *testing.T) {
	tests := []struct {
		name        string
		reporting   *ReportingConfig
		expectError bool
	}{
		{
			name:        "nil reporting",
			reporting:   nil,
			expectError: false,
		},
		{
			name: "valid text format",
			reporting: &ReportingConfig{
				Format: "text",
			},
			expectError: false,
		},
		{
			name: "valid json format",
			reporting: &ReportingConfig{
				Format: "json",
			},
			expectError: false,
		},
		{
			name: "valid html format",
			reporting: &ReportingConfig{
				Format: "html",
			},
			expectError: false,
		},
		{
			name: "valid csv format",
			reporting: &ReportingConfig{
				Format: "csv",
			},
			expectError: false,
		},
		{
			name: "invalid format",
			reporting: &ReportingConfig{
				Format: "invalid",
			},
			expectError: true,
		},
		{
			name: "empty format - valid",
			reporting: &ReportingConfig{
				Output: "report.txt",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePerformanceReporting(tt.reporting)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test parseDurationString
func TestParseDurationString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{
			name:        "seconds shorthand",
			input:       "30s",
			expected:    30 * time.Second,
			expectError: false,
		},
		{
			name:        "minutes shorthand",
			input:       "5m",
			expected:    5 * time.Minute,
			expectError: false,
		},
		{
			name:        "hours shorthand",
			input:       "1h",
			expected:    1 * time.Hour,
			expectError: false,
		},
		{
			name:        "combined duration",
			input:       "1h30m",
			expected:    90 * time.Minute,
			expectError: false,
		},
		{
			name:        "milliseconds",
			input:       "100ms",
			expected:    100 * time.Millisecond,
			expectError: false,
		},
		{
			name:        "microseconds",
			input:       "500us",
			expected:    500 * time.Microsecond,
			expectError: false,
		},
		{
			name:        "nanoseconds",
			input:       "1000ns",
			expected:    1000 * time.Nanosecond,
			expectError: false,
		},
		{
			name:        "complex duration",
			input:       "2h30m15s",
			expected:    2*time.Hour + 30*time.Minute + 15*time.Second,
			expectError: false,
		},
		{
			name:        "with spaces",
			input:       " 30s ",
			expected:    30 * time.Second,
			expectError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDurationString(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}

// Test stringInSlice
func TestStringInSlice(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		slice    []string
		expected bool
	}{
		{
			name:     "string in slice",
			str:      "apple",
			slice:    []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "string not in slice",
			str:      "orange",
			slice:    []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "empty slice",
			str:      "apple",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "empty string",
			str:      "",
			slice:    []string{"apple", "banana", ""},
			expected: true,
		},
		{
			name:     "case sensitive",
			str:      "Apple",
			slice:    []string{"apple", "banana", "cherry"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringInSlice(tt.str, tt.slice)
			if result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}
