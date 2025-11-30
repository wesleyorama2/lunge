package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// TimeBucketStore stores time-bucketed metrics in a ring buffer.
//
// It provides:
// - Continuous time-series data (even when no requests complete)
// - Efficient O(1) append and bounded memory usage
// - Thread-safe access from multiple goroutines
//
// The store maintains a ring buffer of configurable size, automatically
// discarding old buckets when the buffer is full.
type TimeBucketStore struct {
	buckets    []*TimeBucket
	head       int // Next write position
	count      int // Current number of buckets
	maxBuckets int
	mu         sync.RWMutex

	// For interval calculation
	lastBucketTime     time.Time
	lastBucketRequests int64

	// Current interval accumulator (lock-free updates)
	currentRequests  atomic.Int64
	currentSuccesses atomic.Int64
	currentFailures  atomic.Int64
	currentBytes     atomic.Int64
}

// NewTimeBucketStore creates a new time bucket store.
//
// Parameters:
//   - maxBuckets: Maximum number of buckets to retain (ring buffer size)
//
// For a 1-hour test with 1-second buckets, use maxBuckets=3600.
func NewTimeBucketStore(maxBuckets int) *TimeBucketStore {
	if maxBuckets <= 0 {
		maxBuckets = 3600 // Default: 1 hour of data
	}

	return &TimeBucketStore{
		buckets:        make([]*TimeBucket, maxBuckets),
		maxBuckets:     maxBuckets,
		lastBucketTime: time.Now(),
	}
}

// RecordRequest records a request into the current interval accumulator.
//
// This method is lock-free using atomic operations, making it safe
// for high-concurrency scenarios without blocking.
//
// Parameters:
//   - success: true if the request succeeded (status < 400 and no error)
//   - bytes: number of bytes received
func (tbs *TimeBucketStore) RecordRequest(success bool, bytes int64) {
	tbs.currentRequests.Add(1)
	tbs.currentBytes.Add(bytes)

	if success {
		tbs.currentSuccesses.Add(1)
	} else {
		tbs.currentFailures.Add(1)
	}
}

// CreateBucket creates a new bucket with the current metrics.
//
// This method is called by the background emitter (typically every second).
// It captures the current state and resets the interval accumulators.
func (tbs *TimeBucketStore) CreateBucket(
	totalRequests, totalSuccesses, totalFailures, totalBytes int64,
	latencies LatencyPercentiles,
	activeVUs int,
	phase Phase,
) *TimeBucket {
	tbs.mu.Lock()
	defer tbs.mu.Unlock()

	now := time.Now()

	// Calculate interval metrics
	intervalRequests := tbs.currentRequests.Swap(0)
	intervalSuccesses := tbs.currentSuccesses.Swap(0)
	intervalFailures := tbs.currentFailures.Swap(0)
	tbs.currentBytes.Swap(0) // Reset but don't use for interval

	// Calculate RPS for this interval
	intervalDuration := now.Sub(tbs.lastBucketTime).Seconds()
	if intervalDuration <= 0 {
		intervalDuration = 1.0
	}
	intervalRPS := float64(intervalRequests) / intervalDuration

	// Calculate error rate for this interval
	intervalErrorRate := 0.0
	if intervalRequests > 0 {
		intervalErrorRate = float64(intervalFailures) / float64(intervalRequests)
	}

	bucket := &TimeBucket{
		Timestamp:         now,
		TotalRequests:     totalRequests,
		TotalSuccesses:    totalSuccesses,
		TotalFailures:     totalFailures,
		TotalBytes:        totalBytes,
		IntervalRequests:  intervalRequests,
		IntervalRPS:       intervalRPS,
		LatencyMin:        latencies.Min,
		LatencyMax:        latencies.Max,
		LatencyP50:        latencies.P50,
		LatencyP90:        latencies.P90,
		LatencyP95:        latencies.P95,
		LatencyP99:        latencies.P99,
		ActiveVUs:         activeVUs,
		Phase:             phase,
		IntervalErrorRate: intervalErrorRate,
	}

	// Add to ring buffer
	tbs.buckets[tbs.head] = bucket
	tbs.head = (tbs.head + 1) % tbs.maxBuckets
	if tbs.count < tbs.maxBuckets {
		tbs.count++
	}

	// Update for next interval
	tbs.lastBucketTime = now
	tbs.lastBucketRequests = totalRequests
	_ = intervalSuccesses // Used in error rate calculation

	return bucket
}

// GetBuckets returns a copy of all buckets in chronological order.
//
// The returned slice is a copy, safe to use without holding locks.
func (tbs *TimeBucketStore) GetBuckets() []*TimeBucket {
	tbs.mu.RLock()
	defer tbs.mu.RUnlock()

	if tbs.count == 0 {
		return nil
	}

	result := make([]*TimeBucket, tbs.count)

	if tbs.count < tbs.maxBuckets {
		// Buffer not yet full - buckets are in order from 0 to count-1
		for i := 0; i < tbs.count; i++ {
			result[i] = tbs.buckets[i]
		}
	} else {
		// Buffer is full - need to read in order from head to head-1
		for i := 0; i < tbs.count; i++ {
			idx := (tbs.head + i) % tbs.maxBuckets
			result[i] = tbs.buckets[idx]
		}
	}

	return result
}

// GetBucketsForPhase returns buckets for a specific phase.
//
// Useful for calculating phase-specific metrics (e.g., steady-state RPS).
func (tbs *TimeBucketStore) GetBucketsForPhase(phase Phase) []*TimeBucket {
	allBuckets := tbs.GetBuckets()
	result := make([]*TimeBucket, 0)

	for _, b := range allBuckets {
		if b.Phase == phase {
			result = append(result, b)
		}
	}

	return result
}

// GetRecentBuckets returns the N most recent buckets.
func (tbs *TimeBucketStore) GetRecentBuckets(n int) []*TimeBucket {
	tbs.mu.RLock()
	defer tbs.mu.RUnlock()

	if n > tbs.count {
		n = tbs.count
	}
	if n == 0 {
		return nil
	}

	result := make([]*TimeBucket, n)

	// Read from most recent backwards
	for i := 0; i < n; i++ {
		// head-1 is most recent, head-2 is second most recent, etc.
		idx := (tbs.head - 1 - i + tbs.maxBuckets) % tbs.maxBuckets
		result[n-1-i] = tbs.buckets[idx] // Reverse to get chronological order
	}

	return result
}

// GetLatestBucket returns the most recent bucket, or nil if none.
func (tbs *TimeBucketStore) GetLatestBucket() *TimeBucket {
	tbs.mu.RLock()
	defer tbs.mu.RUnlock()

	if tbs.count == 0 {
		return nil
	}

	idx := (tbs.head - 1 + tbs.maxBuckets) % tbs.maxBuckets
	return tbs.buckets[idx]
}

// Count returns the current number of buckets stored.
func (tbs *TimeBucketStore) Count() int {
	tbs.mu.RLock()
	defer tbs.mu.RUnlock()
	return tbs.count
}

// Reset clears all buckets and resets counters.
func (tbs *TimeBucketStore) Reset() {
	tbs.mu.Lock()
	defer tbs.mu.Unlock()

	tbs.buckets = make([]*TimeBucket, tbs.maxBuckets)
	tbs.head = 0
	tbs.count = 0
	tbs.lastBucketTime = time.Now()
	tbs.lastBucketRequests = 0

	tbs.currentRequests.Store(0)
	tbs.currentSuccesses.Store(0)
	tbs.currentFailures.Store(0)
	tbs.currentBytes.Store(0)
}

// CalculateSteadyStateRPS calculates the average RPS during steady-state phase.
//
// This provides a more accurate RPS measurement than overall average,
// as it excludes ramp-up and ramp-down periods.
func (tbs *TimeBucketStore) CalculateSteadyStateRPS() (float64, int) {
	steadyBuckets := tbs.GetBucketsForPhase(PhaseSteady)
	if len(steadyBuckets) == 0 {
		return 0, 0
	}

	var totalRequests int64
	for _, b := range steadyBuckets {
		totalRequests += b.IntervalRequests
	}

	// Total time is number of buckets * 1 second (bucket interval)
	avgRPS := float64(totalRequests) / float64(len(steadyBuckets))
	return avgRPS, len(steadyBuckets)
}
