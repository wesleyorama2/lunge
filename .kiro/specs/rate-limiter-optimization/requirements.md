# Requirements Document

## Introduction

The current performance testing engine in Lunge achieves only 82.5% efficiency when targeting 200 RPS (165 actual vs 200 target), despite having 100 workers with sub-millisecond latency and only 10% CPU usage. This indicates a fundamental inefficiency in the request generation and rate limiting mechanism. This feature will redesign the rate limiting system to achieve 95%+ efficiency across all target RPS levels, ensuring the load testing tool itself is not the bottleneck.

## Glossary

- **Rate_Limiter**: Component responsible for controlling the rate at which requests are generated and sent to workers
- **Request_Generator**: Component that creates and dispatches request work items to the worker pool
- **Token_Bucket**: Algorithm that allows bursts while maintaining average rate over time
- **Generation_Rate**: The rate at which the system attempts to send requests to workers (target RPS)
- **Completion_Rate**: The rate at which workers actually complete requests (actual RPS)
- **Rate_Efficiency**: Ratio of completion rate to generation rate, expressed as percentage
- **Worker_Pool**: Collection of concurrent goroutines that execute HTTP requests
- **Request_Channel**: Buffered channel used to dispatch work items to workers
- **Ticker_Overhead**: CPU time and context switches consumed by time-based scheduling mechanisms
- **Batch_Generation**: Technique of generating multiple requests per scheduling cycle to reduce overhead

## Requirements

### Requirement 1

**User Story:** As a load testing tool, I want to achieve 95%+ rate efficiency at all target RPS levels, so that test results accurately reflect target system performance rather than client-side bottlenecks.

#### Acceptance Criteria

1. WHEN the target RPS is 200, THE Rate_Limiter SHALL achieve at least 190 actual RPS (95% efficiency)
2. WHEN the target RPS is 1000, THE Rate_Limiter SHALL achieve at least 950 actual RPS (95% efficiency)
3. WHEN the target RPS is 10000, THE Rate_Limiter SHALL achieve at least 9000 actual RPS (90% efficiency)
4. THE Rate_Limiter SHALL maintain efficiency across different concurrency levels (10, 50, 100, 500 workers)
5. THE Rate_Limiter SHALL maintain efficiency during ramp-up and ramp-down phases

### Requirement 2

**User Story:** As a load testing tool, I want minimal CPU overhead from rate limiting logic, so that system resources are available for actual request execution.

#### Acceptance Criteria

1. WHEN generating requests at 200 RPS, THE Rate_Limiter SHALL consume less than 5% of one CPU core
2. WHEN generating requests at 1000 RPS, THE Rate_Limiter SHALL consume less than 10% of one CPU core
3. THE Rate_Limiter SHALL avoid nested select statements that increase context switch overhead
4. THE Rate_Limiter SHALL minimize time calculations per request (target: less than 1 calculation per 10 requests)
5. THE Rate_Limiter SHALL use batch generation to reduce per-request overhead

### Requirement 3

**User Story:** As a load testing tool, I want accurate rate limiting without ticker overhead, so that request generation is both precise and efficient.

#### Acceptance Criteria

1. THE Rate_Limiter SHALL implement token bucket algorithm for rate control
2. THE Rate_Limiter SHALL avoid time.Ticker-based scheduling for steady-state generation
3. WHEN the request channel has capacity, THE Rate_Limiter SHALL send requests without blocking
4. WHEN the request channel is full, THE Rate_Limiter SHALL apply backpressure without dropping requests
5. THE Rate_Limiter SHALL calculate timing based on elapsed time rather than tick counts

### Requirement 4

**User Story:** As a load testing tool, I want smooth ramp-up and ramp-down without efficiency loss, so that load transitions are realistic and accurate.

#### Acceptance Criteria

1. WHEN ramping up from 0 to target RPS, THE Rate_Limiter SHALL maintain 95%+ efficiency throughout the ramp
2. WHEN ramping down from target to 0 RPS, THE Rate_Limiter SHALL maintain 95%+ efficiency throughout the ramp
3. THE Rate_Limiter SHALL update target rate smoothly without sudden jumps or drops
4. THE Rate_Limiter SHALL calculate ramp progress based on elapsed time, not tick counts
5. THE Rate_Limiter SHALL use the same efficient generation mechanism during ramps as steady-state

### Requirement 5

**User Story:** As a load testing tool, I want lock-free metrics collection for high-throughput scenarios, so that metrics recording does not become a bottleneck.

#### Acceptance Criteria

1. THE Metrics_Collector SHALL use atomic operations for request counters (total, success, failed)
2. THE Metrics_Collector SHALL buffer response time data before acquiring locks
3. WHEN recording metrics at 10000 RPS, THE Metrics_Collector SHALL consume less than 5% of one CPU core
4. THE Metrics_Collector SHALL separate hot path operations (counters) from cold path operations (analysis)
5. THE Metrics_Collector SHALL avoid global mutex locks on every request completion

### Requirement 6

**User Story:** As a load testing tool, I want accurate generation rate tracking, so that I can visualize the gap between target and actual throughput.

#### Acceptance Criteria

1. WHEN generating requests, THE Rate_Limiter SHALL record the actual generation rate per second
2. THE Rate_Limiter SHALL distinguish between "requests sent to channel" and "requests completed by workers"
3. WHEN the request channel is full, THE Rate_Limiter SHALL not count skipped sends as generated requests
4. THE Metrics_Collector SHALL provide both generation rate and completion rate in time series data
5. THE Performance_Reporter SHALL visualize the gap between generation and completion rates in charts

### Requirement 7

**User Story:** As a developer, I want comprehensive benchmarks for rate limiting components, so that I can validate performance improvements and prevent regressions.

#### Acceptance Criteria

1. THE Rate_Limiter SHALL have benchmark tests measuring pure generation rate without HTTP overhead
2. THE Metrics_Collector SHALL have benchmark tests measuring recording overhead per request
3. THE Request_Channel SHALL have benchmark tests measuring send/receive throughput
4. THE benchmarks SHALL run as part of the test suite to detect performance regressions
5. THE benchmarks SHALL provide baseline measurements for comparison after optimization

### Requirement 8

**User Story:** As a load testing tool, I want efficient request channel management, so that work distribution to workers is not a bottleneck.

#### Acceptance Criteria

1. THE Request_Channel SHALL have buffer size proportional to worker count (minimum 10x workers)
2. WHEN the channel is full, THE Rate_Limiter SHALL apply backpressure without busy-waiting
3. THE Worker_Pool SHALL consume from the channel efficiently without blocking other workers
4. THE Request_Generator SHALL pre-allocate request metadata to minimize allocations
5. THE Request_Channel SHALL support at least 100,000 operations per second

### Requirement 9

**User Story:** As a developer, I want clear separation between rate limiting and request execution, so that each component can be optimized independently.

#### Acceptance Criteria

1. THE Rate_Limiter SHALL be a standalone component with well-defined interface
2. THE Rate_Limiter SHALL not depend on HTTP client implementation details
3. THE Request_Generator SHALL use the Rate_Limiter interface for all rate control
4. THE Rate_Limiter SHALL be testable in isolation without HTTP requests
5. THE Rate_Limiter SHALL support pluggable rate limiting algorithms

### Requirement 10

**User Story:** As a load testing tool, I want validation that rate limiting efficiency meets requirements, so that users can trust the tool's accuracy.

#### Acceptance Criteria

1. WHEN performance tests complete, THE Performance_Reporter SHALL display rate efficiency percentage
2. WHEN efficiency falls below 95%, THE Performance_Reporter SHALL warn users about potential client-side bottlenecks
3. THE Performance_Engine SHALL validate rate limiter performance during startup with a self-test
4. THE Performance_Engine SHALL provide diagnostic information when efficiency is below target
5. THE Performance_Reporter SHALL include efficiency metrics in all report formats (text, JSON, HTML)
