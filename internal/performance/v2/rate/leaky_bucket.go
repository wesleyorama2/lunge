// Package rate provides rate limiting implementations for load testing.
package rate

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// LeakyBucket implements the leaky bucket algorithm for rate limiting.
//
// Unlike token bucket which focuses on "how many tokens are available",
// leaky bucket focuses on "when should the next iteration execute".
// This approach provides smoother rate limiting without bursting issues
// during rate changes (ramp-up/down).
//
// # Algorithm
//
// The leaky bucket maintains a virtual "drip" time that advances at a fixed rate.
// Each call to Next() returns when the next iteration should start.
// If we're behind schedule, iterations execute immediately.
// This naturally handles backpressure and rate changes.
//
// # Thread Safety
//
// LeakyBucket is safe for concurrent use from multiple goroutines.
//
// # Example
//
//	lb := NewLeakyBucket(100.0) // 100 iterations per second
//
//	for {
//	    nextTime := lb.Next()
//	    time.Sleep(time.Until(nextTime))
//	    // Execute iteration
//	}
type LeakyBucket struct {
	rate        float64   // Iterations per second
	lastDrip    time.Time // Last iteration timestamp
	accumulated float64   // Accumulated iterations (fractional)
	maxBurst    float64   // Maximum burst (typically 1.0 for strict timing)
	mu          sync.Mutex

	// Metrics
	totalIterations atomic.Int64 // Total iterations scheduled
	totalWaitTime   atomic.Int64 // Total wait time in nanoseconds
}

// NewLeakyBucket creates a new leaky bucket rate limiter.
//
// Parameters:
//   - rate: Target iterations per second (must be > 0)
//
// Returns a new LeakyBucket configured with the specified rate.
// The bucket starts with zero accumulated iterations, meaning
// the first call to Next() will return immediately.
func NewLeakyBucket(rate float64) *LeakyBucket {
	if rate <= 0 {
		rate = 1.0
	}
	return &LeakyBucket{
		rate:     rate,
		lastDrip: time.Now(),
		maxBurst: 1.0, // Default: no bursting
	}
}

// NewLeakyBucketWithBurst creates a leaky bucket with custom burst capacity.
//
// Parameters:
//   - rate: Target iterations per second
//   - maxBurst: Maximum accumulated iterations allowed (enables controlled bursting)
//
// A maxBurst > 1.0 allows the bucket to "store up" iterations when
// consumers are slow, which can then be executed in a burst.
func NewLeakyBucketWithBurst(rate float64, maxBurst float64) *LeakyBucket {
	if rate <= 0 {
		rate = 1.0
	}
	if maxBurst < 1.0 {
		maxBurst = 1.0
	}
	return &LeakyBucket{
		rate:     rate,
		lastDrip: time.Now(),
		maxBurst: maxBurst,
	}
}

// Next returns when the next iteration should start.
//
// This is the primary method for rate-limited iteration scheduling.
// The returned time may be in the past if we're behind schedule,
// indicating the iteration should execute immediately.
//
// Thread-safe: can be called from multiple goroutines.
func (lb *LeakyBucket) Next() time.Time {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(lb.lastDrip).Seconds()

	// Clamp elapsed to non-negative (can be negative if lastDrip is in the future)
	if elapsed < 0 {
		elapsed = 0
	}

	// Accumulate iterations based on elapsed time
	lb.accumulated += elapsed * lb.rate

	// Cap at max burst
	if lb.accumulated > lb.maxBurst {
		lb.accumulated = lb.maxBurst
	}

	if lb.accumulated >= 1.0 {
		// Can execute immediately - we've accumulated enough time
		lb.accumulated -= 1.0
		lb.lastDrip = now // Update lastDrip to now for immediate execution
		lb.totalIterations.Add(1)
		return now
	}

	// Calculate wait time for next iteration
	deficit := 1.0 - lb.accumulated
	waitSeconds := deficit / lb.rate
	lb.accumulated = 0 // Reset accumulated - we're scheduling a future iteration

	nextTime := now.Add(time.Duration(waitSeconds * float64(time.Second)))

	// KEY FIX: Set lastDrip to nextTime, not now
	// This prevents double-counting when we wake up at nextTime
	// Previously: lastDrip = now caused accumulated to reach 1.0 after sleeping,
	// which triggered an immediate extra iteration
	lb.lastDrip = nextTime

	lb.totalIterations.Add(1)
	lb.totalWaitTime.Add(int64(nextTime.Sub(now)))

	return nextTime
}

// Wait blocks until the next iteration should execute.
//
// This is a convenience method that combines Next() with sleeping.
// It respects context cancellation for graceful shutdown.
//
// Returns:
//   - nil if the wait completed successfully
//   - ctx.Err() if the context was cancelled
func (lb *LeakyBucket) Wait(ctx context.Context) error {
	nextTime := lb.Next()

	waitDuration := time.Until(nextTime)
	if waitDuration <= 0 {
		// Execute immediately
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil
	}
}

// SetRate updates the target rate.
//
// This method is designed for smooth rate transitions during ramp-up/down.
// Unlike token bucket, changing the rate does NOT cause accumulated
// iterations to burst - the accumulated value is reset to zero.
//
// Thread-safe: can be called while other goroutines use Wait() or Next().
func (lb *LeakyBucket) SetRate(rate float64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if rate <= 0 {
		rate = 1.0
	}

	// Don't carry over accumulated iterations when rate changes
	// This prevents bursting during ramp-down
	lb.rate = rate
	lb.accumulated = 0
	lb.lastDrip = time.Now()
}

// GetRate returns the current target rate in iterations per second.
func (lb *LeakyBucket) GetRate() float64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.rate
}

// SetMaxBurst sets the maximum burst size.
//
// A burst size of 1.0 means no bursting (strict timing).
// Higher values allow accumulated iterations during slow periods
// to be executed in bursts when capacity is available.
func (lb *LeakyBucket) SetMaxBurst(burst float64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if burst < 1.0 {
		burst = 1.0
	}
	lb.maxBurst = burst
}

// GetMaxBurst returns the maximum burst size.
func (lb *LeakyBucket) GetMaxBurst() float64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.maxBurst
}

// Stats returns statistics about the leaky bucket's operation.
func (lb *LeakyBucket) Stats() LeakyBucketStats {
	lb.mu.Lock()
	rate := lb.rate
	accumulated := lb.accumulated
	maxBurst := lb.maxBurst
	lb.mu.Unlock()

	return LeakyBucketStats{
		Rate:            rate,
		Accumulated:     accumulated,
		MaxBurst:        maxBurst,
		TotalIterations: lb.totalIterations.Load(),
		TotalWaitTime:   time.Duration(lb.totalWaitTime.Load()),
	}
}

// Reset resets the leaky bucket to its initial state.
//
// This clears accumulated iterations and resets the drip time.
// Useful when reusing a bucket for a new test phase.
func (lb *LeakyBucket) Reset() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.accumulated = 0
	lb.lastDrip = time.Now()
	lb.totalIterations.Store(0)
	lb.totalWaitTime.Store(0)
}

// LeakyBucketStats contains statistics about the leaky bucket.
type LeakyBucketStats struct {
	Rate            float64       `json:"rate"`            // Current rate in iterations/second
	Accumulated     float64       `json:"accumulated"`     // Currently accumulated iterations
	MaxBurst        float64       `json:"maxBurst"`        // Maximum burst size
	TotalIterations int64         `json:"totalIterations"` // Total iterations scheduled
	TotalWaitTime   time.Duration `json:"totalWaitTime"`   // Total time spent waiting
}
