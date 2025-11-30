package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads a test configuration from a file.
//
// The file format is determined by extension:
//   - .yaml, .yml -> YAML
//   - .json -> JSON
//
// Returns the parsed TestConfig or an error if parsing fails.
func LoadConfig(path string) (*TestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return ParseConfig(data, path)
}

// ParseConfig parses configuration data.
//
// The format is determined by the file extension in path, or defaults to YAML
// if the path is empty or has an unknown extension.
func ParseConfig(data []byte, path string) (*TestConfig, error) {
	var config TestConfig

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml", "":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		// Try YAML by default
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config (unknown format %s): %w", ext, err)
		}
	}

	return &config, nil
}

// ParseDurationString parses a duration string with support for common formats.
//
// Supported formats:
//   - Standard Go duration: "30s", "2m", "1h30m", "500ms"
//   - Seconds as integer: "30" (treated as 30 seconds)
//
// Returns the parsed duration or an error.
func ParseDurationString(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Try standard Go duration parsing first
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	// Try parsing as integer seconds
	var seconds int
	if _, err := fmt.Sscanf(s, "%d", &seconds); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s", s)
}

// ParseScenarioDuration parses the duration for a scenario config.
//
// For stage-based executors, if no explicit duration is set,
// the total duration is calculated from all stages.
func ParseScenarioDuration(sc *ScenarioConfig) (time.Duration, error) {
	// If duration is explicitly set, use it
	if sc.Duration != "" {
		return ParseDurationString(sc.Duration)
	}

	// For stage-based executors, sum up stage durations
	if len(sc.Stages) > 0 {
		var total time.Duration
		for _, stage := range sc.Stages {
			stageDur, err := ParseDurationString(stage.Duration)
			if err != nil {
				return 0, fmt.Errorf("invalid stage duration: %w", err)
			}
			total += stageDur
		}
		return total, nil
	}

	return 0, fmt.Errorf("no duration specified and no stages defined")
}

// ResolveVariables resolves variable placeholders in a string.
//
// Variables are specified as {{varName}} and are resolved from:
//  1. Scenario-specific variables
//  2. Global test variables
//  3. Default settings (baseUrl, etc.)
//
// Unresolved variables are left as-is.
func ResolveVariables(input string, globals map[string]string, settings *GlobalSettings) string {
	result := input

	// Resolve from globals first
	for key, value := range globals {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Then resolve special settings
	if settings != nil {
		if settings.BaseURL != "" {
			result = strings.ReplaceAll(result, "{{baseUrl}}", settings.BaseURL)
			result = strings.ReplaceAll(result, "{{baseURL}}", settings.BaseURL)
		}
	}

	return result
}

// MergeVariables merges multiple variable maps in order.
// Later maps override earlier ones.
func MergeVariables(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// ApplyDefaults applies default values to a TestConfig.
func ApplyDefaults(config *TestConfig) {
	// Default settings
	if config.Settings.Timeout == 0 {
		config.Settings.Timeout = Duration(30 * time.Second)
	}
	if config.Settings.MaxConnectionsPerHost == 0 {
		config.Settings.MaxConnectionsPerHost = 100
	}
	if config.Settings.MaxIdleConnsPerHost == 0 {
		config.Settings.MaxIdleConnsPerHost = 100
	}
	if config.Settings.UserAgent == "" {
		config.Settings.UserAgent = "lunge/2.0"
	}

	// Default options
	if config.Options == nil {
		config.Options = &ExecutionOptions{}
	}

	// Apply defaults to each scenario
	for name, sc := range config.Scenarios {
		applyScenarioDefaults(name, sc)
	}
}

// applyScenarioDefaults applies default values to a scenario.
func applyScenarioDefaults(name string, sc *ScenarioConfig) {
	// Default executor
	if sc.Executor == "" {
		sc.Executor = "constant-vus"
	}

	// Defaults based on executor type
	switch sc.Executor {
	case "constant-vus":
		if sc.VUs == 0 {
			sc.VUs = 1
		}
	case "ramping-vus":
		// VUs determined by stages
	case "constant-arrival-rate":
		if sc.Rate == 0 {
			sc.Rate = 1
		}
		if sc.PreAllocatedVUs == 0 {
			sc.PreAllocatedVUs = 1
		}
		if sc.MaxVUs == 0 {
			sc.MaxVUs = sc.PreAllocatedVUs * 10
		}
	case "ramping-arrival-rate":
		if sc.PreAllocatedVUs == 0 {
			sc.PreAllocatedVUs = 1
		}
		if sc.MaxVUs == 0 {
			sc.MaxVUs = 100
		}
	}

	// Default request names
	for i, req := range sc.Requests {
		if req.Name == "" {
			sc.Requests[i].Name = fmt.Sprintf("%s_request_%d", name, i+1)
		}
		if req.Method == "" {
			sc.Requests[i].Method = "GET"
		}
	}
}

// ExecutorConfig is an intermediate representation for executor configuration.
// This is separate from the YAML/JSON schema to allow for duration parsing.
type ExecutorConfig struct {
	Name            string
	Type            string
	VUs             int
	Duration        time.Duration
	Rate            float64
	PreAllocatedVUs int
	MaxVUs          int
	Stages          []ExecutorStage
	GracefulStop    time.Duration
	Pacing          *ExecutorPacing
}

// ExecutorStage represents a parsed stage configuration.
type ExecutorStage struct {
	Duration time.Duration
	Target   int
	Name     string
}

// ExecutorPacing represents parsed pacing configuration.
type ExecutorPacing struct {
	Type     string
	Duration time.Duration
	Min      time.Duration
	Max      time.Duration
}

// ConvertToExecutorConfig converts a ScenarioConfig to an ExecutorConfig.
//
// This function bridges the config package types to the executor package types.
func ConvertToExecutorConfig(name string, sc *ScenarioConfig) (*ExecutorConfig, error) {
	config := &ExecutorConfig{
		Name: name,
		Type: sc.Executor,
		VUs:  sc.VUs,
		Rate: sc.Rate,
	}

	// Parse duration
	if sc.Duration != "" {
		dur, err := ParseDurationString(sc.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %w", err)
		}
		config.Duration = dur
	}

	// Parse graceful stop
	if sc.GracefulStop != "" {
		dur, err := ParseDurationString(sc.GracefulStop)
		if err != nil {
			return nil, fmt.Errorf("invalid gracefulStop: %w", err)
		}
		config.GracefulStop = dur
	}

	// Arrival rate settings
	config.PreAllocatedVUs = sc.PreAllocatedVUs
	config.MaxVUs = sc.MaxVUs

	// Convert stages
	for _, stage := range sc.Stages {
		stageDur, err := ParseDurationString(stage.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid stage duration: %w", err)
		}
		config.Stages = append(config.Stages, ExecutorStage{
			Duration: stageDur,
			Target:   stage.Target,
			Name:     stage.Name,
		})
	}

	// For stage-based executors, calculate total duration from stages
	if len(config.Stages) > 0 && config.Duration == 0 {
		for _, stage := range config.Stages {
			config.Duration += stage.Duration
		}
	}

	// Convert pacing
	if sc.Pacing != nil {
		config.Pacing = &ExecutorPacing{
			Type: sc.Pacing.Type,
		}
		if sc.Pacing.Duration != "" {
			dur, err := ParseDurationString(sc.Pacing.Duration)
			if err != nil {
				return nil, fmt.Errorf("invalid pacing duration: %w", err)
			}
			config.Pacing.Duration = dur
		}
		if sc.Pacing.Min != "" {
			dur, err := ParseDurationString(sc.Pacing.Min)
			if err != nil {
				return nil, fmt.Errorf("invalid pacing min: %w", err)
			}
			config.Pacing.Min = dur
		}
		if sc.Pacing.Max != "" {
			dur, err := ParseDurationString(sc.Pacing.Max)
			if err != nil {
				return nil, fmt.Errorf("invalid pacing max: %w", err)
			}
			config.Pacing.Max = dur
		}
	}

	return config, nil
}
