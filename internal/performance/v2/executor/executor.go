// Package executor provides load generation strategies for performance testing.
package executor

import (
	"context"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// Type identifies the type of executor.
type Type string

const (
	// TypeConstantVUs runs a fixed number of VUs for a duration.
	TypeConstantVUs Type = "constant-vus"

	// TypeRampingVUs ramps VU count up and down according to stages.
	TypeRampingVUs Type = "ramping-vus"

	// TypeConstantArrivalRate maintains a fixed iteration rate.
	TypeConstantArrivalRate Type = "constant-arrival-rate"

	// TypeRampingArrivalRate ramps iteration rate up and down.
	TypeRampingArrivalRate Type = "ramping-arrival-rate"

	// TypePerVUIterations runs a fixed number of iterations per VU.
	TypePerVUIterations Type = "per-vu-iterations"

	// TypeSharedIterations shares a total iteration count across VUs.
	TypeSharedIterations Type = "shared-iterations"
)

// Executor defines the interface for load generation strategies.
//
// Executors control HOW load is generated - whether by managing a pool
// of virtual users or by controlling iteration rates. Each executor
// implements a different load generation strategy suitable for
// different testing scenarios.
type Executor interface {
	// Type returns the executor type.
	Type() Type

	// Init initializes the executor with configuration.
	// Called once before Run().
	Init(ctx context.Context, config *Config) error

	// Run starts the executor and blocks until completion.
	// The executor should respect context cancellation for graceful shutdown.
	Run(ctx context.Context, scheduler *v2.VUScheduler, metrics *metrics.Engine) error

	// GetProgress returns current progress (0.0 to 1.0).
	GetProgress() float64

	// GetActiveVUs returns current active VU count.
	GetActiveVUs() int

	// GetStats returns executor-specific statistics.
	GetStats() *Stats

	// Stop gracefully stops the executor.
	// Called when the test needs to end early.
	Stop(ctx context.Context) error
}

// Config contains configuration for an executor.
type Config struct {
	// Name is the name of this executor instance
	Name string `json:"name" yaml:"name"`

	// Type is the executor type
	Type Type `json:"type" yaml:"type"`

	// VU-based executors
	VUs        int           `json:"vus,omitempty" yaml:"vus,omitempty"`
	Duration   time.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`
	Iterations int64         `json:"iterations,omitempty" yaml:"iterations,omitempty"`

	// Arrival-rate executors
	Rate            float64 `json:"rate,omitempty" yaml:"rate,omitempty"` // iterations/second
	PreAllocatedVUs int     `json:"preAllocatedVUs,omitempty" yaml:"preAllocatedVUs,omitempty"`
	MaxVUs          int     `json:"maxVUs,omitempty" yaml:"maxVUs,omitempty"`

	// Stages (for ramping executors)
	Stages []Stage `json:"stages,omitempty" yaml:"stages,omitempty"`

	// Graceful stop timeout
	GracefulStop time.Duration `json:"gracefulStop,omitempty" yaml:"gracefulStop,omitempty"`

	// Pacing between iterations
	Pacing *PacingConfig `json:"pacing,omitempty" yaml:"pacing,omitempty"`
}

// Stage defines a stage in ramping executors.
type Stage struct {
	// Duration of this stage
	Duration time.Duration `json:"duration" yaml:"duration"`

	// Target VU count (for ramping-vus) or RPS (for ramping-arrival-rate)
	Target int `json:"target" yaml:"target"`

	// Optional name for this stage (for reporting)
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// PacingConfig controls time between iterations.
type PacingConfig struct {
	// Type of pacing: "none", "constant", "random"
	Type PacingType `json:"type" yaml:"type"`

	// Duration for constant pacing
	Duration time.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`

	// Min duration for random pacing
	Min time.Duration `json:"min,omitempty" yaml:"min,omitempty"`

	// Max duration for random pacing
	Max time.Duration `json:"max,omitempty" yaml:"max,omitempty"`
}

// PacingType identifies the type of pacing.
type PacingType string

const (
	PacingNone     PacingType = "none"
	PacingConstant PacingType = "constant"
	PacingRandom   PacingType = "random"
)

// Stats contains real-time executor statistics.
type Stats struct {
	// Timing
	StartTime     time.Time     `json:"startTime"`
	CurrentTime   time.Time     `json:"currentTime"`
	Elapsed       time.Duration `json:"elapsed"`
	TotalDuration time.Duration `json:"totalDuration"`

	// VU stats
	ActiveVUs int `json:"activeVUs"`
	TargetVUs int `json:"targetVUs"`

	// Iteration stats
	Iterations      int64 `json:"iterations"`
	TotalIterations int64 `json:"totalIterations"` // For per-vu-iterations / shared-iterations

	// Stage info (for ramping executors)
	CurrentStage     int    `json:"currentStage"`
	CurrentStageName string `json:"currentStageName"`
	TotalStages      int    `json:"totalStages"`

	// Rate info (for arrival-rate executors)
	CurrentRate float64 `json:"currentRate"`
	TargetRate  float64 `json:"targetRate"`
}

// Validate validates the executor configuration.
func (c *Config) Validate() error {
	if c.Type == "" {
		return &ValidationError{Field: "type", Message: "executor type is required"}
	}

	switch c.Type {
	case TypeConstantVUs:
		if c.VUs <= 0 {
			return &ValidationError{Field: "vus", Message: "vus must be > 0"}
		}
		if c.Duration <= 0 {
			return &ValidationError{Field: "duration", Message: "duration must be > 0"}
		}

	case TypeRampingVUs:
		if len(c.Stages) == 0 {
			return &ValidationError{Field: "stages", Message: "at least one stage is required"}
		}

	case TypeConstantArrivalRate:
		if c.Rate <= 0 {
			return &ValidationError{Field: "rate", Message: "rate must be > 0"}
		}
		if c.Duration <= 0 {
			return &ValidationError{Field: "duration", Message: "duration must be > 0"}
		}

	case TypeRampingArrivalRate:
		if len(c.Stages) == 0 {
			return &ValidationError{Field: "stages", Message: "at least one stage is required"}
		}

	case TypePerVUIterations:
		if c.VUs <= 0 {
			return &ValidationError{Field: "vus", Message: "vus must be > 0"}
		}
		if c.Iterations <= 0 {
			return &ValidationError{Field: "iterations", Message: "iterations must be > 0"}
		}

	case TypeSharedIterations:
		if c.VUs <= 0 {
			return &ValidationError{Field: "vus", Message: "vus must be > 0"}
		}
		if c.Iterations <= 0 {
			return &ValidationError{Field: "iterations", Message: "iterations must be > 0"}
		}

	default:
		return &ValidationError{Field: "type", Message: "unknown executor type: " + string(c.Type)}
	}

	return nil
}

// TotalDuration calculates the total duration for this executor.
func (c *Config) TotalDuration() time.Duration {
	switch c.Type {
	case TypeConstantVUs, TypeConstantArrivalRate:
		return c.Duration

	case TypeRampingVUs, TypeRampingArrivalRate:
		var total time.Duration
		for _, stage := range c.Stages {
			total += stage.Duration
		}
		return total

	case TypePerVUIterations, TypeSharedIterations:
		// Duration not applicable - runs until iterations complete
		return 0

	default:
		return 0
	}
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error on field '" + e.Field + "': " + e.Message
}
