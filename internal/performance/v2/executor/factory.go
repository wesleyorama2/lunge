package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/config"
)

// NewExecutor creates a new executor of the specified type.
//
// Supported types:
//   - "constant-vus" - Fixed number of VUs for a duration
//   - "ramping-vus" - VU count ramps up/down according to stages
//   - "constant-arrival-rate" - Fixed iteration rate (open model)
//   - "ramping-arrival-rate" - Iteration rate ramps up/down
//
// Returns an uninitialized executor. Call Init() before Run().
func NewExecutor(executorType Type) (Executor, error) {
	switch executorType {
	case TypeConstantVUs:
		return NewConstantVUs(), nil
	case TypeRampingVUs:
		return NewRampingVUs(), nil
	case TypeConstantArrivalRate:
		return NewConstantArrivalRate(), nil
	case TypeRampingArrivalRate:
		return NewRampingArrivalRate(), nil
	case TypePerVUIterations:
		return nil, fmt.Errorf("per-vu-iterations executor not yet implemented")
	case TypeSharedIterations:
		return nil, fmt.Errorf("shared-iterations executor not yet implemented")
	default:
		return nil, fmt.Errorf("unknown executor type: %s", executorType)
	}
}

// NewExecutorFromString creates a new executor from a string type name.
//
// This is a convenience wrapper around NewExecutor that accepts string input.
func NewExecutorFromString(executorType string) (Executor, error) {
	return NewExecutor(Type(executorType))
}

// CreateAndInitExecutor creates and initializes an executor with the given config.
//
// This is a convenience function that combines NewExecutor and Init.
func CreateAndInitExecutor(ctx context.Context, cfg *Config) (Executor, error) {
	exec, err := NewExecutor(cfg.Type)
	if err != nil {
		return nil, err
	}

	if err := exec.Init(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize executor: %w", err)
	}

	return exec, nil
}

// CreateExecutorFromScenarioConfig creates and initializes an executor from a scenario config.
//
// This function bridges the config.ScenarioConfig (from YAML/JSON) to the executor.Config.
// It handles all the type conversions and duration parsing.
func CreateExecutorFromScenarioConfig(ctx context.Context, name string, sc *config.ScenarioConfig) (Executor, *Config, error) {
	// Convert scenario config to executor config
	execConfig, err := convertScenarioToExecutorConfig(name, sc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert scenario config: %w", err)
	}

	// Create and initialize executor
	exec, err := CreateAndInitExecutor(ctx, execConfig)
	if err != nil {
		return nil, nil, err
	}

	return exec, execConfig, nil
}

// convertScenarioToExecutorConfig converts a config.ScenarioConfig to executor.Config.
func convertScenarioToExecutorConfig(name string, sc *config.ScenarioConfig) (*Config, error) {
	cfg := &Config{
		Name:            name,
		Type:            Type(sc.Executor),
		VUs:             sc.VUs,
		Rate:            sc.Rate,
		PreAllocatedVUs: sc.PreAllocatedVUs,
		MaxVUs:          sc.MaxVUs,
	}

	// Parse duration
	if sc.Duration != "" {
		dur, err := config.ParseDurationString(sc.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %w", err)
		}
		cfg.Duration = dur
	}

	// Parse graceful stop
	if sc.GracefulStop != "" {
		dur, err := config.ParseDurationString(sc.GracefulStop)
		if err != nil {
			return nil, fmt.Errorf("invalid gracefulStop: %w", err)
		}
		cfg.GracefulStop = dur
	}

	// Convert stages
	for _, stage := range sc.Stages {
		stageDur, err := config.ParseDurationString(stage.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid stage duration: %w", err)
		}
		cfg.Stages = append(cfg.Stages, Stage{
			Duration: stageDur,
			Target:   stage.Target,
			Name:     stage.Name,
		})
	}

	// For stage-based executors, calculate total duration from stages if not set
	if len(cfg.Stages) > 0 && cfg.Duration == 0 {
		for _, stage := range cfg.Stages {
			cfg.Duration += stage.Duration
		}
	}

	// Convert pacing
	if sc.Pacing != nil {
		cfg.Pacing = &PacingConfig{
			Type: PacingType(sc.Pacing.Type),
		}
		if sc.Pacing.Duration != "" {
			dur, err := config.ParseDurationString(sc.Pacing.Duration)
			if err != nil {
				return nil, fmt.Errorf("invalid pacing duration: %w", err)
			}
			cfg.Pacing.Duration = dur
		}
		if sc.Pacing.Min != "" {
			dur, err := config.ParseDurationString(sc.Pacing.Min)
			if err != nil {
				return nil, fmt.Errorf("invalid pacing min: %w", err)
			}
			cfg.Pacing.Min = dur
		}
		if sc.Pacing.Max != "" {
			dur, err := config.ParseDurationString(sc.Pacing.Max)
			if err != nil {
				return nil, fmt.Errorf("invalid pacing max: %w", err)
			}
			cfg.Pacing.Max = dur
		}
	}

	return cfg, nil
}

// IsValidExecutorType returns true if the type is a valid executor type.
func IsValidExecutorType(executorType string) bool {
	switch Type(executorType) {
	case TypeConstantVUs, TypeRampingVUs, TypeConstantArrivalRate, TypeRampingArrivalRate:
		return true
	case TypePerVUIterations, TypeSharedIterations:
		return true // Valid but not yet implemented
	default:
		return false
	}
}

// GetSupportedExecutors returns a list of all supported executor types.
func GetSupportedExecutors() []Type {
	return []Type{
		TypeConstantVUs,
		TypeRampingVUs,
		TypeConstantArrivalRate,
		TypeRampingArrivalRate,
		// TypePerVUIterations,    // Not yet implemented
		// TypeSharedIterations,   // Not yet implemented
	}
}

// ExecutorDescription provides documentation for an executor type.
type ExecutorDescription struct {
	Type        Type
	Name        string
	Description string
	UseCases    []string
}

// GetExecutorDescription returns documentation for an executor type.
func GetExecutorDescription(executorType Type) *ExecutorDescription {
	switch executorType {
	case TypeConstantVUs:
		return &ExecutorDescription{
			Type:        TypeConstantVUs,
			Name:        "Constant VUs",
			Description: "Runs a fixed number of VUs for a specified duration. Each VU runs as fast as it can (closed model).",
			UseCases: []string{
				"Basic load testing",
				"Determining max throughput for N concurrent users",
				"Simple soak testing",
			},
		}
	case TypeRampingVUs:
		return &ExecutorDescription{
			Type:        TypeRampingVUs,
			Name:        "Ramping VUs",
			Description: "Ramps VU count up and down according to stages. Smoothly interpolates between stage targets.",
			UseCases: []string{
				"Realistic traffic simulation (morning ramp-up, evening ramp-down)",
				"Finding the breaking point of a system",
				"Stress testing with gradual load increase",
			},
		}
	case TypeConstantArrivalRate:
		return &ExecutorDescription{
			Type:        TypeConstantArrivalRate,
			Name:        "Constant Arrival Rate",
			Description: "Maintains a fixed iteration rate (iterations per second), regardless of response time. This is an open model where VUs are dynamically allocated.",
			UseCases: []string{
				"Testing system behavior under constant load",
				"SLA validation (e.g., system must handle 100 RPS)",
				"Capacity testing with predictable arrival patterns",
			},
		}
	case TypeRampingArrivalRate:
		return &ExecutorDescription{
			Type:        TypeRampingArrivalRate,
			Name:        "Ramping Arrival Rate",
			Description: "Ramps iteration rate up and down according to stages. Like constant-arrival-rate but with variable rate over time.",
			UseCases: []string{
				"Simulating realistic traffic patterns (gradual load increase)",
				"Finding the breaking point of a system",
				"Testing auto-scaling behavior",
				"Gradual load test warm-up",
			},
		}
	default:
		return nil
	}
}

// CalculateEstimatedDuration calculates the estimated duration for an executor config.
func CalculateEstimatedDuration(cfg *Config) time.Duration {
	return cfg.TotalDuration()
}

// CalculateMaxVUs returns the maximum number of VUs that might be used.
//
// For VU-based executors, this is the VU count or the max stage target.
// For arrival-rate executors, this is MaxVUs.
func CalculateMaxVUs(cfg *Config) int {
	switch cfg.Type {
	case TypeConstantVUs:
		return cfg.VUs
	case TypeRampingVUs:
		maxVUs := 0
		for _, stage := range cfg.Stages {
			if stage.Target > maxVUs {
				maxVUs = stage.Target
			}
		}
		return maxVUs
	case TypeConstantArrivalRate, TypeRampingArrivalRate:
		return cfg.MaxVUs
	default:
		return cfg.VUs
	}
}
