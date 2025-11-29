# Performance Benchmarks for Lunge v2 Engine

This document describes the performance benchmarks for the v2 performance testing engine and how to run them.

## Overview

The v2 engine benchmarks verify that the engine meets its performance goals:

| Criterion | Target | Test |
|-----------|--------|------|
| Accurate Metrics | P99 latency within 1% of actual | `TestLatencyAccuracy` |
| Smooth Ramping | VU count changes gradually | `TestSmoothRamping` |
| Consistent Throughput | Arrival-rate maintains target RPS ±5% | `TestArrivalRateAccuracy` |
| Graceful Shutdown | VUs complete current iteration | `TestGracefulShutdown` |
| Continuous Data | Time buckets for every second | `TestTimeSeriesContinuity` |
| Memory Efficiency | <100MB for 10-min test at 1000 RPS | `TestMemoryUsage_HighLoad` |

## Benchmark Files

Benchmarks are organized by component:

- [`internal/performance/v2/metrics/benchmark_test.go`](../internal/performance/v2/metrics/benchmark_test.go) - Metrics engine and time bucket store
- [`internal/performance/v2/rate/benchmark_test.go`](../internal/performance/v2/rate/benchmark_test.go) - Rate limiter (leaky bucket)
- [`internal/performance/v2/engine/benchmark_test.go`](../internal/performance/v2/engine/benchmark_test.go) - VU, Scheduler, and Engine integration

## Running Benchmarks

### Run All Benchmarks

```bash
# Run all v2 benchmarks
go test -bench=. ./internal/performance/v2/...

# Run with memory profiling
go test -bench=. -benchmem ./internal/performance/v2/...

# Run with verbose output
go test -v -bench=. -benchmem ./internal/performance/v2/...
```

### Run Specific Benchmarks

```bash
# Metrics Engine benchmarks
go test -bench=BenchmarkMetricsEngine ./internal/performance/v2/metrics/

# Rate limiter benchmarks
go test -bench=BenchmarkLeakyBucket ./internal/performance/v2/rate/

# VU iteration benchmarks
go test -bench=BenchmarkVirtualUser ./internal/performance/v2/engine/

# High-load integration benchmark
go test -bench=BenchmarkEngine_HighLoad ./internal/performance/v2/engine/
```

### Run Accuracy Tests

```bash
# Latency accuracy (verifies P99 within 1%)
go test -v -run TestLatencyAccuracy ./internal/performance/v2/metrics/

# Arrival rate accuracy (verifies RPS within 5%)
go test -v -run TestArrivalRateAccuracy ./internal/performance/v2/rate/

# Time series continuity (verifies no gaps)
go test -v -run TestTimeSeriesContinuity ./internal/performance/v2/metrics/

# Smooth ramping (verifies no stepping artifacts)
go test -v -run TestSmoothRamping ./internal/performance/v2/rate/

# Graceful shutdown
go test -v -run TestGracefulShutdown ./internal/performance/v2/engine/
```

### Run Memory Test

```bash
# Memory usage test (requires non-short mode)
go test -v -run TestMemoryUsage_HighLoad ./internal/performance/v2/metrics/
```

Note: Some tests are skipped in `-short` mode. Run without `-short` for full validation.

## Benchmark Descriptions

### Metrics Benchmarks

#### `BenchmarkMetricsEngine_RecordLatency`
Measures the performance of recording latency values in the HDR histogram. This is the core operation called for every HTTP request.

**Expected**: >100k ops/sec with 0 allocations

#### `BenchmarkMetricsEngine_RecordLatency_Parallel`
Measures concurrent latency recording from multiple goroutines (simulating multiple VUs).

**Expected**: High throughput under concurrent access

#### `BenchmarkTimeBucketStore_RecordRequest`
Measures the lock-free atomic counter updates in the time bucket store.

**Expected**: Very high throughput (atomic operations only)

### Rate Limiter Benchmarks

#### `BenchmarkLeakyBucket_Wait`
Measures the overhead of the rate limiter decision-making.

**Expected**: Minimal overhead per iteration

#### `BenchmarkLeakyBucket_Next`
Measures just the timing calculation without sleeping.

**Expected**: <1µs per call

### VU Benchmarks

#### `BenchmarkVirtualUser_RunIteration`
Measures the overhead of running a complete VU iteration against a mock server.

**Expected**: Overhead should be minimal compared to actual HTTP latency

#### `BenchmarkEngine_HighLoad`
Integration benchmark simulating high-load conditions with multiple VUs.

**Expected**: Linear scaling with VU count

## Accuracy Tests

### `TestLatencyAccuracy`
Verifies that the HDR histogram accurately reports P99 latency. Records 10,000 requests with known latency distribution and verifies P99 is within 1% of actual.

**Pass Criterion**: Error < 1%

### `TestArrivalRateAccuracy`
Verifies that the leaky bucket rate limiter maintains the target RPS within tolerance. Runs for 5 seconds at target RPS and measures actual throughput.

**Pass Criterion**: Error < 5%

### `TestTimeSeriesContinuity`
Verifies that time series data has no gaps. Runs for 10 seconds and checks that buckets are created every second.

**Pass Criterion**: No gaps > 1.5 seconds

### `TestMemoryUsage_HighLoad`
Verifies memory usage stays within limits under high load. Simulates 1000 RPS for 30-60 seconds.

**Pass Criterion**: Memory increase < scaled threshold

## Expected Output

### Metrics Benchmarks

```
goos: windows
goarch: amd64
pkg: github.com/wesleyorama2/lunge/internal/performance/v2/metrics
cpu: AMD Ryzen 9 7950X3D 16-Core Processor
BenchmarkMetricsEngine_RecordLatency-32                    81492950        12.98 ns/op        0 B/op       0 allocs/op
BenchmarkMetricsEngine_RecordLatency_Parallel-32           14226790        83.58 ns/op        0 B/op       0 allocs/op
BenchmarkMetricsEngine_RecordLatency_WithRequestName-32    44091872        27.73 ns/op        0 B/op       0 allocs/op
BenchmarkMetricsEngine_GetSnapshot-32                          5793    202360 ns/op      209 B/op       1 allocs/op
BenchmarkMetricsEngine_GetLatencyPercentiles-32               17126     67211 ns/op        0 B/op       0 allocs/op
BenchmarkTimeBucketStore_RecordRequest-32                 254176651         4.636 ns/op      0 B/op       0 allocs/op
BenchmarkTimeBucketStore_RecordRequest_Parallel-32         34541716        34.80 ns/op       0 B/op       0 allocs/op
BenchmarkTimeBucketStore_CreateBucket-32                   12180997        97.97 ns/op     160 B/op       1 allocs/op
BenchmarkTimeBucketStore_GetBuckets-32                      4906557       244.0 ns/op      896 B/op       1 allocs/op
BenchmarkMemoryAllocation-32                               32121547        32.38 ns/op       0 B/op       0 allocs/op
```

### Rate Limiter Benchmarks

```
goos: windows
goarch: amd64
pkg: github.com/wesleyorama2/lunge/internal/performance/v2/rate
cpu: AMD Ryzen 9 7950X3D 16-Core Processor
BenchmarkLeakyBucket_Wait-32              4584       261650 ns/op      124 B/op       1 allocs/op
BenchmarkLeakyBucket_Next-32          56270925        20.73 ns/op        0 B/op       0 allocs/op
BenchmarkLeakyBucket_Next_Parallel-32 15375093        76.58 ns/op        0 B/op       0 allocs/op
BenchmarkLeakyBucket_SetRate-32      173438550         7.249 ns/op       0 B/op       0 allocs/op
BenchmarkLeakyBucket_GetRate-32      329162749         3.539 ns/op       0 B/op       0 allocs/op
BenchmarkLeakyBucket_Stats-32        340521166         3.483 ns/op       0 B/op       0 allocs/op
```

### Engine Benchmarks

```
goos: windows
goarch: amd64
pkg: github.com/wesleyorama2/lunge/internal/performance/v2/engine
cpu: AMD Ryzen 9 7950X3D 16-Core Processor
BenchmarkVirtualUser_RunIteration-32                      18601     57807 ns/op     6724 B/op      75 allocs/op
BenchmarkVirtualUser_RunIteration_Parallel-32             22554     48566 ns/op     8362 B/op      78 allocs/op
BenchmarkVUScheduler_SpawnVU-32                         2409093       462.5 ns/op    478 B/op       4 allocs/op
BenchmarkEngine_HighLoad-32                               22596     54412 ns/op     8435 B/op      78 allocs/op
BenchmarkEngine_WithMetricsRecording-32                   20184     58484 ns/op     6807 B/op      75 allocs/op
BenchmarkVirtualUser_VariableResolution-32                18237     65499 ns/op     8109 B/op     106 allocs/op
BenchmarkScheduler_SharedVsPerVUClient/SharedClient-32   144189      7813 ns/op     6657 B/op      73 allocs/op
BenchmarkScheduler_SharedVsPerVUClient/PerVUClient-32    263372      4132 ns/op     6623 B/op      73 allocs/op
```

### Accuracy Test Results

```
=== RUN   TestLatencyAccuracy
    benchmark_test.go: Actual P99: 50ms, Reported P99: 50.015ms, Error: 0.03%
--- PASS: TestLatencyAccuracy

=== RUN   TestTimeSeriesContinuity
    benchmark_test.go: Total buckets: 11 (expected ~10)
    benchmark_test.go: Time series continuity check passed
--- PASS: TestTimeSeriesContinuity

=== RUN   TestSmoothRamping
    benchmark_test.go: Rate progression: [10 20 30 40 50 60 70 80 90 100]
    benchmark_test.go: Smooth ramping verified - all steps are equal
--- PASS: TestSmoothRamping

=== RUN   TestGracefulShutdown
    benchmark_test.go: VUs properly shut down - graceful shutdown verified
--- PASS: TestGracefulShutdown

=== RUN   TestConcurrentMetricsAccess
    benchmark_test.go: Concurrent access test passed: 100000 requests recorded correctly
--- PASS: TestConcurrentMetricsAccess
```

## Profiling

### CPU Profiling

```bash
go test -bench=BenchmarkEngine_HighLoad -cpuprofile=cpu.prof ./internal/performance/v2/engine/
go tool pprof cpu.prof
```

### Memory Profiling

```bash
go test -bench=BenchmarkEngine_HighLoad -memprofile=mem.prof ./internal/performance/v2/engine/
go tool pprof mem.prof
```

### Trace

```bash
go test -bench=BenchmarkEngine_HighLoad -trace=trace.out ./internal/performance/v2/engine/
go tool trace trace.out
```

## Continuous Integration

These benchmarks should be run as part of CI to catch performance regressions:

```yaml
# Example GitHub Actions step
- name: Run Benchmarks
  run: |
    go test -bench=. -benchmem ./internal/performance/v2/... | tee benchmark_output.txt
    
- name: Run Accuracy Tests
  run: |
    go test -v -run "Test(Latency|ArrivalRate|TimeSeries|Graceful)" ./internal/performance/v2/...
```

## Interpreting Results

### Key Metrics

- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (0 is ideal for hot paths)
- **allocs/op**: Number of allocations per operation (0 is ideal for hot paths)

### Warning Signs

- Increasing allocations in hot paths (RecordLatency, RecordRequest)
- Significant degradation in parallel benchmarks
- Memory growth in long-running tests
- Accuracy tests failing with >threshold error

## Related Documentation

- [Performance Engine Architecture](PERFORMANCE_ENGINE_REWRITE.md)
- [V2 Migration Guide](V2-Migration-Guide.md)
- [Performance Testing Guide](Performance-Testing.md)