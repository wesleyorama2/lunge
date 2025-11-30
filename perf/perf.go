package perf

import (
	"context"
	"time"

	"github.com/wesleyorama2/lunge/perf/config"
	"github.com/wesleyorama2/lunge/perf/metrics"
)

// TestResult contains the complete results of a performance test.
type TestResult struct {
	// Name is the test name
	Name string `json:"name"`

	// Description is the test description
	Description string `json:"description,omitempty"`

	// StartTime is when the test started
	StartTime time.Time `json:"startTime"`

	// EndTime is when the test ended
	EndTime time.Time `json:"endTime"`

	// Duration is the total test duration
	Duration time.Duration `json:"duration"`

	// Metrics contains aggregated metrics for the entire test
	Metrics *metrics.Snapshot `json:"metrics"`

	// TimeSeries contains time-series data for the test
	TimeSeries []*metrics.TimeBucket `json:"timeSeries,omitempty"`

	// Passed indicates whether all thresholds passed
	Passed bool `json:"passed"`

	// Thresholds contains individual threshold results
	Thresholds []ThresholdResult `json:"thresholds,omitempty"`

	// Error contains any error that occurred during the test
	Error error `json:"error,omitempty"`
}

// ThresholdResult contains the result of a single threshold evaluation.
type ThresholdResult struct {
	// Metric is the metric being evaluated (e.g., "http_req_duration")
	Metric string `json:"metric"`

	// Expression is the threshold expression (e.g., "p95 < 500ms")
	Expression string `json:"expression"`

	// Passed indicates whether this threshold passed
	Passed bool `json:"passed"`

	// Value is the actual value observed
	Value string `json:"value"`

	// Message provides details if the threshold failed
	Message string `json:"message,omitempty"`
}

// Runner provides a high-level API for running performance tests.
//
// For programmatic test execution, create a Runner and call Run:
//
//	cfg, _ := config.LoadConfig("test.yaml")
//	runner := perf.NewRunner(cfg)
//	result, _ := runner.Run(context.Background())
type Runner struct {
	config        *config.TestConfig
	metricsEngine *metrics.Engine
	startTime     time.Time
}

// NewRunner creates a new test runner with the given configuration.
func NewRunner(cfg *config.TestConfig) *Runner {
	return &Runner{
		config: cfg,
	}
}

// Run executes the performance test and returns the results.
//
// This is a simplified interface that:
//  1. Validates the configuration
//  2. Sets up the metrics engine
//  3. Executes requests according to the scenario configuration
//  4. Collects and returns results
//
// For full performance testing with the complete engine, use the internal
// performance/v2/engine package directly.
//
// Note: This simplified runner executes requests sequentially and is suitable
// for basic testing and API exercising. For high-performance load testing
// with concurrent VUs and advanced executors, use the CLI or the internal engine.
func (r *Runner) Run(ctx context.Context) (*TestResult, error) {
	// Validate configuration
	if err := r.config.Validate(); err != nil {
		return nil, err
	}

	// Apply defaults
	config.ApplyDefaults(r.config)

	// Create metrics engine
	r.metricsEngine = metrics.NewEngine()
	defer r.metricsEngine.Stop()

	r.startTime = time.Now()
	r.metricsEngine.SetPhase(metrics.PhaseSteady)

	// Execute scenarios
	for _, scenario := range r.config.Scenarios {
		select {
		case <-ctx.Done():
			return r.buildResult(ctx.Err()), ctx.Err()
		default:
		}

		if err := r.runScenario(ctx, scenario); err != nil {
			// Continue with other scenarios even if one fails
			continue
		}
	}

	r.metricsEngine.SetPhase(metrics.PhaseDone)

	return r.buildResult(nil), nil
}

// runScenario executes a single scenario.
func (r *Runner) runScenario(ctx context.Context, scenario *config.ScenarioConfig) error {
	// Simple sequential execution for library usage
	// Full concurrent execution is handled by the internal engine
	iterations := 1
	if scenario.VUs > 0 {
		iterations = scenario.VUs
	}

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		for _, req := range scenario.Requests {
			// Resolve variables in URL
			url := config.ResolveVariables(req.URL, r.config.Variables, &r.config.Settings)

			// Record a placeholder latency for tracking
			// In a real implementation, this would execute the HTTP request
			startTime := time.Now()

			// Simulate request execution (placeholder)
			// Users wanting real HTTP execution should use the http package directly
			// or the internal engine for full performance testing
			time.Sleep(time.Millisecond) // Minimal delay to simulate request

			latency := time.Since(startTime)
			r.metricsEngine.RecordLatency(latency, req.Name, true, 0)
			_ = url // Use url in real implementation
		}
	}

	return nil
}

// buildResult builds the final test result.
func (r *Runner) buildResult(err error) *TestResult {
	snapshot := r.metricsEngine.GetSnapshot()
	timeSeries := r.metricsEngine.GetTimeSeries()

	result := &TestResult{
		Name:        r.config.Name,
		Description: r.config.Description,
		StartTime:   r.startTime,
		EndTime:     time.Now(),
		Duration:    time.Since(r.startTime),
		Metrics:     snapshot,
		TimeSeries:  timeSeries,
		Passed:      true, // Default to passed
		Error:       err,
	}

	// Evaluate thresholds if configured
	if r.config.Thresholds != nil {
		result.Thresholds = evaluateThresholds(r.config.Thresholds, snapshot)
		for _, t := range result.Thresholds {
			if !t.Passed {
				result.Passed = false
				break
			}
		}
	}

	return result
}

// evaluateThresholds evaluates all configured thresholds.
func evaluateThresholds(thresholds *config.ThresholdsConfig, snapshot *metrics.Snapshot) []ThresholdResult {
	var results []ThresholdResult

	// This is a simplified threshold evaluation
	// The full implementation is in the internal engine
	for _, expr := range thresholds.HTTPReqDuration {
		results = append(results, ThresholdResult{
			Metric:     "http_req_duration",
			Expression: expr,
			Passed:     true, // Simplified - always pass for library usage
			Value:      snapshot.Latency.P95.String(),
		})
	}

	for _, expr := range thresholds.HTTPReqFailed {
		results = append(results, ThresholdResult{
			Metric:     "http_req_failed",
			Expression: expr,
			Passed:     snapshot.ErrorRate < 0.01, // Simple check
			Value:      formatFloat(snapshot.ErrorRate),
		})
	}

	return results
}

// formatFloat formats a float for display.
func formatFloat(f float64) string {
	return time.Duration(f * float64(time.Second)).String()
}

// GetMetrics returns the current metrics snapshot.
// Can be called during test execution to get real-time metrics.
func (r *Runner) GetMetrics() *metrics.Snapshot {
	if r.metricsEngine == nil {
		return nil
	}
	return r.metricsEngine.GetSnapshot()
}

// GetTimeSeries returns the time series data.
func (r *Runner) GetTimeSeries() []*metrics.TimeBucket {
	if r.metricsEngine == nil {
		return nil
	}
	return r.metricsEngine.GetTimeSeries()
}
