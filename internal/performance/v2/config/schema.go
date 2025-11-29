// Package config provides configuration parsing and validation for the v2 performance engine.
package config

import (
	"time"
)

// TestConfig is the root configuration for a performance test.
//
// Example YAML:
//
//	name: "API Load Test"
//	settings:
//	  baseUrl: "https://api.example.com"
//	  timeout: 30s
//	scenarios:
//	  browse:
//	    executor: constant-vus
//	    vus: 10
//	    duration: 30s
//	    requests:
//	      - name: "Get Users"
//	        method: GET
//	        url: "{{baseUrl}}/api/users"
type TestConfig struct {
	// Name of the test (for reporting)
	Name string `json:"name" yaml:"name"`

	// Description of the test (optional)
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Settings contains global settings for all scenarios
	Settings GlobalSettings `json:"settings,omitempty" yaml:"settings,omitempty"`

	// Variables are global variables available to all scenarios
	Variables map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`

	// Scenarios defines the load profiles to run
	// Each scenario runs independently with its own executor
	Scenarios map[string]*ScenarioConfig `json:"scenarios" yaml:"scenarios"`

	// Thresholds define pass/fail criteria for metrics
	Thresholds *ThresholdsConfig `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`

	// Options for test execution
	Options *ExecutionOptions `json:"options,omitempty" yaml:"options,omitempty"`
}

// GlobalSettings contains global HTTP and execution settings.
type GlobalSettings struct {
	// BaseURL is the default base URL for all requests
	BaseURL string `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`

	// Timeout is the default HTTP request timeout
	Timeout Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// MaxConnectionsPerHost limits connections per host
	MaxConnectionsPerHost int `json:"maxConnectionsPerHost,omitempty" yaml:"maxConnectionsPerHost,omitempty"`

	// MaxIdleConnsPerHost limits idle connections per host
	MaxIdleConnsPerHost int `json:"maxIdleConnsPerHost,omitempty" yaml:"maxIdleConnsPerHost,omitempty"`

	// InsecureSkipVerify skips TLS certificate verification
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`

	// UserAgent is the default User-Agent header
	UserAgent string `json:"userAgent,omitempty" yaml:"userAgent,omitempty"`

	// Headers are default headers applied to all requests
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// ScenarioConfig defines a single load testing scenario.
type ScenarioConfig struct {
	// Executor specifies the load generation strategy
	// Options: "constant-vus", "ramping-vus", "constant-arrival-rate", "ramping-arrival-rate"
	Executor string `json:"executor" yaml:"executor"`

	// VUs is the number of virtual users (for VU-based executors)
	VUs int `json:"vus,omitempty" yaml:"vus,omitempty"`

	// Duration is how long to run (e.g., "30s", "2m", "1h")
	Duration string `json:"duration,omitempty" yaml:"duration,omitempty"`

	// Rate is iterations per second (for arrival-rate executors)
	Rate float64 `json:"rate,omitempty" yaml:"rate,omitempty"`

	// PreAllocatedVUs is the number of VUs to pre-allocate (for arrival-rate executors)
	PreAllocatedVUs int `json:"preAllocatedVUs,omitempty" yaml:"preAllocatedVUs,omitempty"`

	// MaxVUs is the maximum number of VUs to scale up to (for arrival-rate executors)
	MaxVUs int `json:"maxVUs,omitempty" yaml:"maxVUs,omitempty"`

	// Stages defines ramping stages (for ramping executors)
	Stages []StageConfig `json:"stages,omitempty" yaml:"stages,omitempty"`

	// Requests defines the HTTP requests to execute
	Requests []RequestConfig `json:"requests" yaml:"requests"`

	// GracefulStop is how long to wait for iterations to finish
	GracefulStop string `json:"gracefulStop,omitempty" yaml:"gracefulStop,omitempty"`

	// Pacing controls time between iterations
	Pacing *PacingConfig `json:"pacing,omitempty" yaml:"pacing,omitempty"`

	// StartTime specifies when this scenario should start (relative to test start)
	StartTime string `json:"startTime,omitempty" yaml:"startTime,omitempty"`

	// Tags are custom tags for this scenario's metrics
	Tags map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// StageConfig defines a single stage in a ramping executor.
type StageConfig struct {
	// Duration of this stage (e.g., "30s", "2m")
	Duration string `json:"duration" yaml:"duration"`

	// Target VU count (for ramping-vus) or RPS (for ramping-arrival-rate)
	Target int `json:"target" yaml:"target"`

	// Name is an optional name for this stage (for reporting)
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// RequestConfig defines a single HTTP request.
type RequestConfig struct {
	// Name for this request (used in metrics)
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string `json:"method" yaml:"method"`

	// URL is the request URL (supports variable substitution)
	URL string `json:"url" yaml:"url"`

	// Headers are request-specific headers
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`

	// Body is the request body (supports variable substitution)
	Body string `json:"body,omitempty" yaml:"body,omitempty"`

	// Timeout is request-specific timeout (overrides global)
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// ThinkTime is wait time after this request
	ThinkTime string `json:"thinkTime,omitempty" yaml:"thinkTime,omitempty"`

	// Extract defines variable extraction from response
	Extract []ExtractConfig `json:"extract,omitempty" yaml:"extract,omitempty"`

	// Assertions validate the response
	Assertions []AssertionConfig `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

// PacingConfig controls pacing between iterations.
type PacingConfig struct {
	// Type is the pacing strategy: "none", "constant", "random"
	Type string `json:"type" yaml:"type"`

	// Duration is the wait time for constant pacing
	Duration string `json:"duration,omitempty" yaml:"duration,omitempty"`

	// Min is the minimum wait time for random pacing
	Min string `json:"min,omitempty" yaml:"min,omitempty"`

	// Max is the maximum wait time for random pacing
	Max string `json:"max,omitempty" yaml:"max,omitempty"`
}

// ExtractConfig defines how to extract variables from a response.
type ExtractConfig struct {
	// Name of the variable to store
	Name string `json:"name" yaml:"name"`

	// Source is where to extract from: "body", "header", "status"
	Source string `json:"source" yaml:"source"`

	// Path is the header name, or JSONPath/XPath for body
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Regex is an optional regex pattern for extraction
	Regex string `json:"regex,omitempty" yaml:"regex,omitempty"`
}

// AssertionConfig defines a response validation.
type AssertionConfig struct {
	// Type is the assertion type: "status", "body", "header", "duration"
	Type string `json:"type" yaml:"type"`

	// Condition is the comparison: "eq", "ne", "gt", "lt", "gte", "lte", "contains", "matches"
	Condition string `json:"condition" yaml:"condition"`

	// Value is the expected value
	Value string `json:"value" yaml:"value"`

	// Path is for extracting a specific value (JSONPath for body, header name for header)
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Message is a custom error message on failure
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// ThresholdsConfig defines pass/fail criteria for the test.
type ThresholdsConfig struct {
	// HTTPReqDuration thresholds for request duration
	// e.g., ["p95 < 500ms", "avg < 200ms"]
	HTTPReqDuration []string `json:"http_req_duration,omitempty" yaml:"http_req_duration,omitempty"`

	// HTTPReqFailed thresholds for failure rate
	// e.g., ["rate < 0.01"] (less than 1% failures)
	HTTPReqFailed []string `json:"http_req_failed,omitempty" yaml:"http_req_failed,omitempty"`

	// HTTPReqs thresholds for request count/rate
	// e.g., ["count > 1000", "rate > 100"]
	HTTPReqs []string `json:"http_reqs,omitempty" yaml:"http_reqs,omitempty"`

	// Custom thresholds for scenario-specific metrics
	Custom map[string][]string `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// ExecutionOptions controls test execution behavior.
type ExecutionOptions struct {
	// Sequential runs scenarios one-by-one instead of parallel
	Sequential bool `json:"sequential,omitempty" yaml:"sequential,omitempty"`

	// IterationsTimeout is the maximum time to wait for iterations to complete
	IterationsTimeout string `json:"iterationsTimeout,omitempty" yaml:"iterationsTimeout,omitempty"`

	// SetupTimeout is the maximum time for setup operations
	SetupTimeout string `json:"setupTimeout,omitempty" yaml:"setupTimeout,omitempty"`

	// TeardownTimeout is the maximum time for teardown operations
	TeardownTimeout string `json:"teardownTimeout,omitempty" yaml:"teardownTimeout,omitempty"`

	// NoVUConnectionReuse disables HTTP connection reuse between VUs
	NoVUConnectionReuse bool `json:"noVUConnectionReuse,omitempty" yaml:"noVUConnectionReuse,omitempty"`
}

// Duration is a time.Duration that can be unmarshaled from JSON/YAML strings.
type Duration time.Duration

// ParseDuration parses a duration string (e.g., "30s", "2m", "1h30m").
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// GetDuration returns the duration or a default if empty.
func (d Duration) GetDuration(defaultValue time.Duration) time.Duration {
	if d == 0 {
		return defaultValue
	}
	return time.Duration(d)
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Remove quotes if present
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	if s == "" || s == "null" {
		*d = 0
		return nil
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	if s == "" {
		*d = 0
		return nil
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// String returns the duration as a string.
func (d Duration) String() string {
	return time.Duration(d).String()
}
