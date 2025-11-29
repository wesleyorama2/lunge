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

// RampingVUs ramps VU count up and down according to stages.
//
// This executor smoothly interpolates VU counts between stages,
// avoiding step-wise VU changes that cause jarring throughput variations.
//
// Use cases:
//   - Realistic traffic simulation (morning ramp-up, evening ramp-down)
//   - Finding the breaking point of a system
//   - Stress testing with gradual load increase
//
// Example stages:
//
//	stages:
//	  - duration: 30s
//	    target: 10     # Ramp from 0 to 10 VUs over 30s
//	  - duration: 2m
//	    target: 10     # Stay at 10 VUs for 2 minutes
//	  - duration: 30s
//	    target: 0      # Ramp down to 0 VUs over 30s
type RampingVUs struct {
	config    *Config
	scheduler *v2.VUScheduler
	metrics   *metrics.Engine

	// State
	startTime    time.Time
	activeVUs    atomic.Int32
	targetVUs    atomic.Int32
	iterations   atomic.Int64
	currentStage atomic.Int32
	running      atomic.Bool

	// Cancellation
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup

	// VU tracking
	vus   []*v2.VirtualUser
	vusMu sync.Mutex

	// Stats
	mu sync.RWMutex
}

// NewRampingVUs creates a new ramping VUs executor.
func NewRampingVUs() *RampingVUs {
	return &RampingVUs{
		vus: make([]*v2.VirtualUser, 0),
	}
}

// Type returns the executor type.
func (e *RampingVUs) Type() Type {
	return TypeRampingVUs
}

// Init initializes the executor with configuration.
func (e *RampingVUs) Init(ctx context.Context, config *Config) error {
	if config.Type != TypeRampingVUs {
		return fmt.Errorf("invalid config type: expected %s, got %s", TypeRampingVUs, config.Type)
	}

	if err := config.Validate(); err != nil {
		return err
	}

	e.config = config
	return nil
}

// Run starts the executor and blocks until completion.
func (e *RampingVUs) Run(ctx context.Context, scheduler *v2.VUScheduler, metricsEngine *metrics.Engine) error {
	e.scheduler = scheduler
	e.metrics = metricsEngine
	e.running.Store(true)
	e.startTime = time.Now()

	// Calculate total duration from stages
	totalDuration := e.config.TotalDuration()

	// Create cancellable context with duration timeout
	runCtx, cancel := context.WithTimeout(ctx, totalDuration)
	e.cancelFunc = cancel
	defer cancel()

	// Start VU controller (adjusts VU count smoothly)
	controllerDone := make(chan struct{})
	go func() {
		e.vuController(runCtx)
		close(controllerDone)
	}()

	// Wait for context to complete or be cancelled
	<-runCtx.Done()

	// Wait for controller to finish
	<-controllerDone

	// Graceful shutdown - wait for VUs to finish current iteration
	e.gracefulShutdown()

	// Mark as done
	e.metrics.SetPhase(metrics.PhaseDone)
	e.running.Store(false)

	return nil
}

// vuController adjusts VU count according to stages.
func (e *RampingVUs) vuController(ctx context.Context) {
	// Adjust VUs every 100ms for smooth ramping
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			targetVUs := e.calculateTargetVUs()
			e.targetVUs.Store(int32(targetVUs))
			e.adjustVUs(ctx, targetVUs)
			e.updatePhase()
		}
	}
}

// calculateTargetVUs calculates the target VU count based on elapsed time.
func (e *RampingVUs) calculateTargetVUs() int {
	elapsed := time.Since(e.startTime)

	// Find current stage and interpolate target
	var stageStart time.Duration
	prevTarget := 0

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
			targetVUs := float64(prevTarget) + float64(stage.Target-prevTarget)*stageProgress
			return int(targetVUs + 0.5) // Round to nearest
		}

		prevTarget = stage.Target
		stageStart = stageEnd
	}

	// Past all stages - return last target
	if len(e.config.Stages) > 0 {
		return e.config.Stages[len(e.config.Stages)-1].Target
	}
	return 0
}

// adjustVUs adjusts the VU count to match the target.
func (e *RampingVUs) adjustVUs(ctx context.Context, targetVUs int) {
	e.vusMu.Lock()
	defer e.vusMu.Unlock()

	currentVUs := len(e.vus)

	if targetVUs > currentVUs {
		// Spawn new VUs
		for i := currentVUs; i < targetVUs; i++ {
			vu := e.scheduler.SpawnVU()
			e.vus = append(e.vus, vu)
			e.wg.Add(1)
			go e.runVU(ctx, vu)
		}
	} else if targetVUs < currentVUs {
		// Stop excess VUs (from the end)
		for i := currentVUs - 1; i >= targetVUs; i-- {
			e.vus[i].RequestStop()
		}
		// Remove stopped VUs from slice
		e.vus = e.vus[:targetVUs]
	}

	// Update metrics
	e.metrics.SetActiveVUs(targetVUs)
}

// updatePhase updates the metrics phase based on current stage.
func (e *RampingVUs) updatePhase() {
	stageIdx := int(e.currentStage.Load())
	if stageIdx >= len(e.config.Stages) {
		return
	}

	stage := e.config.Stages[stageIdx]

	// Determine phase based on stage characteristics
	if stageIdx == 0 && stage.Target > 0 {
		// First stage ramping up
		e.metrics.SetPhase(metrics.PhaseRampUp)
	} else if stageIdx == len(e.config.Stages)-1 && stage.Target == 0 {
		// Last stage ramping down
		e.metrics.SetPhase(metrics.PhaseRampDown)
	} else {
		// Check if we're ramping or steady
		prevTarget := 0
		if stageIdx > 0 {
			prevTarget = e.config.Stages[stageIdx-1].Target
		}

		if stage.Target == prevTarget {
			e.metrics.SetPhase(metrics.PhaseSteady)
		} else if stage.Target > prevTarget {
			e.metrics.SetPhase(metrics.PhaseRampUp)
		} else {
			e.metrics.SetPhase(metrics.PhaseRampDown)
		}
	}
}

// runVU runs a single VU until stopped.
func (e *RampingVUs) runVU(ctx context.Context, vu *v2.VirtualUser) {
	defer e.wg.Done()
	defer vu.MarkStopped()

	e.activeVUs.Add(1)
	defer e.activeVUs.Add(-1)

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
			if ctx.Err() != nil || vu.GetState() == v2.VUStateStopping {
				return
			}
		}

		e.iterations.Add(1)

		// Apply pacing between iterations
		if e.config.Pacing != nil {
			e.applyPacing(ctx, vu)
		}
	}
}

// applyPacing waits between iterations.
func (e *RampingVUs) applyPacing(ctx context.Context, vu *v2.VirtualUser) {
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

// gracefulShutdown waits for all VUs to finish their current iteration.
func (e *RampingVUs) gracefulShutdown() {
	e.vusMu.Lock()
	// Request all VUs to stop
	for _, vu := range e.vus {
		vu.RequestStop()
	}
	e.vusMu.Unlock()

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
		// All VUs stopped
	case <-time.After(graceful):
		// Timeout expired
	}
}

// GetProgress returns current progress (0.0 to 1.0).
func (e *RampingVUs) GetProgress() float64 {
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
func (e *RampingVUs) GetActiveVUs() int {
	return int(e.activeVUs.Load())
}

// GetStats returns executor statistics.
func (e *RampingVUs) GetStats() *Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var elapsed time.Duration
	if !e.startTime.IsZero() {
		elapsed = time.Since(e.startTime)
	}

	stageIdx := int(e.currentStage.Load())
	stageName := ""
	if stageIdx < len(e.config.Stages) {
		stageName = e.config.Stages[stageIdx].Name
	}

	return &Stats{
		StartTime:        e.startTime,
		CurrentTime:      time.Now(),
		Elapsed:          elapsed,
		TotalDuration:    e.config.TotalDuration(),
		ActiveVUs:        int(e.activeVUs.Load()),
		TargetVUs:        int(e.targetVUs.Load()),
		Iterations:       e.iterations.Load(),
		CurrentStage:     stageIdx,
		CurrentStageName: stageName,
		TotalStages:      len(e.config.Stages),
	}
}

// Stop gracefully stops the executor.
func (e *RampingVUs) Stop(ctx context.Context) error {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	e.gracefulShutdown()
	return nil
}

// Ensure RampingVUs implements Executor
var _ Executor = (*RampingVUs)(nil)
