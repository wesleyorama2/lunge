package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// Engine collects and aggregates performance metrics using HDR histograms.
//
// Key features:
// - HDR histogram for accurate latency percentiles (O(1) calculation)
// - Continuous time-bucket emission (even during low activity)
// - Lock-free counter updates for high concurrency
// - Phase-aware metrics aggregation
//
// # Thread Safety
//
// Engine is safe for concurrent use. Counters use atomic operations,
// histograms use mutex protection, and the background emitter runs
// in its own goroutine.
type Engine struct {
	// HDR Histogram for latency measurement
	// Range: 1 microsecond to 1 hour, 3 significant figures
	latencyHist   *hdrhistogram.Histogram
	latencyHistMu sync.Mutex

	// Per-request-name histograms (optional, for detailed breakdown)
	requestHists   map[string]*hdrhistogram.Histogram
	requestHistsMu sync.RWMutex

	// Atomic counters for lock-free updates
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64
	totalBytes      atomic.Int64

	// Active VU tracking
	activeVUs atomic.Int32

	// Time-bucketed metrics store
	bucketStore *TimeBucketStore

	// Phase tracking
	currentPhase Phase
	phaseMu      sync.RWMutex
	phaseHistory []PhaseChange

	// Timing
	startTime time.Time

	// Background emitter
	emitterCtx    context.Context
	emitterCancel context.CancelFunc
	emitterWg     sync.WaitGroup

	// Configuration
	config EngineConfig
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

// PhaseChange records when a phase transition occurred.
type PhaseChange struct {
	Phase     Phase
	Timestamp time.Time
	Requests  int64
}

// NewEngine creates a new metrics engine with default configuration.
func NewEngine() *Engine {
	return NewEngineWithConfig(DefaultEngineConfig())
}

// NewEngineWithConfig creates a new metrics engine with custom configuration.
func NewEngineWithConfig(config EngineConfig) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	hist := hdrhistogram.New(config.HistogramMin, config.HistogramMax, config.HistogramSigFigs)

	engine := &Engine{
		latencyHist:   hist,
		requestHists:  make(map[string]*hdrhistogram.Histogram),
		bucketStore:   NewTimeBucketStore(config.MaxBuckets),
		currentPhase:  PhaseInit,
		phaseHistory:  make([]PhaseChange, 0),
		startTime:     time.Now(),
		emitterCtx:    ctx,
		emitterCancel: cancel,
		config:        config,
	}

	// Start background emitter
	engine.emitterWg.Add(1)
	go engine.runEmitter()

	return engine
}

// RecordLatency records a request latency.
//
// This is the primary method for recording request timing.
// It updates both the overall histogram and per-request histograms.
//
// Parameters:
//   - duration: The request latency
//   - requestName: Optional name for per-request breakdown (empty string to skip)
//   - success: Whether the request succeeded
//   - bytes: Number of bytes received
func (e *Engine) RecordLatency(duration time.Duration, requestName string, success bool, bytes int64) {
	// Convert to microseconds for HDR histogram
	latencyMicros := duration.Microseconds()

	// Clamp to valid range
	if latencyMicros < e.config.HistogramMin {
		latencyMicros = e.config.HistogramMin
	}
	if latencyMicros > e.config.HistogramMax {
		latencyMicros = e.config.HistogramMax
	}

	// Record in overall histogram (thread-safe via mutex)
	e.latencyHistMu.Lock()
	e.latencyHist.RecordValue(latencyMicros)
	e.latencyHistMu.Unlock()

	// Record in per-request histogram (if name provided)
	if requestName != "" {
		e.recordRequestHistogram(requestName, latencyMicros)
	}

	// Update atomic counters
	e.totalRequests.Add(1)
	e.totalBytes.Add(bytes)

	if success {
		e.successRequests.Add(1)
	} else {
		e.failedRequests.Add(1)
	}

	// Record in bucket store for time-series
	e.bucketStore.RecordRequest(success, bytes)
}

// recordRequestHistogram records a latency in a per-request histogram.
// NOTE: HDR histogram RecordValue is NOT thread-safe, so we must hold a lock.
func (e *Engine) recordRequestHistogram(name string, latencyMicros int64) {
	e.requestHistsMu.Lock()
	defer e.requestHistsMu.Unlock()

	hist, exists := e.requestHists[name]
	if !exists {
		hist = hdrhistogram.New(e.config.HistogramMin, e.config.HistogramMax, e.config.HistogramSigFigs)
		e.requestHists[name] = hist
	}

	hist.RecordValue(latencyMicros)
}

// SetPhase updates the current test phase.
//
// This is called by executors to mark phase transitions.
// Phase information is included in time-series buckets.
func (e *Engine) SetPhase(phase Phase) {
	e.phaseMu.Lock()
	defer e.phaseMu.Unlock()

	if e.currentPhase == phase {
		return // No change
	}

	e.currentPhase = phase
	e.phaseHistory = append(e.phaseHistory, PhaseChange{
		Phase:     phase,
		Timestamp: time.Now(),
		Requests:  e.totalRequests.Load(),
	})
}

// GetPhase returns the current test phase.
func (e *Engine) GetPhase() Phase {
	e.phaseMu.RLock()
	defer e.phaseMu.RUnlock()
	return e.currentPhase
}

// SetActiveVUs updates the active VU count.
func (e *Engine) SetActiveVUs(count int) {
	e.activeVUs.Store(int32(count))
}

// GetActiveVUs returns the current active VU count.
func (e *Engine) GetActiveVUs() int {
	return int(e.activeVUs.Load())
}

// runEmitter runs the background time-bucket emitter.
func (e *Engine) runEmitter() {
	defer e.emitterWg.Done()

	ticker := time.NewTicker(e.config.BucketInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.emitterCtx.Done():
			return
		case <-ticker.C:
			e.emitBucket()
		}
	}
}

// emitBucket creates a new time-series bucket with current metrics.
func (e *Engine) emitBucket() {
	// Get current latency percentiles
	latencies := e.GetLatencyPercentiles()

	// Get current phase and VU count
	phase := e.GetPhase()
	activeVUs := e.GetActiveVUs()

	// Get current totals
	totalRequests := e.totalRequests.Load()
	totalSuccesses := e.successRequests.Load()
	totalFailures := e.failedRequests.Load()
	totalBytes := e.totalBytes.Load()

	// Create bucket
	e.bucketStore.CreateBucket(
		totalRequests, totalSuccesses, totalFailures, totalBytes,
		latencies, activeVUs, phase,
	)
}

// GetLatencyPercentiles returns current latency percentiles.
func (e *Engine) GetLatencyPercentiles() LatencyPercentiles {
	e.latencyHistMu.Lock()
	defer e.latencyHistMu.Unlock()

	return LatencyPercentiles{
		Min: time.Duration(e.latencyHist.Min()) * time.Microsecond,
		Max: time.Duration(e.latencyHist.Max()) * time.Microsecond,
		P50: time.Duration(e.latencyHist.ValueAtQuantile(50)) * time.Microsecond,
		P90: time.Duration(e.latencyHist.ValueAtQuantile(90)) * time.Microsecond,
		P95: time.Duration(e.latencyHist.ValueAtQuantile(95)) * time.Microsecond,
		P99: time.Duration(e.latencyHist.ValueAtQuantile(99)) * time.Microsecond,
	}
}

// GetSnapshot returns a point-in-time snapshot of all metrics.
func (e *Engine) GetSnapshot() *Snapshot {
	e.latencyHistMu.Lock()
	latencyStats := LatencyStats{
		Min:    time.Duration(e.latencyHist.Min()) * time.Microsecond,
		Max:    time.Duration(e.latencyHist.Max()) * time.Microsecond,
		Mean:   time.Duration(e.latencyHist.Mean()) * time.Microsecond,
		StdDev: time.Duration(e.latencyHist.StdDev()) * time.Microsecond,
		P50:    time.Duration(e.latencyHist.ValueAtQuantile(50)) * time.Microsecond,
		P90:    time.Duration(e.latencyHist.ValueAtQuantile(90)) * time.Microsecond,
		P95:    time.Duration(e.latencyHist.ValueAtQuantile(95)) * time.Microsecond,
		P99:    time.Duration(e.latencyHist.ValueAtQuantile(99)) * time.Microsecond,
		Count:  e.latencyHist.TotalCount(),
	}
	e.latencyHistMu.Unlock()

	elapsed := time.Since(e.startTime)
	totalReqs := e.totalRequests.Load()
	failedReqs := e.failedRequests.Load()

	// Calculate overall RPS
	overallRPS := 0.0
	if elapsed.Seconds() > 0 {
		overallRPS = float64(totalReqs) / elapsed.Seconds()
	}

	// Calculate steady-state RPS (more accurate)
	steadyRPS, steadyBuckets := e.bucketStore.CalculateSteadyStateRPS()

	// Use steady-state RPS if available, otherwise overall
	rps := overallRPS
	if steadyBuckets > 0 {
		rps = steadyRPS
	}

	// Error rate
	errorRate := 0.0
	if totalReqs > 0 {
		errorRate = float64(failedReqs) / float64(totalReqs)
	}

	return &Snapshot{
		TotalRequests:   totalReqs,
		SuccessRequests: e.successRequests.Load(),
		FailedRequests:  failedReqs,
		TotalBytes:      e.totalBytes.Load(),
		Latency:         latencyStats,
		RPS:             rps,
		SteadyStateRPS:  steadyRPS,
		ErrorRate:       errorRate,
		ActiveVUs:       e.GetActiveVUs(),
		CurrentPhase:    e.GetPhase(),
		Elapsed:         elapsed,
		StartTime:       e.startTime,
		Timestamp:       time.Now(),
	}
}

// GetTimeSeries returns all time-series buckets.
func (e *Engine) GetTimeSeries() []*TimeBucket {
	return e.bucketStore.GetBuckets()
}

// GetPhaseHistory returns the history of phase changes.
func (e *Engine) GetPhaseHistory() []PhaseChange {
	e.phaseMu.RLock()
	defer e.phaseMu.RUnlock()

	result := make([]PhaseChange, len(e.phaseHistory))
	copy(result, e.phaseHistory)
	return result
}

// GetRequestStats returns per-request statistics.
func (e *Engine) GetRequestStats() map[string]LatencyStats {
	e.requestHistsMu.RLock()
	defer e.requestHistsMu.RUnlock()

	result := make(map[string]LatencyStats)

	for name, hist := range e.requestHists {
		result[name] = LatencyStats{
			Min:    time.Duration(hist.Min()) * time.Microsecond,
			Max:    time.Duration(hist.Max()) * time.Microsecond,
			Mean:   time.Duration(hist.Mean()) * time.Microsecond,
			StdDev: time.Duration(hist.StdDev()) * time.Microsecond,
			P50:    time.Duration(hist.ValueAtQuantile(50)) * time.Microsecond,
			P90:    time.Duration(hist.ValueAtQuantile(90)) * time.Microsecond,
			P95:    time.Duration(hist.ValueAtQuantile(95)) * time.Microsecond,
			P99:    time.Duration(hist.ValueAtQuantile(99)) * time.Microsecond,
			Count:  hist.TotalCount(),
		}
	}

	return result
}

// Stop stops the metrics engine and emits a final bucket.
func (e *Engine) Stop() {
	e.emitterCancel()
	e.emitterWg.Wait()

	// Emit final bucket
	e.emitBucket()
}

// Reset resets all metrics to initial state.
func (e *Engine) Reset() {
	e.latencyHistMu.Lock()
	e.latencyHist.Reset()
	e.latencyHistMu.Unlock()

	e.requestHistsMu.Lock()
	e.requestHists = make(map[string]*hdrhistogram.Histogram)
	e.requestHistsMu.Unlock()

	e.totalRequests.Store(0)
	e.successRequests.Store(0)
	e.failedRequests.Store(0)
	e.totalBytes.Store(0)
	e.activeVUs.Store(0)

	e.phaseMu.Lock()
	e.currentPhase = PhaseInit
	e.phaseHistory = make([]PhaseChange, 0)
	e.phaseMu.Unlock()

	e.bucketStore.Reset()
	e.startTime = time.Now()
}

// Snapshot contains a point-in-time view of all metrics.
type Snapshot struct {
	TotalRequests   int64         `json:"totalRequests"`
	SuccessRequests int64         `json:"successRequests"`
	FailedRequests  int64         `json:"failedRequests"`
	TotalBytes      int64         `json:"totalBytes"`
	Latency         LatencyStats  `json:"latency"`
	RPS             float64       `json:"rps"`
	SteadyStateRPS  float64       `json:"steadyStateRps"`
	ErrorRate       float64       `json:"errorRate"`
	ActiveVUs       int           `json:"activeVUs"`
	CurrentPhase    Phase         `json:"currentPhase"`
	Elapsed         time.Duration `json:"elapsed"`
	StartTime       time.Time     `json:"startTime"`
	Timestamp       time.Time     `json:"timestamp"`
}

// LatencyStats contains latency statistics.
type LatencyStats struct {
	Min    time.Duration `json:"min"`
	Max    time.Duration `json:"max"`
	Mean   time.Duration `json:"mean"`
	StdDev time.Duration `json:"stdDev"`
	P50    time.Duration `json:"p50"`
	P90    time.Duration `json:"p90"`
	P95    time.Duration `json:"p95"`
	P99    time.Duration `json:"p99"`
	Count  int64         `json:"count"`
}
