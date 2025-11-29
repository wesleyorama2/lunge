// Package engine provides the main orchestrator for v2 performance testing.
package engine

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/config"
	"github.com/wesleyorama2/lunge/internal/performance/v2/executor"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// Engine is the main orchestrator for v2 performance testing.
//
// It coordinates:
//   - Configuration parsing and validation
//   - Scenario execution with their respective executors
//   - Metrics collection and aggregation
//   - Threshold evaluation
//
// Example usage:
//
//	cfg, _ := config.LoadConfig("test.yaml")
//	engine, _ := NewEngine(cfg)
//	result, _ := engine.Run(context.Background())
//	fmt.Printf("Test passed: %v\n", result.Passed)
type Engine struct {
	// Configuration
	config *config.TestConfig

	// Metrics engine (shared across all scenarios)
	metricsEngine *metrics.Engine

	// HTTP client configuration
	httpConfig v2.HTTPClientConfig

	// Scenario runners
	scenarios map[string]*ScenarioRunner
	mu        sync.RWMutex

	// State
	startTime time.Time
	running   bool
}

// ScenarioRunner manages the execution of a single scenario.
type ScenarioRunner struct {
	Name      string
	Config    *config.ScenarioConfig
	Executor  executor.Executor
	Scheduler *v2.VUScheduler
	Scenario  *v2.Scenario
	Result    *ScenarioResult
}

// ScenarioResult contains the results of a single scenario.
type ScenarioResult struct {
	Name         string                  `json:"name"`
	Executor     string                  `json:"executor"`
	Duration     time.Duration           `json:"duration"`
	Iterations   int64                   `json:"iterations"`
	ActiveVUs    int                     `json:"activeVUs"`
	Metrics      *metrics.Snapshot       `json:"metrics"`
	TimeSeries   []*metrics.TimeBucket   `json:"timeSeries,omitempty"`
	RequestStats map[string]RequestStats `json:"requestStats,omitempty"`
	Error        error                   `json:"error,omitempty"`
}

// RequestStats contains statistics for a specific request.
type RequestStats struct {
	Name    string               `json:"name"`
	Count   int64                `json:"count"`
	Latency metrics.LatencyStats `json:"latency"`
}

// TestResult contains the complete test results.
type TestResult struct {
	// Test metadata
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	StartTime   time.Time     `json:"startTime"`
	EndTime     time.Time     `json:"endTime"`
	Duration    time.Duration `json:"duration"`

	// Scenario results
	Scenarios map[string]*ScenarioResult `json:"scenarios"`

	// Aggregated metrics across all scenarios
	Metrics    *metrics.Snapshot     `json:"metrics"`
	TimeSeries []*metrics.TimeBucket `json:"timeSeries,omitempty"`

	// Threshold evaluation
	Passed     bool              `json:"passed"`
	Thresholds []ThresholdResult `json:"thresholds,omitempty"`

	// Error if the test failed catastrophically
	Error error `json:"error,omitempty"`
}

// ThresholdResult contains the result of a threshold evaluation.
type ThresholdResult struct {
	Metric     string `json:"metric"`
	Expression string `json:"expression"`
	Passed     bool   `json:"passed"`
	Value      string `json:"value"`
	Message    string `json:"message,omitempty"`
}

// NewEngine creates a new v2 performance engine.
//
// Parameters:
//   - cfg: The test configuration (from LoadConfig or created programmatically)
//
// Returns the engine or an error if configuration is invalid.
func NewEngine(cfg *config.TestConfig) (*Engine, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply defaults
	config.ApplyDefaults(cfg)

	// Create HTTP client configuration from settings
	httpConfig := v2.HTTPClientConfig{
		Timeout:             time.Duration(cfg.Settings.Timeout),
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: cfg.Settings.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.Settings.MaxConnectionsPerHost,
		IdleConnTimeout:     90 * time.Second,
		InsecureSkipVerify:  cfg.Settings.InsecureSkipVerify,
		UseSharedClient:     true,
	}

	if httpConfig.Timeout == 0 {
		httpConfig.Timeout = 30 * time.Second
	}
	if httpConfig.MaxIdleConnsPerHost == 0 {
		httpConfig.MaxIdleConnsPerHost = 100
	}

	return &Engine{
		config:     cfg,
		httpConfig: httpConfig,
		scenarios:  make(map[string]*ScenarioRunner),
	}, nil
}

// Run executes all scenarios and returns the test results.
//
// By default, all scenarios run concurrently. If Options.Sequential is true,
// scenarios run one at a time.
//
// The context can be used for cancellation - all scenarios will stop gracefully
// if the context is cancelled.
func (e *Engine) Run(ctx context.Context) (*TestResult, error) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil, fmt.Errorf("engine is already running")
	}
	e.running = true
	e.startTime = time.Now()
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	// Create shared metrics engine
	e.metricsEngine = metrics.NewEngine()
	defer e.metricsEngine.Stop()

	// Set initial phase
	e.metricsEngine.SetPhase(metrics.PhaseInit)

	// Initialize all scenarios
	if err := e.initializeScenarios(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize scenarios: %w", err)
	}

	// Run scenarios
	var scenarioResults map[string]*ScenarioResult
	var runErr error

	if e.config.Options != nil && e.config.Options.Sequential {
		scenarioResults, runErr = e.runScenariosSequentially(ctx)
	} else {
		scenarioResults, runErr = e.runScenariosConcurrently(ctx)
	}

	// Get final metrics
	finalMetrics := e.metricsEngine.GetSnapshot()
	timeSeries := e.metricsEngine.GetTimeSeries()

	// Evaluate thresholds
	thresholdResults := e.evaluateThresholds(finalMetrics)
	passed := true
	for _, tr := range thresholdResults {
		if !tr.Passed {
			passed = false
			break
		}
	}

	result := &TestResult{
		Name:        e.config.Name,
		Description: e.config.Description,
		StartTime:   e.startTime,
		EndTime:     time.Now(),
		Duration:    time.Since(e.startTime),
		Scenarios:   scenarioResults,
		Metrics:     finalMetrics,
		TimeSeries:  timeSeries,
		Passed:      passed,
		Thresholds:  thresholdResults,
		Error:       runErr,
	}

	return result, runErr
}

// initializeScenarios creates executors and schedulers for all scenarios.
func (e *Engine) initializeScenarios(ctx context.Context) error {
	for name, scenarioConfig := range e.config.Scenarios {
		// Create the scenario (requests to execute)
		scenario := e.createScenario(name, scenarioConfig)

		// Create scheduler
		scheduler := v2.NewVUScheduler(scenario, e.metricsEngine, e.httpConfig)

		// Create and initialize executor
		exec, execConfig, err := executor.CreateExecutorFromScenarioConfig(ctx, name, scenarioConfig)
		if err != nil {
			return fmt.Errorf("failed to create executor for scenario %s: %w", name, err)
		}

		// Initialize executor with config
		if err := exec.Init(ctx, execConfig); err != nil {
			return fmt.Errorf("failed to initialize executor for scenario %s: %w", name, err)
		}

		runner := &ScenarioRunner{
			Name:      name,
			Config:    scenarioConfig,
			Executor:  exec,
			Scheduler: scheduler,
			Scenario:  scenario,
		}

		e.scenarios[name] = runner
	}

	return nil
}

// createScenario creates a Scenario from the config.
func (e *Engine) createScenario(name string, sc *config.ScenarioConfig) *v2.Scenario {
	scenario := &v2.Scenario{
		Name:      name,
		Variables: make(map[string]string),
	}

	// Merge global variables with scenario tags
	for k, v := range e.config.Variables {
		scenario.Variables[k] = v
	}
	for k, v := range sc.Tags {
		scenario.Variables[k] = v
	}

	// Add baseUrl to variables
	if e.config.Settings.BaseURL != "" {
		scenario.Variables["baseUrl"] = e.config.Settings.BaseURL
		scenario.Variables["baseURL"] = e.config.Settings.BaseURL
	}

	// Convert requests
	for i, req := range sc.Requests {
		reqConfig := &v2.RequestConfig{
			Name:    req.Name,
			Method:  req.Method,
			URL:     req.URL,
			Headers: req.Headers,
			Body:    req.Body,
		}

		// Assign default name if not provided
		if reqConfig.Name == "" {
			reqConfig.Name = fmt.Sprintf("%s_request_%d", name, i+1)
		}

		// Parse timeout
		if req.Timeout != "" {
			if dur, err := config.ParseDurationString(req.Timeout); err == nil {
				reqConfig.Timeout = dur
			}
		}

		// Parse think time
		if req.ThinkTime != "" {
			if dur, err := config.ParseDurationString(req.ThinkTime); err == nil {
				reqConfig.ThinkTime = dur
			}
		}

		// Convert extracts
		for _, ext := range req.Extract {
			reqConfig.Extract = append(reqConfig.Extract, v2.ExtractConfig{
				Name:   ext.Name,
				Source: ext.Source,
				Path:   ext.Path,
				Regex:  ext.Regex,
			})
		}

		scenario.Requests = append(scenario.Requests, reqConfig)
	}

	return scenario
}

// runScenariosConcurrently runs all scenarios in parallel.
func (e *Engine) runScenariosConcurrently(ctx context.Context) (map[string]*ScenarioResult, error) {
	results := make(map[string]*ScenarioResult)
	var resultsMu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for name, runner := range e.scenarios {
		wg.Add(1)
		go func(name string, runner *ScenarioRunner) {
			defer wg.Done()

			result, err := e.runScenario(ctx, runner)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("scenario %s failed: %w", name, err)
				}
				errMu.Unlock()
			}

			resultsMu.Lock()
			results[name] = result
			resultsMu.Unlock()
		}(name, runner)
	}

	wg.Wait()
	return results, firstErr
}

// runScenariosSequentially runs all scenarios one at a time.
func (e *Engine) runScenariosSequentially(ctx context.Context) (map[string]*ScenarioResult, error) {
	results := make(map[string]*ScenarioResult)

	for name, runner := range e.scenarios {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := e.runScenario(ctx, runner)
		results[name] = result

		if err != nil {
			return results, fmt.Errorf("scenario %s failed: %w", name, err)
		}
	}

	return results, nil
}

// runScenario runs a single scenario.
func (e *Engine) runScenario(ctx context.Context, runner *ScenarioRunner) (*ScenarioResult, error) {
	startTime := time.Now()

	// Run the executor
	err := runner.Executor.Run(ctx, runner.Scheduler, e.metricsEngine)

	duration := time.Since(startTime)
	stats := runner.Executor.GetStats()

	// Get request-specific stats
	requestStats := make(map[string]RequestStats)
	perRequestStats := e.metricsEngine.GetRequestStats()
	for reqName, latencyStats := range perRequestStats {
		requestStats[reqName] = RequestStats{
			Name:    reqName,
			Count:   latencyStats.Count,
			Latency: latencyStats,
		}
	}

	result := &ScenarioResult{
		Name:         runner.Name,
		Executor:     string(runner.Executor.Type()),
		Duration:     duration,
		Iterations:   stats.Iterations,
		ActiveVUs:    stats.ActiveVUs,
		Metrics:      e.metricsEngine.GetSnapshot(),
		RequestStats: requestStats,
		Error:        err,
	}

	// Shutdown scheduler
	runner.Scheduler.Shutdown(30 * time.Second)

	runner.Result = result
	return result, err
}

// evaluateThresholds evaluates all configured thresholds.
func (e *Engine) evaluateThresholds(snapshot *metrics.Snapshot) []ThresholdResult {
	if e.config.Thresholds == nil {
		return nil
	}

	var results []ThresholdResult

	// Evaluate http_req_duration thresholds
	for _, expr := range e.config.Thresholds.HTTPReqDuration {
		result := e.evaluateDurationThreshold(expr, snapshot)
		results = append(results, result)
	}

	// Evaluate http_req_failed thresholds
	for _, expr := range e.config.Thresholds.HTTPReqFailed {
		result := e.evaluateFailedThreshold(expr, snapshot)
		results = append(results, result)
	}

	// Evaluate http_reqs thresholds
	for _, expr := range e.config.Thresholds.HTTPReqs {
		result := e.evaluateRequestsThreshold(expr, snapshot)
		results = append(results, result)
	}

	return results
}

// evaluateDurationThreshold evaluates a duration threshold expression.
func (e *Engine) evaluateDurationThreshold(expr string, snapshot *metrics.Snapshot) ThresholdResult {
	result := ThresholdResult{
		Metric:     "http_req_duration",
		Expression: expr,
	}

	// Parse expression like "p95 < 500ms"
	metric, op, valueStr, err := parseThresholdExpression(expr)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse expression: %v", err)
		return result
	}

	// Get the actual value based on metric
	var actualValue time.Duration
	switch metric {
	case "min":
		actualValue = snapshot.Latency.Min
	case "max":
		actualValue = snapshot.Latency.Max
	case "avg", "med":
		actualValue = snapshot.Latency.Mean
	case "p50":
		actualValue = snapshot.Latency.P50
	case "p90":
		actualValue = snapshot.Latency.P90
	case "p95":
		actualValue = snapshot.Latency.P95
	case "p99":
		actualValue = snapshot.Latency.P99
	default:
		result.Message = fmt.Sprintf("unknown metric: %s", metric)
		return result
	}

	// Parse threshold value
	thresholdValue, err := time.ParseDuration(valueStr)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse threshold value: %v", err)
		return result
	}

	result.Value = actualValue.String()
	result.Passed = compareValues(float64(actualValue), op, float64(thresholdValue))

	if !result.Passed {
		result.Message = fmt.Sprintf("%s is %s, threshold: %s %s", metric, actualValue, op, thresholdValue)
	}

	return result
}

// evaluateFailedThreshold evaluates a failure rate threshold expression.
func (e *Engine) evaluateFailedThreshold(expr string, snapshot *metrics.Snapshot) ThresholdResult {
	result := ThresholdResult{
		Metric:     "http_req_failed",
		Expression: expr,
	}

	// Parse expression like "rate < 0.01"
	metric, op, valueStr, err := parseThresholdExpression(expr)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse expression: %v", err)
		return result
	}

	if metric != "rate" {
		result.Message = fmt.Sprintf("http_req_failed only supports 'rate' metric, got: %s", metric)
		return result
	}

	thresholdValue, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse threshold value: %v", err)
		return result
	}

	result.Value = fmt.Sprintf("%.4f", snapshot.ErrorRate)
	result.Passed = compareValues(snapshot.ErrorRate, op, thresholdValue)

	if !result.Passed {
		result.Message = fmt.Sprintf("error rate is %.4f, threshold: %s %.4f", snapshot.ErrorRate, op, thresholdValue)
	}

	return result
}

// evaluateRequestsThreshold evaluates a request count/rate threshold expression.
func (e *Engine) evaluateRequestsThreshold(expr string, snapshot *metrics.Snapshot) ThresholdResult {
	result := ThresholdResult{
		Metric:     "http_reqs",
		Expression: expr,
	}

	// Parse expression like "count > 1000" or "rate > 100"
	metric, op, valueStr, err := parseThresholdExpression(expr)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse expression: %v", err)
		return result
	}

	thresholdValue, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse threshold value: %v", err)
		return result
	}

	var actualValue float64
	switch metric {
	case "count":
		actualValue = float64(snapshot.TotalRequests)
	case "rate":
		actualValue = snapshot.RPS
	default:
		result.Message = fmt.Sprintf("http_reqs only supports 'count' or 'rate' metrics, got: %s", metric)
		return result
	}

	result.Value = fmt.Sprintf("%.2f", actualValue)
	result.Passed = compareValues(actualValue, op, thresholdValue)

	if !result.Passed {
		result.Message = fmt.Sprintf("%s is %.2f, threshold: %s %.2f", metric, actualValue, op, thresholdValue)
	}

	return result
}

// parseThresholdExpression parses an expression like "p95 < 500ms".
func parseThresholdExpression(expr string) (metric, op, value string, err error) {
	expr = strings.TrimSpace(expr)

	// Regex to parse expressions like "p95 < 500ms" or "rate < 0.01"
	re := regexp.MustCompile(`^(\w+)\s*([<>=!]+)\s*(.+)$`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 4 {
		return "", "", "", fmt.Errorf("invalid expression format: %s", expr)
	}

	return matches[1], matches[2], strings.TrimSpace(matches[3]), nil
}

// compareValues compares two values using the given operator.
func compareValues(actual float64, op string, threshold float64) bool {
	switch op {
	case "<":
		return actual < threshold
	case "<=":
		return actual <= threshold
	case ">":
		return actual > threshold
	case ">=":
		return actual >= threshold
	case "==", "=":
		return actual == threshold
	case "!=", "<>":
		return actual != threshold
	default:
		return false
	}
}

// GetConfig returns the test configuration.
func (e *Engine) GetConfig() *config.TestConfig {
	return e.config
}

// GetMetrics returns the current metrics snapshot.
func (e *Engine) GetMetrics() *metrics.Snapshot {
	if e.metricsEngine == nil {
		return nil
	}
	return e.metricsEngine.GetSnapshot()
}

// GetTimeSeries returns the time series data.
func (e *Engine) GetTimeSeries() []*metrics.TimeBucket {
	if e.metricsEngine == nil {
		return nil
	}
	return e.metricsEngine.GetTimeSeries()
}

// IsRunning returns true if the engine is currently running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// Stop gracefully stops the engine and all running scenarios.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return nil
	}
	scenarios := e.scenarios
	e.mu.RUnlock()

	var lastErr error
	for _, runner := range scenarios {
		if err := runner.Executor.Stop(ctx); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetProgress returns the overall test progress (0.0 to 1.0).
func (e *Engine) GetProgress() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.scenarios) == 0 {
		return 0.0
	}

	var totalProgress float64
	for _, runner := range e.scenarios {
		totalProgress += runner.Executor.GetProgress()
	}

	return totalProgress / float64(len(e.scenarios))
}

// GetScenarioStats returns current stats for all scenarios.
func (e *Engine) GetScenarioStats() map[string]*executor.Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := make(map[string]*executor.Stats)
	for name, runner := range e.scenarios {
		stats[name] = runner.Executor.GetStats()
	}
	return stats
}
