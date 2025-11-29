package metrics

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// MetricsEngine Benchmarks
// =============================================================================

// BenchmarkMetricsEngine_RecordLatency measures the performance of recording
// latency values in the HDR histogram.
//
// Success criteria: Should be fast enough for high-throughput scenarios
// (>100k ops/sec)
func BenchmarkMetricsEngine_RecordLatency(b *testing.B) {
	engine := NewEngine()
	defer engine.Stop()

	latencies := []time.Duration{
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		latency := latencies[i%len(latencies)]
		engine.RecordLatency(latency, "", true, 1024)
	}
}

// BenchmarkMetricsEngine_RecordLatency_Parallel measures concurrent latency recording.
//
// This is the primary use case - multiple VUs recording simultaneously.
func BenchmarkMetricsEngine_RecordLatency_Parallel(b *testing.B) {
	engine := NewEngine()
	defer engine.Stop()

	latencies := []time.Duration{
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			latency := latencies[i%len(latencies)]
			engine.RecordLatency(latency, "", true, 1024)
			i++
		}
	})
}

// BenchmarkMetricsEngine_RecordLatency_WithRequestName measures recording
// with per-request name tracking (more expensive operation).
func BenchmarkMetricsEngine_RecordLatency_WithRequestName(b *testing.B) {
	engine := NewEngine()
	defer engine.Stop()

	requestNames := []string{"login", "get-profile", "update-settings", "logout", "list-items"}
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		name := requestNames[i%len(requestNames)]
		latency := latencies[i%len(latencies)]
		engine.RecordLatency(latency, name, true, 1024)
	}
}

// BenchmarkMetricsEngine_GetSnapshot measures the cost of taking a metrics snapshot.
func BenchmarkMetricsEngine_GetSnapshot(b *testing.B) {
	engine := NewEngine()
	defer engine.Stop()

	// Pre-populate with data
	for i := 0; i < 10000; i++ {
		engine.RecordLatency(time.Duration(rand.Intn(100))*time.Millisecond, "", true, 1024)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = engine.GetSnapshot()
	}
}

// BenchmarkMetricsEngine_GetLatencyPercentiles measures percentile calculation.
func BenchmarkMetricsEngine_GetLatencyPercentiles(b *testing.B) {
	engine := NewEngine()
	defer engine.Stop()

	// Pre-populate with data
	for i := 0; i < 10000; i++ {
		engine.RecordLatency(time.Duration(rand.Intn(100))*time.Millisecond, "", true, 1024)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = engine.GetLatencyPercentiles()
	}
}

// =============================================================================
// TimeBucketStore Benchmarks
// =============================================================================

// BenchmarkTimeBucketStore_RecordRequest measures time bucket recording performance.
//
// Success criteria: Lock-free recording for high-throughput scenarios.
func BenchmarkTimeBucketStore_RecordRequest(b *testing.B) {
	store := NewTimeBucketStore(3600)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store.RecordRequest(true, 1024)
	}
}

// BenchmarkTimeBucketStore_RecordRequest_Parallel measures concurrent bucket recording.
func BenchmarkTimeBucketStore_RecordRequest_Parallel(b *testing.B) {
	store := NewTimeBucketStore(3600)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			store.RecordRequest(true, 1024)
		}
	})
}

// BenchmarkTimeBucketStore_CreateBucket measures bucket creation performance.
func BenchmarkTimeBucketStore_CreateBucket(b *testing.B) {
	store := NewTimeBucketStore(3600)

	latencies := LatencyPercentiles{
		Min: 1 * time.Millisecond,
		Max: 100 * time.Millisecond,
		P50: 10 * time.Millisecond,
		P90: 50 * time.Millisecond,
		P95: 75 * time.Millisecond,
		P99: 90 * time.Millisecond,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store.CreateBucket(int64(i), int64(i), 0, int64(i*1024), latencies, 10, PhaseSteady)
	}
}

// BenchmarkTimeBucketStore_GetBuckets measures retrieval of all buckets.
func BenchmarkTimeBucketStore_GetBuckets(b *testing.B) {
	store := NewTimeBucketStore(3600)

	// Pre-populate
	latencies := LatencyPercentiles{
		Min: 1 * time.Millisecond,
		Max: 100 * time.Millisecond,
		P50: 10 * time.Millisecond,
		P90: 50 * time.Millisecond,
		P95: 75 * time.Millisecond,
		P99: 90 * time.Millisecond,
	}
	for i := 0; i < 100; i++ {
		store.CreateBucket(int64(i), int64(i), 0, int64(i*1024), latencies, 10, PhaseSteady)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = store.GetBuckets()
	}
}

// =============================================================================
// Accuracy Tests (Success Criteria Verification)
// =============================================================================

// TestLatencyAccuracy verifies that P99 latency matches actual timing within 1%.
//
// Success criteria: P99 latency matches actual request timing within 1%
func TestLatencyAccuracy(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Generate known latencies with a specific distribution
	// We'll use exact values and verify the histogram reports them correctly
	numSamples := 10000

	// Generate latencies: 90% at 10ms, 9% at 50ms, 1% at 100ms
	// This gives us a known P99 of 100ms
	actualLatencies := make([]time.Duration, numSamples)
	for i := 0; i < numSamples; i++ {
		switch {
		case i < int(float64(numSamples)*0.90): // 90%
			actualLatencies[i] = 10 * time.Millisecond
		case i < int(float64(numSamples)*0.99): // 9%
			actualLatencies[i] = 50 * time.Millisecond
		default: // 1%
			actualLatencies[i] = 100 * time.Millisecond
		}
	}

	// Shuffle to simulate realistic arrival
	rand.Shuffle(len(actualLatencies), func(i, j int) {
		actualLatencies[i], actualLatencies[j] = actualLatencies[j], actualLatencies[i]
	})

	// Record all latencies
	for _, latency := range actualLatencies {
		engine.RecordLatency(latency, "", true, 1024)
	}

	// Get the reported percentiles
	snapshot := engine.GetSnapshot()

	// Calculate actual P99 from the raw data
	sorted := make([]time.Duration, len(actualLatencies))
	copy(sorted, actualLatencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx99 := int(math.Ceil(float64(numSamples)*0.99)) - 1
	actualP99 := sorted[idx99]

	// Verify accuracy
	reportedP99 := snapshot.Latency.P99

	// Calculate error percentage
	errorPercent := math.Abs(float64(reportedP99-actualP99)) / float64(actualP99) * 100

	t.Logf("Actual P99: %v, Reported P99: %v, Error: %.2f%%", actualP99, reportedP99, errorPercent)

	if errorPercent > 1.0 {
		t.Errorf("P99 accuracy exceeds 1%% threshold: %.2f%%", errorPercent)
	}
}

// TestLatencyAccuracy_HighPrecision tests latency accuracy with microsecond precision.
func TestLatencyAccuracy_HighPrecision(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Generate latencies with microsecond precision
	numSamples := 10000
	actualLatencies := make([]time.Duration, numSamples)

	// Create a distribution: mostly 1-10ms, with tail extending to 100ms
	for i := 0; i < numSamples; i++ {
		if i < int(float64(numSamples)*0.95) {
			// 95% between 1-10ms
			actualLatencies[i] = time.Duration(1000+rand.Intn(9000)) * time.Microsecond
		} else {
			// 5% between 10-100ms
			actualLatencies[i] = time.Duration(10000+rand.Intn(90000)) * time.Microsecond
		}
	}

	// Record all latencies
	for _, latency := range actualLatencies {
		engine.RecordLatency(latency, "", true, 1024)
	}

	// Get the reported percentiles
	snapshot := engine.GetSnapshot()

	// Calculate actual percentiles
	sorted := make([]time.Duration, len(actualLatencies))
	copy(sorted, actualLatencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	actualP50 := sorted[int(float64(numSamples)*0.50)-1]
	actualP90 := sorted[int(float64(numSamples)*0.90)-1]
	actualP95 := sorted[int(float64(numSamples)*0.95)-1]
	actualP99 := sorted[int(float64(numSamples)*0.99)-1]

	// Log comparisons
	t.Logf("P50: actual=%v, reported=%v", actualP50, snapshot.Latency.P50)
	t.Logf("P90: actual=%v, reported=%v", actualP90, snapshot.Latency.P90)
	t.Logf("P95: actual=%v, reported=%v", actualP95, snapshot.Latency.P95)
	t.Logf("P99: actual=%v, reported=%v", actualP99, snapshot.Latency.P99)

	// Check P99 accuracy (main success criterion)
	errorPercent := math.Abs(float64(snapshot.Latency.P99-actualP99)) / float64(actualP99) * 100
	if errorPercent > 1.0 {
		t.Errorf("P99 accuracy exceeds 1%% threshold: %.2f%%", errorPercent)
	}
}

// TestTimeSeriesContinuity verifies no gaps in time-series data.
//
// Success criteria: Time-series buckets show data for every second
func TestTimeSeriesContinuity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping time series continuity test in short mode")
	}

	engine := NewEngine()

	// Run for 10 seconds, recording some data
	testDuration := 10 * time.Second
	done := make(chan struct{})

	// Record data continuously
	go func() {
		deadline := time.Now().Add(testDuration)
		for time.Now().Before(deadline) {
			engine.RecordLatency(10*time.Millisecond, "", true, 1024)
			time.Sleep(10 * time.Millisecond) // ~100 RPS
		}
		close(done)
	}()

	<-done
	engine.Stop()

	// Check time series
	buckets := engine.GetTimeSeries()

	t.Logf("Total buckets: %d (expected ~%d)", len(buckets), int(testDuration.Seconds()))

	// Verify we have roughly the expected number of buckets
	expectedBuckets := int(testDuration.Seconds())
	minBuckets := expectedBuckets - 2 // Allow for startup/shutdown
	maxBuckets := expectedBuckets + 2

	if len(buckets) < minBuckets || len(buckets) > maxBuckets {
		t.Errorf("Expected ~%d buckets, got %d", expectedBuckets, len(buckets))
	}

	// Check for gaps (timestamps should be roughly 1 second apart)
	maxGap := 1500 * time.Millisecond // Allow up to 1.5s gap

	for i := 1; i < len(buckets); i++ {
		gap := buckets[i].Timestamp.Sub(buckets[i-1].Timestamp)
		if gap > maxGap {
			t.Errorf("Gap between buckets %d and %d exceeds threshold: %v", i-1, i, gap)
		}
	}

	t.Logf("Time series continuity check passed")
}

// =============================================================================
// Memory Tests (Success Criteria Verification)
// =============================================================================

// TestMemoryUsage_HighLoad verifies memory stays under acceptable limits.
//
// Success criteria: <100MB memory for 10-minute test at 1000 RPS
// Note: We run a shorter test here for CI, validating the pattern scales.
func TestMemoryUsage_HighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Force GC to get clean baseline
	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	engine := NewEngine()

	// Simulate high load for 30 seconds
	testDuration := 30 * time.Second
	targetRPS := 1000

	done := make(chan struct{})
	var iterations int64
	var mu sync.Mutex

	// Use multiple goroutines to simulate concurrent VUs
	numWorkers := 10
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deadline := time.Now().Add(testDuration)
			localIter := int64(0)

			sleepDuration := time.Duration(float64(time.Second) / float64(targetRPS) * float64(numWorkers))

			for time.Now().Before(deadline) {
				// Simulate recording a request
				latency := time.Duration(5+rand.Intn(95)) * time.Millisecond
				engine.RecordLatency(latency, "test_request", true, 1024)
				localIter++
				time.Sleep(sleepDuration)
			}

			mu.Lock()
			iterations += localIter
			mu.Unlock()
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	<-done
	engine.Stop()

	// Force GC and measure memory
	runtime.GC()

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory increase
	memIncrease := memAfter.Alloc - memBefore.Alloc
	memIncreaseMB := float64(memIncrease) / 1024 / 1024

	// Also check heap in use
	heapInUseMB := float64(memAfter.HeapInuse) / 1024 / 1024

	t.Logf("Iterations: %d", iterations)
	t.Logf("Memory increase: %.2f MB", memIncreaseMB)
	t.Logf("Heap in use: %.2f MB", heapInUseMB)
	t.Logf("Total allocations: %d", memAfter.Mallocs-memBefore.Mallocs)

	// For a 30-second test, memory should be well under scaled limit
	// (scaled from 100MB for 10 minutes: 100MB * (30s / 600s) = 5MB, but give margin)
	maxMemMB := 15.0
	if memIncreaseMB > maxMemMB {
		t.Errorf("Memory usage %.2f MB exceeds threshold %.2f MB", memIncreaseMB, maxMemMB)
	}
}

// BenchmarkMemoryAllocation measures per-request memory allocation.
func BenchmarkMemoryAllocation(b *testing.B) {
	engine := NewEngine()
	defer engine.Stop()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		latency := time.Duration(1+rand.Intn(100)) * time.Millisecond
		engine.RecordLatency(latency, "request", true, 1024)
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

// TestConcurrentMetricsAccess verifies thread-safety under high concurrency.
func TestConcurrentMetricsAccess(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	numGoroutines := 100
	iterationsPerGoroutine := 1000

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				latency := time.Duration(1+rand.Intn(100)) * time.Millisecond
				engine.RecordLatency(latency, "request", rand.Float32() > 0.05, 1024)
			}
		}()
	}

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				_ = engine.GetSnapshot()
				_ = engine.GetTimeSeries()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()

	// Verify data integrity
	snapshot := engine.GetSnapshot()
	expectedRequests := int64(numGoroutines * iterationsPerGoroutine)

	if snapshot.TotalRequests != expectedRequests {
		t.Errorf("Expected %d requests, got %d", expectedRequests, snapshot.TotalRequests)
	}

	t.Logf("Concurrent access test passed: %d requests recorded correctly", snapshot.TotalRequests)
}
