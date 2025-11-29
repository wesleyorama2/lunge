package executor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
	"github.com/wesleyorama2/lunge/internal/performance/v2/rate"
)

// RampingArrivalRate ramps iteration rate up and down according to stages.
//
// Like ConstantArrivalRate, this is an open-model executor where iterations
// are scheduled at a target rate regardless of response time. The difference
// is that the rate changes over time according to defined stages.
//
// This executor uses a LeakyBucket with SetRate() called every 100ms for
// smooth rate transitions without bursting.
//
// IMPORTANT: The first stage STARTS at its target rate immediately (no ramp from 0).
// Subsequent stages ramp from the previous stage's target to the current stage's target.
// If you need to explicitly ramp from 0, add a first stage with target: 0.
//
// Use cases:
//   - Simulating realistic traffic patterns (gradual load increase)
//   - Finding the breaking point of a system
//   - Testing auto-scaling behavior
//   - Gradual load test warm-up
//
// Example:
//
//	config:
//	  type: ramping-arrival-rate
//	  stages:
//	    - duration: 1m
//	      target: 50           # Start at 50 RPS and hold for 1 minute
//	    - duration: 3m
//	      target: 50           # Stay at 50 RPS for 3 minutes
//	    - duration: 1m
//	      target: 100          # Ramp from 50 to 100 RPS over 1 minute
//	    - duration: 1m
//	      target: 0            # Ramp down from 100 to 0 RPS
//	  preAllocatedVUs: 10
//	  maxVUs: 100
type RampingArrivalRate struct {
	config    *Config
	scheduler *v2.VUScheduler
	metrics   *metrics.Engine

	// Rate limiter
	bucket *rate.LeakyBucket

	// VU pool management
	vuPool     chan *v2.VirtualUser // Available VUs ready to execute
	allVUs     []*v2.VirtualUser    // All VUs (for cleanup)
	currentVUs atomic.Int32         // Current total VU count
	vuPoolMu   sync.Mutex

	// State
	startTime    time.Time
	iterations   atomic.Int64
	currentStage atomic.Int32
	currentRate  atomic.Int64 // Stored as rate * 1000 for precision
	running      atomic.Bool

	// Cancellation
	cancelMu   sync.Mutex // Protects cancelFunc
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup

	// Stats
	mu sync.RWMutex
}

// NewRampingArrivalRate creates a new ramping arrival rate executor.
func NewRampingArrivalRate() *RampingArrivalRate {
	return &RampingArrivalRate{}
}

// Type returns the executor type.
func (e *RampingArrivalRate) Type() Type {
	return TypeRampingArrivalRate
}

// Init initializes the executor with configuration.
func (e *RampingArrivalRate) Init(ctx context.Context, config *Config) error {
	if config.Type != TypeRampingArrivalRate {
		return fmt.Errorf("invalid config type: expected %s, got %s", TypeRampingArrivalRate, config.Type)
	}

	if err := config.Validate(); err != nil {
		return err
	}

	// Set defaults for VU pool
	if config.PreAllocatedVUs <= 0 {
		config.PreAllocatedVUs = 1
	}
	if config.MaxVUs <= 0 {
		config.MaxVUs = config.PreAllocatedVUs
	}
	if config.MaxVUs < config.PreAllocatedVUs {
		config.MaxVUs = config.PreAllocatedVUs
	}

	e.config = config
	return nil
}

// Run starts the executor and blocks until completion.
func (e *RampingArrivalRate) Run(ctx context.Context, scheduler *v2.VUScheduler, metricsEngine *metrics.Engine) error {
	e.scheduler = scheduler
	e.metrics = metricsEngine
	e.running.Store(true)
	e.startTime = time.Now()

	// Calculate initial rate (first stage starts at its target rate)
	initialRate := e.calculateTargetRate()
	if initialRate < 0.01 {
		initialRate = 0.01 // Minimum rate to avoid division by zero
	}

	// Create rate limiter with initial rate
	e.bucket = rate.NewLeakyBucket(initialRate)
	e.currentRate.Store(int64(initialRate * 1000))

	// Calculate total duration from stages
	totalDuration := e.config.TotalDuration()

	// Initialize VU pool
	e.vuPool = make(chan *v2.VirtualUser, e.config.MaxVUs)
	e.allVUs = make([]*v2.VirtualUser, 0, e.config.MaxVUs)

	// Create cancellable context with duration timeout
	runCtx, cancel := context.WithTimeout(ctx, totalDuration)
	e.cancelMu.Lock()
	e.cancelFunc = cancel
	e.cancelMu.Unlock()
	defer cancel()

	// Pre-allocate VUs
	for i := 0; i < e.config.PreAllocatedVUs; i++ {
		vu := scheduler.SpawnVU()
		e.allVUs = append(e.allVUs, vu)
		e.vuPool <- vu
		e.currentVUs.Add(1)
	}

	e.metrics.SetActiveVUs(e.config.PreAllocatedVUs)

	// Start rate controller (adjusts rate smoothly every 100ms)
	e.wg.Add(1)
	go e.rateController(runCtx)

	// Start iteration scheduler
	e.wg.Add(1)
	go e.iterationScheduler(runCtx)

	// Wait for context to complete
	<-runCtx.Done()

	// Wait for all in-flight iterations to complete
	e.wg.Wait()

	// Graceful shutdown
	e.gracefulShutdown()

	// Mark as done
	e.metrics.SetPhase(metrics.PhaseDone)
	e.running.Store(false)

	return nil
}

// rateController adjusts the rate according to stages every 100ms.
func (e *RampingArrivalRate) rateController(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			targetRate := e.calculateTargetRate()
			if targetRate < 0.01 {
				targetRate = 0.01 // Minimum rate
			}

			e.bucket.SetRate(targetRate)
			e.currentRate.Store(int64(targetRate * 1000))
			e.updatePhase()
		}
	}
}

// calculateTargetRate calculates the target rate (RPS) based on elapsed time.
//
// For the first stage, the rate STARTS at the stage's target (no ramp from 0).
// For subsequent stages, the rate interpolates linearly from the previous target
// to the current target over the stage duration.
func (e *RampingArrivalRate) calculateTargetRate() float64 {
	elapsed := time.Since(e.startTime)

	// Find current stage and interpolate rate
	var stageStart time.Duration

	// For the first stage, prevTarget is the first stage's target (start immediately at target)
	// This ensures load generation begins immediately rather than ramping from 0
	prevTarget := 0.0
	if len(e.config.Stages) > 0 {
		prevTarget = float64(e.config.Stages[0].Target)
	}

	for i, stage := range e.config.Stages {
		stageEnd := stageStart + stage.Duration

		if elapsed < stageEnd {
			e.currentStage.Store(int32(i))

			// Calculate progress within this stage (0.0 to 1.0)
			stageProgress := float64(elapsed-stageStart) / float64(stage.Duration)
			if stageProgress < 0 {
				stageProgress = 0
			}
			if stageProgress > 1 {
				stageProgress = 1
			}

			// Linear interpolation between previous and current target
			targetRate := prevTarget + float64(stage.Target-int(prevTarget))*stageProgress
			return targetRate
		}

		prevTarget = float64(stage.Target)
		stageStart = stageEnd
	}

	// Past all stages - return last target
	if len(e.config.Stages) > 0 {
		return float64(e.config.Stages[len(e.config.Stages)-1].Target)
	}
	return 0
}

// updatePhase updates the metrics phase based on current stage.
func (e *RampingArrivalRate) updatePhase() {
	stageIdx := int(e.currentStage.Load())
	if stageIdx >= len(e.config.Stages) {
		return
	}

	stage := e.config.Stages[stageIdx]

	// Determine phase based on stage characteristics
	// First stage: prevTarget is the same as target (start at target, no ramp)
	// Subsequent stages: prevTarget is the previous stage's target
	prevTarget := stage.Target // Default for first stage (steady)
	if stageIdx > 0 {
		prevTarget = e.config.Stages[stageIdx-1].Target
	}

	if stageIdx == 0 {
		// First stage - always steady since we start at target
		e.metrics.SetPhase(metrics.PhaseSteady)
	} else if stageIdx == len(e.config.Stages)-1 && stage.Target == 0 {
		// Last stage ramping down
		e.metrics.SetPhase(metrics.PhaseRampDown)
	} else if stage.Target == prevTarget {
		e.metrics.SetPhase(metrics.PhaseSteady)
	} else if stage.Target > prevTarget {
		e.metrics.SetPhase(metrics.PhaseRampUp)
	} else {
		e.metrics.SetPhase(metrics.PhaseRampDown)
	}
}

// iterationScheduler schedules iterations based on the current rate.
func (e *RampingArrivalRate) iterationScheduler(ctx context.Context) {
	defer e.wg.Done()

	for {
		// Wait for next iteration slot
		err := e.bucket.Wait(ctx)
		if err != nil {
			// Context cancelled - stop scheduling
			return
		}

		// Check if rate is too low (nearly 0)
		currentRate := float64(e.currentRate.Load()) / 1000.0
		if currentRate < 0.01 {
			// Rate too low, wait a bit and check again
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		// Try to get a VU from the pool
		vu := e.getVU(ctx)
		if vu == nil {
			// Context cancelled while waiting for VU
			return
		}

		// Schedule iteration on the VU
		e.wg.Add(1)
		go e.runIteration(ctx, vu)
	}
}

// getVU gets an available VU from the pool, spawning a new one if needed.
func (e *RampingArrivalRate) getVU(ctx context.Context) *v2.VirtualUser {
	// Try to get from pool (non-blocking)
	select {
	case vu := <-e.vuPool:
		return vu
	default:
		// Pool empty - try to spawn a new VU
	}

	// Check if we can spawn more VUs
	e.vuPoolMu.Lock()
	currentCount := int(e.currentVUs.Load())
	if currentCount < e.config.MaxVUs {
		// Spawn new VU
		vu := e.scheduler.SpawnVU()
		e.allVUs = append(e.allVUs, vu)
		e.currentVUs.Add(1)
		e.metrics.SetActiveVUs(int(e.currentVUs.Load()))
		e.vuPoolMu.Unlock()
		return vu
	}
	e.vuPoolMu.Unlock()

	// At max VUs - wait for one to become available
	select {
	case <-ctx.Done():
		return nil
	case vu := <-e.vuPool:
		return vu
	}
}

// returnVU returns a VU to the pool.
func (e *RampingArrivalRate) returnVU(vu *v2.VirtualUser) {
	// Check if VU is still usable
	state := vu.GetState()
	if state == v2.VUStateStopping || state == v2.VUStateStopped {
		return
	}

	// Return to pool (non-blocking - drop if full)
	select {
	case e.vuPool <- vu:
	default:
		// Pool full - this shouldn't happen with proper sizing
	}
}

// runIteration runs a single iteration on a VU.
func (e *RampingArrivalRate) runIteration(ctx context.Context, vu *v2.VirtualUser) {
	defer e.wg.Done()
	defer e.returnVU(vu)

	// Run the iteration
	err := vu.RunIteration(ctx)
	if err != nil {
		// Error logged but not fatal - iteration still counts
	}

	e.iterations.Add(1)
}

// gracefulShutdown waits for all VUs to finish their current iteration.
func (e *RampingArrivalRate) gracefulShutdown() {
	e.vuPoolMu.Lock()
	// Request all VUs to stop
	for _, vu := range e.allVUs {
		vu.RequestStop()
	}
	e.vuPoolMu.Unlock()

	// Wait with timeout
	graceful := e.config.GracefulStop
	if graceful == 0 {
		graceful = 30 * time.Second
	}

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All iterations completed
	case <-time.After(graceful):
		// Timeout expired
	}
}

// GetProgress returns current progress (0.0 to 1.0).
func (e *RampingArrivalRate) GetProgress() float64 {
	if !e.running.Load() {
		if e.startTime.IsZero() {
			return 0.0
		}
		return 1.0
	}

	totalDuration := e.config.TotalDuration()
	if totalDuration == 0 {
		return 1.0
	}

	elapsed := time.Since(e.startTime)
	progress := float64(elapsed) / float64(totalDuration)
	if progress > 1.0 {
		progress = 1.0
	}
	return progress
}

// GetActiveVUs returns current active VU count.
func (e *RampingArrivalRate) GetActiveVUs() int {
	return int(e.currentVUs.Load())
}

// GetStats returns executor statistics.
func (e *RampingArrivalRate) GetStats() *Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var elapsed time.Duration
	if !e.startTime.IsZero() {
		elapsed = time.Since(e.startTime)
	}

	stageIdx := int(e.currentStage.Load())
	stageName := ""
	targetRate := 0.0
	if stageIdx < len(e.config.Stages) {
		stageName = e.config.Stages[stageIdx].Name
		targetRate = float64(e.config.Stages[stageIdx].Target)
	}

	currentRate := float64(e.currentRate.Load()) / 1000.0

	return &Stats{
		StartTime:        e.startTime,
		CurrentTime:      time.Now(),
		Elapsed:          elapsed,
		TotalDuration:    e.config.TotalDuration(),
		ActiveVUs:        int(e.currentVUs.Load()),
		TargetVUs:        e.config.MaxVUs,
		Iterations:       e.iterations.Load(),
		CurrentStage:     stageIdx,
		CurrentStageName: stageName,
		TotalStages:      len(e.config.Stages),
		CurrentRate:      currentRate,
		TargetRate:       targetRate,
	}
}

// Stop gracefully stops the executor.
func (e *RampingArrivalRate) Stop(ctx context.Context) error {
	e.cancelMu.Lock()
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
	e.cancelMu.Unlock()

	e.gracefulShutdown()
	return nil
}

// Ensure RampingArrivalRate implements Executor
var _ Executor = (*RampingArrivalRate)(nil)
