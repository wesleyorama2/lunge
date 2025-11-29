# Atomic Metrics Collector - Design

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   AtomicMetricsCollector                     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  HOT PATH (Lock-Free)                                        │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Atomic Counters                                     │    │
│  │  - totalRequests (atomic.Int64)                     │    │
│  │  - successfulRequests (atomic.Int64)                │    │
│  │  - failedRequests (atomic.Int64)                    │    │
│  │  - totalBytes (atomic.Int64)                        │    │
│  └────────────────────────────────────────────────────┘    │
│                                                               │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Lock-Free Ring Buffer                               │    │
│  │  - responseTimes []time.Duration                    │    │
│  │  - writePos (atomic.Uint64)                         │    │
│  │  - Fixed size: 10,000 samples                       │    │
│  └────────────────────────────────────────────────────┘    │
│                                                               │
│  COLD PATH (Mutex Protected)                                 │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Error Details (map[int]int64)                       │    │
│  │ Time Series ([]TimeSeriesPoint)                     │    │
│  │ Aggregated Stats                                     │    │
│  └────────────────────────────────────────────────────┘    │
│                                                               │
│  BACKGROUND FLUSH                                             │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Periodic aggregation (every 100ms)                  │    │
│  │  - Flush ring buffer to analysis                    │    │
│  │  - Calculate percentiles                            │    │
│  │  - Update time series                               │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. Atomic Counters (Hot Path)

```go
type AtomicMetricsCollector struct {
    // Hot path - no locks
    totalRequests      atomic.Int64
    successfulRequests atomic.Int64
    failedRequests     atomic.Int64
    totalBytes         atomic.Int64
    
    // Ring buffer for response times
    responseBuffer *LockFreeRingBuffer
    
    // Cold path - mutex protected
    mu             sync.RWMutex
    errorsByStatus map[int]int64
    errorsByType   map[string]int64
    timeSeries     []TimeSeriesPoint
    
    // Background flush
    flushTicker *time.Ticker
    stopChan    chan struct{}
}
```

**Key Design Decisions**:
- Use `atomic.Int64` for all counters (lock-free increment)
- Separate hot path (atomic) from cold path (mutex)
- Workers never block on metrics recording

### 2. Lock-Free Ring Buffer

```go
type LockFreeRingBuffer struct {
    buffer   []time.Duration
    writePos atomic.Uint64
    size     uint64
}

func (rb *LockFreeRingBuffer) Write(duration time.Duration) {
    pos := rb.writePos.Add(1) - 1
    index := pos % rb.size
    rb.buffer[index] = duration
}
```

**Key Design Decisions**:
- Fixed-size circular buffer (no allocations)
- Atomic write position (lock-free)
- Overwrite old samples when full (acceptable for statistics)
- Read operations happen in background flush (no contention)

### 3. Background Flush Goroutine

```go
func (ac *AtomicMetricsCollector) startFlushRoutine() {
    ac.flushTicker = time.NewTicker(100 * time.Millisecond)
    go func() {
        for {
            select {
            case <-ac.flushTicker.C:
                ac.flush()
            case <-ac.stopChan:
                return
            }
        }
    }()
}

func (ac *AtomicMetricsCollector) flush() {
    // Read current write position
    currentPos := ac.responseBuffer.writePos.Load()
    
    // Copy samples for analysis (no lock needed for read)
    samples := ac.responseBuffer.ReadSamples(currentPos)
    
    // Aggregate under lock (cold path)
    ac.mu.Lock()
    ac.calculatePercentiles(samples)
    ac.updateTimeSeries()
    ac.mu.Unlock()
}
```

**Key Design Decisions**:
- Flush every 100ms (balance between accuracy and overhead)
- Reading ring buffer doesn't block writers
- Aggregation happens in background, not on hot path
- Time series updated periodically, not per-request

### 4. RecordRequest Implementation

```go
func (ac *AtomicMetricsCollector) RecordRequest(result *RequestResult) error {
    // HOT PATH - All atomic operations, no locks
    ac.totalRequests.Add(1)
    ac.totalBytes.Add(result.BytesReceived)
    
    // Record response time in ring buffer (lock-free)
    ac.responseBuffer.Write(result.Duration)
    
    // Update success/failure counters
    if result.Error != nil || result.StatusCode >= 400 {
        ac.failedRequests.Add(1)
        
        // COLD PATH - Only for errors (infrequent)
        if result.StatusCode > 0 || result.Error != nil {
            ac.mu.Lock()
            if result.StatusCode > 0 {
                ac.errorsByStatus[result.StatusCode]++
            }
            if result.Error != nil {
                ac.errorsByType["error"]++
            }
            ac.mu.Unlock()
        }
    } else {
        ac.successfulRequests.Add(1)
    }
    
    return nil
}
```

**Performance Characteristics**:
- Success case: ~50ns (3 atomic adds + 1 ring buffer write)
- Error case: ~500ns (includes mutex lock for error details)
- 99% of requests are success → hot path dominates

### 5. GetSnapshot Implementation

```go
func (ac *AtomicMetricsCollector) GetSnapshot() *MetricsSnapshot {
    // Read atomic counters (no lock)
    total := ac.totalRequests.Load()
    success := ac.successfulRequests.Load()
    failed := ac.failedRequests.Load()
    bytes := ac.totalBytes.Load()
    
    // Get aggregated stats (with lock)
    ac.mu.RLock()
    responseTimes := ac.cachedResponseTimeStats
    errors := ac.copyErrorMaps()
    timeSeries := ac.copyTimeSeries()
    ac.mu.RUnlock()
    
    return &MetricsSnapshot{
        TotalRequests:      total,
        SuccessfulRequests: success,
        FailedRequests:     failed,
        // ... rest of snapshot
    }
}
```

**Key Design Decisions**:
- Atomic reads for counters (instant, no lock)
- Cached aggregated stats from last flush
- Short read lock for cold-path data
- Snapshot is eventually consistent (acceptable)

## Data Flow

```
Worker Thread 1 ──┐
Worker Thread 2 ──┼──> RecordRequest() ──> Atomic Counters
Worker Thread N ──┘                    └──> Ring Buffer
                                            (lock-free)
                                                │
                                                │
                                                ▼
                                        Background Flush
                                        (every 100ms)
                                                │
                                                ├──> Calculate Percentiles
                                                ├──> Update Time Series
                                                └──> Cache Stats
                                                        │
                                                        ▼
                                                GetSnapshot()
                                                (read cached)
```

## Synchronization Strategy

### Hot Path (No Locks)
- Atomic operations for counters
- Lock-free ring buffer writes
- No blocking, no contention

### Cold Path (Mutex Protected)
- Error details (infrequent)
- Time series aggregation (background)
- Snapshot reads (cached data)

### Memory Ordering
- Atomic operations provide sequential consistency
- Ring buffer uses atomic write position
- Flush reads are eventually consistent (acceptable)

## Performance Analysis

### Expected Performance

| Operation | Current (Mutex) | New (Atomic) | Improvement |
|-----------|----------------|--------------|-------------|
| RecordRequest (success) | ~5,000ns | ~50ns | 100x faster |
| RecordRequest (error) | ~5,000ns | ~500ns | 10x faster |
| GetSnapshot | ~1,000ns | ~100ns | 10x faster |
| Memory per collector | ~100KB | ~150KB | 50% more |

### Throughput Capacity

- **Current**: ~130 RPS (mutex bottleneck)
- **Expected**: 10,000+ RPS (atomic operations)
- **Target**: 200 RPS (easily achievable)

## Migration Strategy

### Phase 1: Create New Collector
- Implement `AtomicMetricsCollector`
- Maintain existing interface
- Add comprehensive tests

### Phase 2: Feature Flag
```go
func NewMetricsCollector(useAtomic bool) MetricsCollector {
    if useAtomic {
        return NewAtomicMetricsCollector(10000)
    }
    return NewDefaultMetricsCollector()
}
```

### Phase 3: Gradual Rollout
- Enable in performance tests first
- Monitor for issues
- Enable by default

### Phase 4: Deprecation
- Mark old collector as deprecated
- Remove after 2 releases

## Testing Strategy

1. **Unit Tests**: Verify atomic operations
2. **Concurrency Tests**: 1000 goroutines hammering collector
3. **Race Detector**: `go test -race`
4. **Benchmarks**: Compare old vs new performance
5. **Integration Tests**: Full performance test scenarios

## Alternatives Considered

### Alternative 1: Channel-Based Collection
- **Pros**: Simple, no locks
- **Cons**: Channel becomes bottleneck, same issue
- **Decision**: Rejected

### Alternative 2: Per-Worker Collectors
- **Pros**: No contention
- **Cons**: Complex aggregation, memory overhead
- **Decision**: Rejected (over-engineered)

### Alternative 3: Sampling
- **Pros**: Reduce contention by recording 10% of requests
- **Cons**: Inaccurate counts, still has mutex
- **Decision**: Rejected (doesn't solve root cause)

## Open Questions

1. **Ring buffer size**: 10,000 samples sufficient? (Yes, for 200 RPS)
2. **Flush interval**: 100ms optimal? (Test 50ms, 100ms, 200ms)
3. **Error recording**: Always lock for errors? (Yes, errors are rare)
4. **Backward compatibility**: Support old interface? (Yes, required)

## Success Metrics

- [ ] Achieve 200+ RPS with 200 RPS target
- [ ] <5% CPU overhead for metrics collection
- [ ] Pass all existing tests
- [ ] No data races detected
- [ ] Accurate metrics (within 1%)
