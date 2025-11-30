// Package metrics provides high-performance metrics collection for load testing.
//
// This package uses HDR histograms for accurate latency percentile calculations
// with O(1) time complexity, even at high throughput.
//
// # Key Features
//
//   - HDR histogram for accurate percentile calculation (p50, p90, p95, p99)
//   - Lock-free counter updates using atomic operations
//   - Time-series data collection with configurable bucket intervals
//   - Phase-aware metrics (warmup, ramp-up, steady-state, ramp-down)
//   - Background metric emission for continuous monitoring
//
// # Basic Usage
//
//	engine := metrics.NewEngine()
//	defer engine.Stop()
//
//	// Record request latencies
//	engine.RecordLatency(150*time.Millisecond, "GET /users", true, 1024)
//	engine.RecordLatency(200*time.Millisecond, "GET /users", true, 2048)
//	engine.RecordLatency(50*time.Millisecond, "GET /health", true, 64)
//
//	// Get current metrics snapshot
//	snapshot := engine.GetSnapshot()
//	fmt.Printf("Total Requests: %d\n", snapshot.TotalRequests)
//	fmt.Printf("RPS: %.2f\n", snapshot.RPS)
//	fmt.Printf("P95 Latency: %v\n", snapshot.Latency.P95)
//	fmt.Printf("Error Rate: %.2f%%\n", snapshot.ErrorRate*100)
//
// # Time-Series Data
//
// The engine automatically collects time-series data in 1-second buckets:
//
//	timeSeries := engine.GetTimeSeries()
//	for _, bucket := range timeSeries {
//	    fmt.Printf("[%s] RPS: %.2f, P95: %v\n",
//	        bucket.Timestamp.Format(time.RFC3339),
//	        bucket.IntervalRPS,
//	        bucket.LatencyP95)
//	}
//
// # Phase Tracking
//
// Track test phases for accurate steady-state metrics:
//
//	engine.SetPhase(metrics.PhaseWarmup)
//	// ... warmup requests ...
//	engine.SetPhase(metrics.PhaseSteady)
//	// ... main test requests ...
//	engine.SetPhase(metrics.PhaseCooldown)
//
// # Thread Safety
//
// Engine is safe for concurrent use. Counters use atomic operations,
// histograms use mutex protection, and the background emitter runs
// in its own goroutine.
package metrics
