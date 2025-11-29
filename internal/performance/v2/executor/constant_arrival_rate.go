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

// ConstantArrivalRate maintains a fixed iteration rate (open model).
//
// Unlike VU-based executors where throughput depends on response time,
// arrival-rate executors schedule iterations at a constant rate regardless
// of how long each iteration takes. This models real-world scenarios where
// users arrive at a constant rate.
//
// The executor uses a LeakyBucket to precisely schedule iterations and
// maintains a pool of VUs to execute them. If VUs are exhausted and
// iterations are backing up, the executor spawns more VUs up to MaxVUs.
//
// Use cases:
//   - Testing system behavior under constant load
//   - SLA validation (e.g., "system must handle 100 RPS")
//   - Capacity testing with predictable arrival patterns
//
// Example:
//
//	config:
//	  type: constant-arrival-rate
//	  rate: 100              # 100 iterations per second
//	  duration: 5m           # Run for 5 minutes
//	  preAllocatedVUs: 10    # Start with 10 VUs
//	  maxVUs: 50             # Scale up to 50 VUs if needed
type ConstantArrivalRate struct {
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
	startTime  time.Time
	iterations atomic.Int64
	running    atomic.Bool

	// Cancellation
	cancelMu   sync.Mutex // Protects cancelFunc
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup

	// Stats
	mu sync.RWMutex
}

// NewConstantArrivalRate creates a new constant arrival rate executor.
func NewConstantArrivalRate() *ConstantArrivalRate {
	return &ConstantArrivalRate{}
}

// Type returns the executor type.
func (e *ConstantArrivalRate) Type() Type {
	return TypeConstantArrivalRate
}

// Init initializes the executor with configuration.
func (e *ConstantArrivalRate) Init(ctx context.Context, config *Config) error {
	if config.Type != TypeConstantArrivalRate {
		return fmt.Errorf("invalid config type: expected %s, got %s", TypeConstantArrivalRate, config.Type)
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
func (e *ConstantArrivalRate) Run(ctx context.Context, scheduler *v2.VUScheduler, metricsEngine *metrics.Engine) error {
	e.scheduler = scheduler
	e.metrics = metricsEngine
	e.running.Store(true)
	e.startTime = time.Now()

	// Create rate limiter
	e.bucket = rate.NewLeakyBucket(e.config.Rate)

	// Initialize VU pool
	e.vuPool = make(chan *v2.VirtualUser, e.config.MaxVUs)
	e.allVUs = make([]*v2.VirtualUser, 0, e.config.MaxVUs)

	// Create cancellable context with duration timeout
	runCtx, cancel := context.WithTimeout(ctx, e.config.Duration)
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

	// Set phase to steady
	e.metrics.SetPhase(metrics.PhaseSteady)
	e.metrics.SetActiveVUs(e.config.PreAllocatedVUs)

	// Run the iteration scheduler
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

// iterationScheduler schedules iterations at the configured rate.
func (e *ConstantArrivalRate) iterationScheduler(ctx context.Context) {
	defer e.wg.Done()

	for {
		// Wait for next iteration slot
		err := e.bucket.Wait(ctx)
		if err != nil {
			// Context cancelled - stop scheduling
			return
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
func (e *ConstantArrivalRate) getVU(ctx context.Context) *v2.VirtualUser {
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
func (e *ConstantArrivalRate) returnVU(vu *v2.VirtualUser) {
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
func (e *ConstantArrivalRate) runIteration(ctx context.Context, vu *v2.VirtualUser) {
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
func (e *ConstantArrivalRate) gracefulShutdown() {
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
func (e *ConstantArrivalRate) GetProgress() float64 {
	if !e.running.Load() {
		if e.startTime.IsZero() {
			return 0.0
		}
		return 1.0
	}

	elapsed := time.Since(e.startTime)
	progress := float64(elapsed) / float64(e.config.Duration)
	if progress > 1.0 {
		progress = 1.0
	}
	return progress
}

// GetActiveVUs returns current active VU count.
func (e *ConstantArrivalRate) GetActiveVUs() int {
	return int(e.currentVUs.Load())
}

// GetStats returns executor statistics.
func (e *ConstantArrivalRate) GetStats() *Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var elapsed time.Duration
	if !e.startTime.IsZero() {
		elapsed = time.Since(e.startTime)
	}

	return &Stats{
		StartTime:     e.startTime,
		CurrentTime:   time.Now(),
		Elapsed:       elapsed,
		TotalDuration: e.config.Duration,
		ActiveVUs:     int(e.currentVUs.Load()),
		TargetVUs:     e.config.MaxVUs,
		Iterations:    e.iterations.Load(),
		CurrentRate:   e.config.Rate,
		TargetRate:    e.config.Rate,
	}
}

// Stop gracefully stops the executor.
func (e *ConstantArrivalRate) Stop(ctx context.Context) error {
	e.cancelMu.Lock()
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
	e.cancelMu.Unlock()

	e.gracefulShutdown()
	return nil
}

// Ensure ConstantArrivalRate implements Executor
var _ Executor = (*ConstantArrivalRate)(nil)
