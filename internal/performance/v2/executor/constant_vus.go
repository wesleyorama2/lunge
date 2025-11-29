package executor

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// ConstantVUs runs a fixed number of VUs for a specified duration.
//
// This is the simplest executor: spawn N VUs and let them run iterations
// until the duration expires. Each VU runs as fast as it can (closed model),
// optionally with pacing between iterations.
//
// Use cases:
//   - Basic load testing
//   - Determining max throughput for N concurrent users
//   - Simple soak testing
type ConstantVUs struct {
	config    *Config
	scheduler *v2.VUScheduler
	metrics   *metrics.Engine

	// State
	startTime  time.Time
	activeVUs  atomic.Int32
	iterations atomic.Int64
	running    atomic.Bool

	// Cancellation
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup

	// Stats
	mu sync.RWMutex
}

// NewConstantVUs creates a new constant VUs executor.
func NewConstantVUs() *ConstantVUs {
	return &ConstantVUs{}
}

// Type returns the executor type.
func (e *ConstantVUs) Type() Type {
	return TypeConstantVUs
}

// Init initializes the executor with configuration.
func (e *ConstantVUs) Init(ctx context.Context, config *Config) error {
	if config.Type != TypeConstantVUs {
		return fmt.Errorf("invalid config type: expected %s, got %s", TypeConstantVUs, config.Type)
	}

	if err := config.Validate(); err != nil {
		return err
	}

	e.config = config
	return nil
}

// Run starts the executor and blocks until completion.
func (e *ConstantVUs) Run(ctx context.Context, scheduler *v2.VUScheduler, metricsEngine *metrics.Engine) error {
	e.scheduler = scheduler
	e.metrics = metricsEngine
	e.running.Store(true)
	e.startTime = time.Now()

	// Create cancellable context with duration timeout
	runCtx, cancel := context.WithTimeout(ctx, e.config.Duration)
	e.cancelFunc = cancel
	defer cancel()

	// Set phase to steady (constant VUs has no ramp)
	e.metrics.SetPhase(metrics.PhaseSteady)

	// Spawn all VUs
	for i := 0; i < e.config.VUs; i++ {
		vu := scheduler.SpawnVU()
		e.wg.Add(1)
		go e.runVU(runCtx, vu)
	}

	// Wait for all VUs to complete
	e.wg.Wait()

	// Mark as done
	e.metrics.SetPhase(metrics.PhaseDone)
	e.running.Store(false)

	return nil
}

// runVU runs a single VU until the context is cancelled.
func (e *ConstantVUs) runVU(ctx context.Context, vu *v2.VirtualUser) {
	defer e.wg.Done()
	defer vu.MarkStopped()

	e.activeVUs.Add(1)
	e.metrics.SetActiveVUs(int(e.activeVUs.Load())) // Update metrics engine
	defer func() {
		e.activeVUs.Add(-1)
		e.metrics.SetActiveVUs(int(e.activeVUs.Load())) // Update metrics engine
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check if VU was stopped
		if vu.GetState() == v2.VUStateStopping || vu.GetState() == v2.VUStateStopped {
			return
		}

		// Run one iteration
		err := vu.RunIteration(ctx)
		if err != nil {
			// Context cancelled or VU stopping - exit gracefully
			if ctx.Err() != nil || vu.GetState() == v2.VUStateStopping {
				return
			}
			// Other errors - continue to next iteration
		}

		e.iterations.Add(1)

		// Apply pacing between iterations
		if e.config.Pacing != nil {
			e.applyPacing(ctx)
		}
	}
}

// applyPacing waits between iterations according to pacing config.
func (e *ConstantVUs) applyPacing(ctx context.Context) {
	if e.config.Pacing == nil || e.config.Pacing.Type == PacingNone {
		return
	}

	var wait time.Duration
	switch e.config.Pacing.Type {
	case PacingConstant:
		wait = e.config.Pacing.Duration
	case PacingRandom:
		diff := e.config.Pacing.Max - e.config.Pacing.Min
		if diff > 0 {
			wait = e.config.Pacing.Min + time.Duration(rand.Int63n(int64(diff)))
		} else {
			wait = e.config.Pacing.Min
		}
	}

	if wait > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
}

// GetProgress returns current progress (0.0 to 1.0).
func (e *ConstantVUs) GetProgress() float64 {
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
func (e *ConstantVUs) GetActiveVUs() int {
	return int(e.activeVUs.Load())
}

// GetStats returns executor statistics.
func (e *ConstantVUs) GetStats() *Stats {
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
		ActiveVUs:     int(e.activeVUs.Load()),
		TargetVUs:     e.config.VUs,
		Iterations:    e.iterations.Load(),
	}
}

// Stop gracefully stops the executor.
func (e *ConstantVUs) Stop(ctx context.Context) error {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	// Wait for VUs with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	graceful := e.config.GracefulStop
	if graceful == 0 {
		graceful = 30 * time.Second
	}

	select {
	case <-done:
		return nil
	case <-time.After(graceful):
		return fmt.Errorf("graceful stop timeout after %v", graceful)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Ensure ConstantVUs implements Executor
var _ Executor = (*ConstantVUs)(nil)
