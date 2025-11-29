package config

import (
	"strings"
	"testing"
)

func TestValidate_MinimalValid(t *testing.T) {
	config := &TestConfig{
		Name: "Test",
		Scenarios: map[string]*ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      10,
				Duration: "30s",
				Requests: []RequestConfig{
					{Method: "GET", URL: "/api/test"},
				},
			},
		},
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Validate() returned error for valid config: %v", err)
	}
}

func TestValidate_NoScenarios(t *testing.T) {
	config := &TestConfig{
		Name:      "Test",
		Scenarios: map[string]*ScenarioConfig{},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should return error when no scenarios defined")
	}

	if !strings.Contains(err.Error(), "scenario") {
		t.Errorf("Error should mention 'scenario', got: %v", err)
	}
}

func TestValidate_ConstantVUs(t *testing.T) {
	tests := []struct {
		name    string
		config  *ScenarioConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			config: &ScenarioConfig{
				Executor: "constant-vus",
				VUs:      10,
				Duration: "30s",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: false,
		},
		{
			name: "missing VUs",
			config: &ScenarioConfig{
				Executor: "constant-vus",
				VUs:      0,
				Duration: "30s",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "vus",
		},
		{
			name: "missing duration",
			config: &ScenarioConfig{
				Executor: "constant-vus",
				VUs:      10,
				Duration: "",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "duration",
		},
		{
			name: "invalid duration",
			config: &ScenarioConfig{
				Executor: "constant-vus",
				VUs:      10,
				Duration: "invalid",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name:      "Test",
				Scenarios: map[string]*ScenarioConfig{"test": tt.config},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_RampingVUs(t *testing.T) {
	tests := []struct {
		name    string
		config  *ScenarioConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			config: &ScenarioConfig{
				Executor: "ramping-vus",
				Stages: []StageConfig{
					{Duration: "30s", Target: 10},
					{Duration: "1m", Target: 10},
					{Duration: "30s", Target: 0},
				},
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: false,
		},
		{
			name: "no stages",
			config: &ScenarioConfig{
				Executor: "ramping-vus",
				Stages:   []StageConfig{},
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "stage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name:      "Test",
				Scenarios: map[string]*ScenarioConfig{"test": tt.config},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_ConstantArrivalRate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ScenarioConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			config: &ScenarioConfig{
				Executor:        "constant-arrival-rate",
				Rate:            100,
				Duration:        "1m",
				PreAllocatedVUs: 10,
				MaxVUs:          50,
				Requests:        []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: false,
		},
		{
			name: "missing rate",
			config: &ScenarioConfig{
				Executor: "constant-arrival-rate",
				Rate:     0,
				Duration: "1m",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "rate",
		},
		{
			name: "missing duration",
			config: &ScenarioConfig{
				Executor: "constant-arrival-rate",
				Rate:     100,
				Duration: "",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "duration",
		},
		{
			name: "preAllocatedVUs > maxVUs",
			config: &ScenarioConfig{
				Executor:        "constant-arrival-rate",
				Rate:            100,
				Duration:        "1m",
				PreAllocatedVUs: 100,
				MaxVUs:          50,
				Requests:        []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "preallocatedvus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name:      "Test",
				Scenarios: map[string]*ScenarioConfig{"test": tt.config},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_RampingArrivalRate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ScenarioConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			config: &ScenarioConfig{
				Executor:        "ramping-arrival-rate",
				PreAllocatedVUs: 10,
				MaxVUs:          100,
				Stages: []StageConfig{
					{Duration: "1m", Target: 50},
					{Duration: "2m", Target: 50},
					{Duration: "1m", Target: 0},
				},
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: false,
		},
		{
			name: "no stages",
			config: &ScenarioConfig{
				Executor:        "ramping-arrival-rate",
				PreAllocatedVUs: 10,
				MaxVUs:          100,
				Stages:          []StageConfig{},
				Requests:        []RequestConfig{{Method: "GET", URL: "/test"}},
			},
			wantErr: true,
			errMsg:  "stage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name:      "Test",
				Scenarios: map[string]*ScenarioConfig{"test": tt.config},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_InvalidExecutor(t *testing.T) {
	config := &TestConfig{
		Name: "Test",
		Scenarios: map[string]*ScenarioConfig{
			"test": {
				Executor: "unknown-executor",
				VUs:      10,
				Duration: "30s",
				Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should return error for unknown executor")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "executor") {
		t.Errorf("Error should mention 'executor', got: %v", err)
	}
}

func TestValidate_Requests(t *testing.T) {
	tests := []struct {
		name    string
		request RequestConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid GET",
			request: RequestConfig{Method: "GET", URL: "http://example.com/api"},
			wantErr: false,
		},
		{
			name:    "valid POST",
			request: RequestConfig{Method: "POST", URL: "/api/users", Body: `{"name":"test"}`},
			wantErr: false,
		},
		{
			name:    "missing method",
			request: RequestConfig{Method: "", URL: "/test"},
			wantErr: true,
			errMsg:  "method",
		},
		{
			name:    "invalid method",
			request: RequestConfig{Method: "INVALID", URL: "/test"},
			wantErr: true,
			errMsg:  "method",
		},
		{
			name:    "missing URL",
			request: RequestConfig{Method: "GET", URL: ""},
			wantErr: true,
			errMsg:  "url",
		},
		{
			name:    "URL with variables",
			request: RequestConfig{Method: "GET", URL: "{{baseUrl}}/api/users"},
			wantErr: false, // Variables are allowed
		},
		{
			name:    "invalid timeout",
			request: RequestConfig{Method: "GET", URL: "/test", Timeout: "invalid"},
			wantErr: true,
			errMsg:  "timeout",
		},
		{
			name:    "valid timeout",
			request: RequestConfig{Method: "GET", URL: "/test", Timeout: "30s"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name: "Test",
				Scenarios: map[string]*ScenarioConfig{
					"test": {
						Executor: "constant-vus",
						VUs:      10,
						Duration: "30s",
						Requests: []RequestConfig{tt.request},
					},
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_Pacing(t *testing.T) {
	tests := []struct {
		name    string
		pacing  *PacingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid none",
			pacing:  &PacingConfig{Type: "none"},
			wantErr: false,
		},
		{
			name:    "valid constant",
			pacing:  &PacingConfig{Type: "constant", Duration: "100ms"},
			wantErr: false,
		},
		{
			name:    "valid random",
			pacing:  &PacingConfig{Type: "random", Min: "50ms", Max: "100ms"},
			wantErr: false,
		},
		{
			name:    "invalid type",
			pacing:  &PacingConfig{Type: "invalid"},
			wantErr: true,
			errMsg:  "type",
		},
		{
			name:    "constant missing duration",
			pacing:  &PacingConfig{Type: "constant"},
			wantErr: true,
			errMsg:  "duration",
		},
		{
			name:    "random missing min",
			pacing:  &PacingConfig{Type: "random", Max: "100ms"},
			wantErr: true,
			errMsg:  "min",
		},
		{
			name:    "random missing max",
			pacing:  &PacingConfig{Type: "random", Min: "50ms"},
			wantErr: true,
			errMsg:  "max",
		},
		{
			name:    "random min > max",
			pacing:  &PacingConfig{Type: "random", Min: "200ms", Max: "100ms"},
			wantErr: true,
			errMsg:  "min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name: "Test",
				Scenarios: map[string]*ScenarioConfig{
					"test": {
						Executor: "constant-vus",
						VUs:      10,
						Duration: "30s",
						Pacing:   tt.pacing,
						Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
					},
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_Stages(t *testing.T) {
	tests := []struct {
		name    string
		stage   StageConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid",
			stage:   StageConfig{Duration: "30s", Target: 10},
			wantErr: false,
		},
		{
			name:    "missing duration",
			stage:   StageConfig{Duration: "", Target: 10},
			wantErr: true,
			errMsg:  "duration",
		},
		{
			name:    "invalid duration",
			stage:   StageConfig{Duration: "invalid", Target: 10},
			wantErr: true,
			errMsg:  "duration",
		},
		{
			name:    "negative target",
			stage:   StageConfig{Duration: "30s", Target: -1},
			wantErr: true,
			errMsg:  "target",
		},
		{
			name:    "zero target allowed",
			stage:   StageConfig{Duration: "30s", Target: 0},
			wantErr: false, // 0 is valid (for ramp-down)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name: "Test",
				Scenarios: map[string]*ScenarioConfig{
					"test": {
						Executor: "ramping-vus",
						Stages:   []StageConfig{tt.stage},
						Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
					},
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_Thresholds(t *testing.T) {
	tests := []struct {
		name       string
		thresholds *ThresholdsConfig
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid duration threshold",
			thresholds: &ThresholdsConfig{
				HTTPReqDuration: []string{"p95 < 500ms"},
			},
			wantErr: false,
		},
		{
			name: "valid failure rate threshold",
			thresholds: &ThresholdsConfig{
				HTTPReqFailed: []string{"rate < 0.01"},
			},
			wantErr: false,
		},
		{
			name: "valid request count threshold",
			thresholds: &ThresholdsConfig{
				HTTPReqs: []string{"count > 1000"},
			},
			wantErr: false,
		},
		{
			name: "multiple thresholds",
			thresholds: &ThresholdsConfig{
				HTTPReqDuration: []string{"p95 < 500ms", "avg < 200ms"},
				HTTPReqFailed:   []string{"rate < 0.01"},
			},
			wantErr: false,
		},
		{
			name: "empty threshold",
			thresholds: &ThresholdsConfig{
				HTTPReqDuration: []string{""},
			},
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name: "invalid threshold format",
			thresholds: &ThresholdsConfig{
				HTTPReqDuration: []string{"invalid threshold"},
			},
			wantErr: true,
			errMsg:  "metric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name: "Test",
				Scenarios: map[string]*ScenarioConfig{
					"test": {
						Executor: "constant-vus",
						VUs:      10,
						Duration: "30s",
						Requests: []RequestConfig{{Method: "GET", URL: "/test"}},
					},
				},
				Thresholds: tt.thresholds,
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_NoRequests(t *testing.T) {
	config := &TestConfig{
		Name: "Test",
		Scenarios: map[string]*ScenarioConfig{
			"test": {
				Executor: "constant-vus",
				VUs:      10,
				Duration: "30s",
				Requests: []RequestConfig{},
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should return error when no requests defined")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "request") {
		t.Errorf("Error should mention 'request', got: %v", err)
	}
}

func TestValidate_Extract(t *testing.T) {
	tests := []struct {
		name    string
		extract ExtractConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid body extract",
			extract: ExtractConfig{Name: "userId", Source: "body", Path: "$.data.id"},
			wantErr: false,
		},
		{
			name:    "valid header extract",
			extract: ExtractConfig{Name: "token", Source: "header", Path: "Authorization"},
			wantErr: false,
		},
		{
			name:    "valid status extract",
			extract: ExtractConfig{Name: "status", Source: "status"},
			wantErr: false,
		},
		{
			name:    "missing name",
			extract: ExtractConfig{Name: "", Source: "body"},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name:    "missing source",
			extract: ExtractConfig{Name: "test", Source: ""},
			wantErr: true,
			errMsg:  "source",
		},
		{
			name:    "invalid source",
			extract: ExtractConfig{Name: "test", Source: "invalid"},
			wantErr: true,
			errMsg:  "source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{
				Name: "Test",
				Scenarios: map[string]*ScenarioConfig{
					"test": {
						Executor: "constant-vus",
						VUs:      10,
						Duration: "30s",
						Requests: []RequestConfig{
							{
								Method:  "GET",
								URL:     "/test",
								Extract: []ExtractConfig{tt.extract},
							},
						},
					},
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	errs := &ValidationErrors{}

	if errs.HasErrors() {
		t.Error("Empty ValidationErrors should not have errors")
	}

	errs.Add("field1", "message1")
	errs.Add("field2", "message2")

	if !errs.HasErrors() {
		t.Error("ValidationErrors with errors should have errors")
	}

	if len(errs.Errors) != 2 {
		t.Errorf("len(Errors) = %v, want %v", len(errs.Errors), 2)
	}

	errStr := errs.Error()
	if !strings.Contains(errStr, "field1") || !strings.Contains(errStr, "field2") {
		t.Errorf("Error string should contain all fields, got: %v", errStr)
	}

	if !strings.Contains(errStr, "2 validation errors") {
		t.Errorf("Error string should mention count, got: %v", errStr)
	}
}

func TestValidationError_Single(t *testing.T) {
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "testField") {
		t.Errorf("Error should contain field name, got: %v", errStr)
	}

	if !strings.Contains(errStr, "test message") {
		t.Errorf("Error should contain message, got: %v", errStr)
	}
}
