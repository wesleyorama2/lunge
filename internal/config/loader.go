package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config represents the top-level configuration
type Config struct {
	Environments map[string]Environment     `json:"environments"`
	Requests     map[string]Request         `json:"requests"`
	Suites       map[string]Suite           `json:"suites"`
	Schemas      map[string]json.RawMessage `json:"schemas,omitempty"`
	Performance  map[string]PerformanceTest `json:"performance,omitempty"`
}

// Environment represents an environment configuration
type Environment struct {
	BaseURL string            `json:"baseUrl"`
	Headers map[string]string `json:"headers,omitempty"`
	Vars    map[string]string `json:"variables,omitempty"`
}

// Request represents a request configuration
type Request struct {
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string]string      `json:"queryParams,omitempty"`
	Body        interface{}            `json:"body,omitempty"`
	Extract     map[string]string      `json:"extract,omitempty"`
	Validate    map[string]interface{} `json:"validate,omitempty"`
}

// Suite represents a suite of requests
type Suite struct {
	Requests []string          `json:"requests"`
	Vars     map[string]string `json:"variables,omitempty"`
	Tests    []Test            `json:"tests,omitempty"`
}

// Test represents a test configuration
type Test struct {
	Name       string                   `json:"name"`
	Request    string                   `json:"request"`
	Assertions []map[string]interface{} `json:"assertions"`
}

// PerformanceTest represents a performance test configuration
type PerformanceTest struct {
	Name       string                `json:"name"`
	Request    string                `json:"request"`
	Load       PerformanceLoadConfig `json:"load"`
	Monitoring MonitoringConfig      `json:"monitoring,omitempty"`
	Thresholds ThresholdConfig       `json:"thresholds,omitempty"`
	Reporting  ReportingConfig       `json:"reporting,omitempty"`
}

// PerformanceLoadConfig defines load generation parameters
type PerformanceLoadConfig struct {
	Concurrency int          `json:"concurrency"`
	Iterations  int          `json:"iterations,omitempty"`
	Duration    string       `json:"duration,omitempty"`
	RPS         float64      `json:"rps,omitempty"`
	RampUp      string       `json:"rampUp,omitempty"`
	RampDown    string       `json:"rampDown,omitempty"`
	Pattern     string       `json:"pattern,omitempty"`
	Warmup      WarmupConfig `json:"warmup,omitempty"`
}

// WarmupConfig defines warmup phase configuration
type WarmupConfig struct {
	Duration   string  `json:"duration,omitempty"`
	Iterations int     `json:"iterations,omitempty"`
	RPS        float64 `json:"rps,omitempty"`
}

// ThresholdConfig defines performance thresholds
type ThresholdConfig struct {
	MaxResponseTime string  `json:"maxResponseTime,omitempty"`
	MaxErrorRate    float64 `json:"maxErrorRate,omitempty"`
	MinThroughput   float64 `json:"minThroughput,omitempty"`
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	RealTime  bool   `json:"realTime,omitempty"`
	Interval  string `json:"interval,omitempty"`
	Resources bool   `json:"resources,omitempty"`
	Alerts    bool   `json:"alerts,omitempty"`
}

// ReportingConfig defines reporting configuration
type ReportingConfig struct {
	Format   string `json:"format,omitempty"`
	Output   string `json:"output,omitempty"`
	Template string `json:"template,omitempty"`
}

// LoadConfig loads a configuration file
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate performance configurations if present
	if err := ValidatePerformanceConfigurations(&config); err != nil {
		return nil, fmt.Errorf("performance configuration validation failed: %w", err)
	}

	return &config, nil
}

// ValidatePerformanceConfigurations validates all performance test configurations
func ValidatePerformanceConfigurations(config *Config) error {
	if config.Performance == nil {
		return nil // No performance tests to validate
	}

	for name, perfTest := range config.Performance {
		if err := ValidatePerformanceTest(&perfTest); err != nil {
			return fmt.Errorf("invalid performance test '%s': %w", name, err)
		}
	}

	return nil
}

// ValidatePerformanceTest validates a single performance test configuration
func ValidatePerformanceTest(perfTest *PerformanceTest) error {
	if perfTest == nil {
		return fmt.Errorf("performance test cannot be nil")
	}

	// Validate name
	if perfTest.Name == "" {
		return fmt.Errorf("performance test name cannot be empty")
	}

	// Validate request reference
	if perfTest.Request == "" {
		return fmt.Errorf("performance test must reference a request")
	}

	// Validate load configuration
	if err := ValidatePerformanceLoadConfig(&perfTest.Load); err != nil {
		return fmt.Errorf("invalid load configuration: %w", err)
	}

	// Validate thresholds
	if err := ValidatePerformanceThresholds(&perfTest.Thresholds); err != nil {
		return fmt.Errorf("invalid thresholds: %w", err)
	}

	// Validate monitoring
	if err := ValidatePerformanceMonitoring(&perfTest.Monitoring); err != nil {
		return fmt.Errorf("invalid monitoring configuration: %w", err)
	}

	// Validate reporting
	if err := ValidatePerformanceReporting(&perfTest.Reporting); err != nil {
		return fmt.Errorf("invalid reporting configuration: %w", err)
	}

	return nil
}

// ValidatePerformanceLoadConfig validates load configuration
func ValidatePerformanceLoadConfig(load *PerformanceLoadConfig) error {
	if load == nil {
		return fmt.Errorf("load configuration cannot be nil")
	}

	// Validate concurrency
	if load.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}
	if load.Concurrency > 1000 {
		return fmt.Errorf("concurrency cannot exceed 1000")
	}

	// Validate test duration parameters
	hasDuration := load.Duration != ""
	hasIterations := load.Iterations > 0

	if !hasDuration && !hasIterations {
		return fmt.Errorf("either iterations or duration must be specified")
	}

	if hasDuration && hasIterations {
		return fmt.Errorf("cannot specify both iterations and duration")
	}

	// Validate duration format
	if hasDuration {
		if _, err := parseDurationString(load.Duration); err != nil {
			return fmt.Errorf("invalid duration format '%s': %w", load.Duration, err)
		}
	}

	// Validate RPS
	if load.RPS < 0 {
		return fmt.Errorf("RPS cannot be negative")
	}

	// Validate ramp durations
	if load.RampUp != "" {
		if _, err := parseDurationString(load.RampUp); err != nil {
			return fmt.Errorf("invalid ramp-up duration '%s': %w", load.RampUp, err)
		}
	}

	if load.RampDown != "" {
		if _, err := parseDurationString(load.RampDown); err != nil {
			return fmt.Errorf("invalid ramp-down duration '%s': %w", load.RampDown, err)
		}
	}

	// Validate pattern
	if load.Pattern != "" {
		validPatterns := []string{"constant", "linear", "step"}
		if !stringInSlice(load.Pattern, validPatterns) {
			return fmt.Errorf("invalid pattern '%s', must be one of: %s", load.Pattern, strings.Join(validPatterns, ", "))
		}
	}

	// Validate warmup
	if err := ValidateWarmupConfig(&load.Warmup); err != nil {
		return fmt.Errorf("invalid warmup configuration: %w", err)
	}

	return nil
}

// ValidateWarmupConfig validates warmup configuration
func ValidateWarmupConfig(warmup *WarmupConfig) error {
	if warmup == nil {
		return nil // Warmup is optional
	}

	// Validate duration format if specified
	if warmup.Duration != "" {
		if _, err := parseDurationString(warmup.Duration); err != nil {
			return fmt.Errorf("invalid warmup duration '%s': %w", warmup.Duration, err)
		}
	}

	// Validate that at least one parameter is specified
	if warmup.Duration == "" && warmup.Iterations <= 0 {
		return fmt.Errorf("warmup must specify either duration or iterations")
	}

	// Validate RPS
	if warmup.RPS < 0 {
		return fmt.Errorf("warmup RPS cannot be negative")
	}

	return nil
}

// ValidatePerformanceThresholds validates threshold configuration
func ValidatePerformanceThresholds(thresholds *ThresholdConfig) error {
	if thresholds == nil {
		return nil // Thresholds are optional
	}

	// Validate max response time
	if thresholds.MaxResponseTime != "" {
		if _, err := parseDurationString(thresholds.MaxResponseTime); err != nil {
			return fmt.Errorf("invalid max response time '%s': %w", thresholds.MaxResponseTime, err)
		}
	}

	// Validate max error rate
	if thresholds.MaxErrorRate < 0 || thresholds.MaxErrorRate > 1 {
		return fmt.Errorf("max error rate must be between 0 and 1")
	}

	// Validate min throughput
	if thresholds.MinThroughput < 0 {
		return fmt.Errorf("min throughput cannot be negative")
	}

	return nil
}

// ValidatePerformanceMonitoring validates monitoring configuration
func ValidatePerformanceMonitoring(monitoring *MonitoringConfig) error {
	if monitoring == nil {
		return nil // Monitoring is optional
	}

	// Validate interval
	if monitoring.Interval != "" {
		if _, err := parseDurationString(monitoring.Interval); err != nil {
			return fmt.Errorf("invalid monitoring interval '%s': %w", monitoring.Interval, err)
		}
	}

	return nil
}

// ValidatePerformanceReporting validates reporting configuration
func ValidatePerformanceReporting(reporting *ReportingConfig) error {
	if reporting == nil {
		return nil // Reporting is optional
	}

	// Validate format
	if reporting.Format != "" {
		validFormats := []string{"text", "json", "html", "csv"}
		if !stringInSlice(reporting.Format, validFormats) {
			return fmt.Errorf("invalid report format '%s'", reporting.Format)
		}
	}

	return nil
}

// Helper functions

// parseDurationString parses duration strings like "30s", "5m", "1h"
func parseDurationString(duration string) (time.Duration, error) {
	// Handle common duration formats
	duration = strings.TrimSpace(duration)
	if duration == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}

	// Try parsing as Go duration
	if d, err := time.ParseDuration(duration); err == nil {
		return d, nil
	}

	// Handle additional formats like "1 minute", "30 seconds"
	duration = strings.ToLower(duration)
	duration = strings.ReplaceAll(duration, " ", "")

	// Convert common words to Go duration format
	replacements := map[string]string{
		"second":  "s",
		"seconds": "s",
		"minute":  "m",
		"minutes": "m",
		"hour":    "h",
		"hours":   "h",
	}

	for word, abbrev := range replacements {
		duration = strings.ReplaceAll(duration, word, abbrev)
	}

	return time.ParseDuration(duration)
}

// stringInSlice checks if a string is in a slice
func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// ProcessEnvironment processes environment variables in a string
func ProcessEnvironment(input string, env map[string]string) string {
	result := input

	// Replace environment variables
	for key, value := range env {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}

	return result
}

// ProcessEnvironmentInMap processes environment variables in a map
func ProcessEnvironmentInMap(input map[string]string, env map[string]string) map[string]string {
	result := make(map[string]string)

	for key, value := range input {
		result[key] = ProcessEnvironment(value, env)
	}

	return result
}

// MergeEnvironments merges two environments, with the second taking precedence
func MergeEnvironments(base, override map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy base environment
	for key, value := range base {
		result[key] = value
	}

	// Override with second environment
	for key, value := range override {
		result[key] = value
	}

	return result
}

// GetConfigDir returns the directory containing the config file
func GetConfigDir(configPath string) string {
	return filepath.Dir(configPath)
}
