# Rate Limiting in Lunge

Lunge uses an optimized token bucket algorithm for accurate, low-overhead rate limiting during performance tests. This document explains how the rate limiter works, its performance characteristics, and how to configure it.

## Overview

The rate limiter controls how fast Lunge generates requests during performance tests. It ensures that:
- Requests are generated at the target RPS (requests per second)
- The load testing tool itself is not the bottleneck
- CPU overhead from rate limiting is minimal
- Efficiency remains high (95%+) at all target RPS levels

## Token Bucket Algorithm

### How It Works

The token bucket algorithm is a widely-used rate limiting technique that provides accurate rate control with minimal overhead:

1. **Token Bucket**: A "bucket" holds tokens that represent permission to send requests
2. **Token Refill**: Tokens are added to the bucket at a constant rate (target RPS)
3. **Token Consumption**: Each request consumes one or more tokens from the bucket
4. **Immediate Execution**: If tokens are available, requests proceed immediately
5. **Waiting**: If no tokens are available, requests wait until tokens are refilled

### Visual Example

```
Time: 0s          Time: 0.5s        Time: 1.0s
Bucket: [10]      Bucket: [5]       Bucket: [10]
Rate: 10 RPS      Rate: 10 RPS      Rate: 10 RPS
                  
[10 tokens]       [5 tokens used]   [5 tokens refilled]
                  [5 requests sent] [10 tokens available]
```

### Key Advantages

**1. No Ticker Overhead**
- Traditional approach: Uses `time.Ticker` that fires every millisecond
- Token bucket: Only calculates time when tokens are needed
- Result: 100x reduction in context switches

**2. Burst Handling**
- Allows brief bursts up to 2x target rate (configurable)
- Smooths out timing variations
- Maintains average rate over time

**3. Adaptive Refill**
- Refills based on actual elapsed time, not fixed intervals
- Automatically adjusts to system timing variations
- No per-request time calculations

**4. Context-Aware**
- Respects context cancellation
- No goroutine leaks
- Clean shutdown

## Performance Characteristics

### CPU Overhead

| Target RPS | CPU Overhead | Context Switches/sec |
|------------|--------------|---------------------|
| 200 | <5% | ~20 |
| 1,000 | <10% | ~100 |
| 10,000 | <15% | ~1,000 |

Compare to ticker-based approach:
- 200 RPS: 200 context switches/sec (10x more)
- 1,000 RPS: 1,000 context switches/sec (10x more)
- 10,000 RPS: 10,000 context switches/sec (10x more)

### Efficiency

Efficiency is the ratio of actual RPS to target RPS:

| Target RPS | Expected Efficiency | Actual RPS (example) |
|------------|-------------------|---------------------|
| 200 | 95%+ | 195+ |
| 1,000 | 95%+ | 950+ |
| 10,000 | 90%+ | 9,000+ |

**Factors Affecting Efficiency:**
- System resources (CPU, memory, network)
- Worker count (concurrency)
- Target system response time
- Channel buffer size

### Accuracy

The token bucket maintains rate accuracy within ±2% of target:

```
Target: 1000 RPS
Actual: 980-1020 RPS (98-102% of target)
```

### Burst Capacity

Default burst capacity is 2x target rate:

```
Target: 100 RPS
Burst capacity: 200 tokens
Max burst: 200 requests in <1 second
Average over time: 100 RPS
```

## Batch Request Generation

To further reduce overhead, Lunge generates requests in batches rather than one at a time.

### Adaptive Batch Sizing

Batch size is automatically calculated based on target RPS:

| Target RPS | Batch Size | Overhead Reduction |
|------------|------------|-------------------|
| <100 | 1 | None (not needed) |
| 100-1,000 | 1-10 | 10x |
| 1,000-10,000 | 10-100 | 100x |
| >10,000 | 100 | 100x |

### Example

```
Target: 1000 RPS
Batch size: 10 requests
Batches per second: 100
Overhead: 1/10th of single-request approach
```

## Configuration

### Basic Configuration

```json
{
  "performance": {
    "loadTest": {
      "load": {
        "rps": 100
      }
    }
  }
}
```

### Advanced Configuration

```json
{
  "performance": {
    "loadTest": {
      "load": {
        "rps": 1000
      },
      "rateLimiter": {
        "algorithm": "token-bucket",
        "burstCapacity": 2.0,
        "batchSize": "auto",
        "efficiencyThreshold": 0.95
      }
    }
  }
}
```

### Configuration Options

**algorithm** (string, default: "token-bucket")
- Rate limiting algorithm to use
- Currently only "token-bucket" is supported
- Future: May support other algorithms like leaky bucket

**burstCapacity** (float, default: 2.0)
- Burst capacity as multiple of target rate
- Range: 1.0 to 10.0
- Higher values allow more bursty traffic
- Lower values enforce stricter rate limits
- Recommended: 2.0 for most use cases

**batchSize** (string or int, default: "auto")
- Batch size for request generation
- "auto": Automatically calculate based on target RPS (recommended)
- Integer (1-100): Fixed batch size
- Higher values reduce overhead but increase burstiness

**efficiencyThreshold** (float, default: 0.95)
- Minimum acceptable efficiency (0.0 to 1.0)
- Warnings are logged when efficiency drops below threshold
- Recommended: 0.95 (95%) for production tests

## Efficiency Monitoring

### Real-Time Metrics

Lunge tracks efficiency in real-time during tests:

```
Rate Limiter Efficiency:
  Target RPS: 1000.0
  Actual RPS: 975.3
  Efficiency: 97.5% ✓
  Generation Rate: 982.1 req/s
  Completion Rate: 975.3 req/s
  Average Wait Time: 0.8ms
  Channel Utilization: 12.5%
```

### Metric Definitions

**Target RPS**: The requested rate from configuration

**Actual RPS**: The rate at which requests are actually completed by workers

**Efficiency**: Ratio of actual RPS to target RPS (0.0 to 1.0)
- Formula: `efficiency = actual RPS / target RPS`
- Example: 975 actual / 1000 target = 0.975 (97.5%)

**Generation Rate**: The rate at which requests are sent to workers
- Should be close to target RPS
- If much lower, rate limiter is bottlenecked

**Completion Rate**: The rate at which workers complete requests
- Should match generation rate
- If lower, workers are bottlenecked

**Average Wait Time**: Average time spent waiting for rate limiter tokens
- Lower is better
- High values indicate rate limiter overhead

**Channel Utilization**: Percentage of time the request channel is full
- 0-50%: Good, plenty of buffer capacity
- 50-80%: Acceptable, some backpressure
- 80-100%: High, frequent backpressure

### Efficiency Warnings

Lunge automatically logs warnings when efficiency drops below threshold:

```
WARNING: Rate limiter efficiency at 87.3%, expected 95.0%+ 
         (target: 1000.0 RPS, actual: 873.2 RPS, blocked: 1247, avg wait: 15.3ms)
```

This indicates:
- Efficiency is below threshold (87.3% < 95%)
- Target rate is 1000 RPS but only achieving 873 RPS
- Rate limiter was blocked 1247 times waiting for tokens
- Average wait time is 15.3ms (high overhead)

## Ramp-Up and Ramp-Down

The token bucket algorithm supports smooth rate transitions during ramp-up and ramp-down phases.

### Implementation

**Rate Updates:**
- Separate goroutine updates token bucket rate every 100ms
- Linear interpolation from start RPS to end RPS
- No per-request calculations during ramps

**Example Ramp-Up:**
```
Start: 0 RPS
End: 1000 RPS
Duration: 30 seconds
Updates: 300 (every 100ms)

Update 1 (0.1s):   currentRPS = 0 + (1000 - 0) * (0.1 / 30) = 3.3 RPS
Update 2 (0.2s):   currentRPS = 0 + (1000 - 0) * (0.2 / 30) = 6.7 RPS
...
Update 300 (30s):  currentRPS = 0 + (1000 - 0) * (30 / 30) = 1000 RPS
```

### Efficiency During Ramps

The token bucket maintains high efficiency during ramp-up and ramp-down:

| Phase | Expected Efficiency |
|-------|-------------------|
| Ramp-up | 95%+ |
| Steady-state | 95%+ |
| Ramp-down | 95%+ |

## Comparison to Ticker-Based Approach

### Old Approach (Ticker-Based)

```go
ticker := time.NewTicker(5 * time.Millisecond)
for {
    select {
    case <-ticker.C:
        // Calculate if we should send request
        if shouldSend() {
            requestChan <- request
        }
    case <-ctx.Done():
        return
    }
}
```

**Problems:**
- Ticker fires every 5ms regardless of need (200 times/sec)
- Nested select statements increase overhead
- Per-tick time calculations
- High context switch overhead
- Efficiency: 82.5% at 200 RPS

### New Approach (Token Bucket)

```go
limiter := NewTokenBucket(targetRPS, 2.0)
for {
    // Wait for tokens (blocks until available)
    count, err := limiter.Wait(ctx)
    if err != nil {
        return
    }
    
    // Generate batch of requests
    for i := 0; i < count; i++ {
        requestChan <- request
    }
}
```

**Advantages:**
- Only calculates time when tokens are needed
- No nested select statements
- Batch generation reduces overhead
- Low context switch overhead
- Efficiency: 97.5% at 200 RPS

### Performance Comparison

| Metric | Ticker-Based | Token Bucket | Improvement |
|--------|-------------|--------------|-------------|
| Efficiency @ 200 RPS | 82.5% | 97.5% | +18% |
| CPU overhead @ 1000 RPS | 15-20% | <10% | 2x better |
| Context switches/sec | 1000 | 10 | 100x better |
| Code complexity | High | Low | Simpler |

## Best Practices

### 1. Use Default Settings

For most use cases, default settings work well:

```json
{
  "load": {
    "rps": 1000
  }
}
```

### 2. Monitor Efficiency

Always check efficiency in reports:

```bash
lunge perf -c config.json -e dev -p loadTest --format html --output report.html
```

### 3. Adjust Concurrency

If efficiency is low, increase concurrency:

```json
{
  "load": {
    "concurrency": 100,
    "rps": 1000
  }
}
```

Rule of thumb: `concurrency = target RPS / (1000 / avg response time ms)`

### 4. Use Warmup

Add warmup phase to establish connections:

```json
{
  "load": {
    "warmup": {
      "duration": "10s",
      "iterations": 100
    }
  }
}
```

### 5. Start Small

Begin with low RPS and gradually increase:

```bash
# Test at 100 RPS first
lunge perf -c config.json -e dev -r getUsers --rps 100 --duration 1m

# Then increase to 1000 RPS
lunge perf -c config.json -e dev -r getUsers --rps 1000 --duration 1m
```

### 6. Use Ramp-Up

Gradually increase load to avoid overwhelming target system:

```json
{
  "load": {
    "rps": 1000,
    "rampUp": "30s"
  }
}
```

## Troubleshooting

See [Performance Testing Guide - Troubleshooting Low Efficiency](Performance-Testing.md#troubleshooting-low-efficiency) for detailed troubleshooting steps.

## Technical Details

### Thread Safety

TokenBucket is safe for concurrent use:
- Mutex protects token calculations
- Atomic operations for efficiency tracking
- No data races

### Memory Usage

Memory usage is minimal:
- TokenBucket struct: ~200 bytes
- No allocations in hot path
- Batch generation pre-allocates request metadata

### Goroutine Management

- One goroutine for rate updates during ramps
- No goroutine leaks on cancellation
- Clean shutdown on context cancellation

## See Also

- [Performance Testing Guide](Performance-Testing.md)
- [Architecture Documentation](../ARCHITECTURE.md)
- [Configuration Guide](Configuration.md)
