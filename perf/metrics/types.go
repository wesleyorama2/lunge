package metrics

import "time"

// Phase represents a phase of the load test.
type Phase string

const (
	// PhaseInit is the initialization phase before the test starts
	PhaseInit Phase = "init"

	// PhaseWarmup is the warmup phase before the main test
	PhaseWarmup Phase = "warmup"

	// PhaseRampUp is the ramp-up phase when load is increasing
	PhaseRampUp Phase = "ramp-up"

	// PhaseSteady is the steady-state phase at target load
	PhaseSteady Phase = "steady"

	// PhaseRampDown is the ramp-down phase when load is decreasing
	PhaseRampDown Phase = "ramp-down"

	// PhaseCooldown is the cooldown phase after the main test
	PhaseCooldown Phase = "cooldown"

	// PhaseDone indicates the test has completed
	PhaseDone Phase = "done"
)

// Snapshot contains a point-in-time view of all metrics.
type Snapshot struct {
	// TotalRequests is the total number of requests made
	TotalRequests int64 `json:"totalRequests"`

	// SuccessRequests is the number of successful requests (status < 400)
	SuccessRequests int64 `json:"successRequests"`

	// FailedRequests is the number of failed requests
	FailedRequests int64 `json:"failedRequests"`

	// TotalBytes is the total bytes received
	TotalBytes int64 `json:"totalBytes"`

	// Latency contains latency statistics
	Latency LatencyStats `json:"latency"`

	// RPS is the current requests per second
	RPS float64 `json:"rps"`

	// SteadyStateRPS is the RPS calculated only from steady-state phase
	SteadyStateRPS float64 `json:"steadyStateRps"`

	// ErrorRate is the fraction of failed requests (0.0 to 1.0)
	ErrorRate float64 `json:"errorRate"`

	// ActiveVUs is the current number of active virtual users
	ActiveVUs int `json:"activeVUs"`

	// CurrentPhase is the current test phase
	CurrentPhase Phase `json:"currentPhase"`

	// Elapsed is the time elapsed since test start
	Elapsed time.Duration `json:"elapsed"`

	// StartTime is when the test started
	StartTime time.Time `json:"startTime"`

	// Timestamp is when this snapshot was taken
	Timestamp time.Time `json:"timestamp"`
}

// LatencyStats contains latency statistics.
type LatencyStats struct {
	// Min is the minimum latency observed
	Min time.Duration `json:"min"`

	// Max is the maximum latency observed
	Max time.Duration `json:"max"`

	// Mean is the average latency
	Mean time.Duration `json:"mean"`

	// StdDev is the standard deviation of latencies
	StdDev time.Duration `json:"stdDev"`

	// P50 is the 50th percentile (median) latency
	P50 time.Duration `json:"p50"`

	// P90 is the 90th percentile latency
	P90 time.Duration `json:"p90"`

	// P95 is the 95th percentile latency
	P95 time.Duration `json:"p95"`

	// P99 is the 99th percentile latency
	P99 time.Duration `json:"p99"`

	// Count is the number of latency observations
	Count int64 `json:"count"`
}

// LatencyPercentiles holds latency percentile values.
type LatencyPercentiles struct {
	Min time.Duration
	Max time.Duration
	P50 time.Duration
	P90 time.Duration
	P95 time.Duration
	P99 time.Duration
}

// TimeBucket represents metrics for a 1-second interval.
//
// Each bucket captures a snapshot of the system state at a point in time,
// including both cumulative totals and interval-specific deltas.
type TimeBucket struct {
	// Timestamp when this bucket was created
	Timestamp time.Time `json:"timestamp"`

	// Cumulative counters (total since test start)
	TotalRequests  int64 `json:"totalRequests"`
	TotalSuccesses int64 `json:"totalSuccesses"`
	TotalFailures  int64 `json:"totalFailures"`
	TotalBytes     int64 `json:"totalBytes"`

	// Interval metrics (for this 1-second bucket only)
	IntervalRequests int64   `json:"intervalRequests"`
	IntervalRPS      float64 `json:"intervalRPS"`

	// Latency percentiles (from HDR histogram at this point in time)
	LatencyMin time.Duration `json:"latencyMin"`
	LatencyMax time.Duration `json:"latencyMax"`
	LatencyP50 time.Duration `json:"latencyP50"`
	LatencyP90 time.Duration `json:"latencyP90"`
	LatencyP95 time.Duration `json:"latencyP95"`
	LatencyP99 time.Duration `json:"latencyP99"`

	// Active state
	ActiveVUs int   `json:"activeVUs"`
	Phase     Phase `json:"phase"`

	// Error rate for this interval
	IntervalErrorRate float64 `json:"intervalErrorRate"`
}

// PhaseChange records when a phase transition occurred.
type PhaseChange struct {
	// Phase is the phase that was entered
	Phase Phase

	// Timestamp is when the phase change occurred
	Timestamp time.Time

	// Requests is the total request count at the time of the change
	Requests int64
}

// EngineConfig contains configuration for the metrics engine.
type EngineConfig struct {
	// BucketInterval is the interval for time-series buckets (default: 1s)
	BucketInterval time.Duration

	// MaxBuckets is the maximum number of buckets to retain (default: 3600)
	MaxBuckets int

	// HistogramMin is the minimum recordable value in microseconds (default: 1)
	HistogramMin int64

	// HistogramMax is the maximum recordable value in microseconds (default: 3600000000 = 1 hour)
	HistogramMax int64

	// HistogramSigFigs is the number of significant figures (default: 3)
	HistogramSigFigs int
}

// DefaultEngineConfig returns the default configuration.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		BucketInterval:   time.Second,
		MaxBuckets:       3600,
		HistogramMin:     1,
		HistogramMax:     3600000000, // 1 hour in microseconds
		HistogramSigFigs: 3,
	}
}
