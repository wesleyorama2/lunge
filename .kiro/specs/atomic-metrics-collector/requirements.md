# Atomic Metrics Collector - Requirements

## Problem Statement

The current `DefaultMetricsCollector` uses a mutex lock on every `RecordRequest()` call, creating severe contention at high RPS. Testing shows:

- **Simple test script**: 30,000 RPS (using atomic counters)
- **Lunge with mutex-based collector**: 132 RPS (23x slower!)
- **Rate limiter efficiency**: 100% (proves rate limiter works perfectly)
- **Server capacity**: 4,000+ RPS (httpbin), 30,000+ RPS (Go test server)

**Root cause**: Single mutex lock serializes all metric recording across 1000 workers.

## Current Bottleneck

```go
func (d *DefaultMetricsCollector) RecordRequest(result *RequestResult) error {
    d.mu.Lock()  // ← BOTTLENECK: All 1000 workers contend for this lock
    defer d.mu.Unlock()
    
    d.totalRequests++
    d.successfulRequests++
    // ... more operations under lock
}
```

At 200 RPS target with 1000 workers, this creates massive lock contention.

## Success Criteria

1. **Performance**: Achieve 200+ RPS with 200 RPS target (100% throughput)
2. **Accuracy**: Maintain accurate metrics (within 1% of actual)
3. **Compatibility**: Drop-in replacement for existing `MetricsCollector` interface
4. **Safety**: Thread-safe with no data races
5. **Efficiency**: Minimal CPU overhead (<5% at 1000 RPS)

## Requirements

### Functional Requirements

1. **Atomic Counters**: Use `atomic.Int64` for hot-path metrics:
   - Total requests
   - Successful requests
   - Failed requests
   - Total bytes transferred

2. **Lock-Free Response Time Recording**: Use ring buffer with atomic write pointer
   - Fixed-size buffer (10,000 samples)
   - Lock-free writes
   - Periodic flush to analysis buffer

3. **Batched Cold-Path Operations**: Aggregate error details periodically
   - Error by status code
   - Error by type
   - Time series data points

4. **Backward Compatibility**: Implement existing `MetricsCollector` interface
   - `RecordRequest(result *RequestResult) error`
   - `RecordGenerationRate(rate float64)`
   - `GetSnapshot() *MetricsSnapshot`
   - `GetTimeSeries() *TimeSeriesData`
   - `Reset() error`

### Non-Functional Requirements

1. **Performance**:
   - Support 10,000+ RPS on modern hardware
   - <100ns per `RecordRequest()` call (hot path)
   - <1ms for `GetSnapshot()` (cold path)

2. **Memory**:
   - Fixed memory footprint
   - No unbounded growth
   - Efficient ring buffer usage

3. **Accuracy**:
   - Exact counts for requests/bytes
   - Statistical accuracy for response times (sampling acceptable)
   - Time series within 1-second granularity

## Out of Scope

- Real-time streaming metrics (batch collection is acceptable)
- Distributed metrics aggregation
- Persistent storage
- Custom metric types

## Constraints

1. Must work on Windows (current development platform)
2. Must integrate with existing performance engine
3. Cannot break existing tests
4. Must maintain current reporting format

## Dependencies

- Go 1.19+ (for `atomic` package improvements)
- Existing `MetricsSnapshot` and `TimeSeriesData` types
- Current reporting infrastructure

## Risks

1. **Complexity**: Atomic operations are harder to debug than mutexes
2. **Race Conditions**: Careful synchronization needed between hot/cold paths
3. **Testing**: Need comprehensive concurrency tests
4. **Migration**: Existing code depends on current collector

## Mitigation Strategies

1. **Incremental Migration**: Create new collector alongside old one
2. **Feature Flag**: Allow switching between implementations
3. **Extensive Testing**: Benchmark and race detector tests
4. **Documentation**: Clear comments on synchronization patterns
