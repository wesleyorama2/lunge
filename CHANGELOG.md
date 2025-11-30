# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

*No changes yet.*

## [2.0.0] - 2025-11-30

### Added

#### Performance Testing Engine v2
- **New executor types** inspired by K6/Gatling for professional-grade load testing:
  - `constant-vus` - Maintains a fixed number of virtual users throughout the test
  - `ramping-vus` - VU count changes through defined stages (ramp-up, steady-state, ramp-down)
  - `constant-arrival-rate` - Maintains a constant rate of new iterations per second
  - `ramping-arrival-rate` - Iteration rate changes through defined stages
- **HTML reports** with Chart.js visualization including:
  - Response time distribution charts
  - RPS over time charts
  - Error breakdown
  - Threshold results
  - Per-scenario metrics
- **YAML configuration support** for scenarios and stages
- **Threshold support** for pass/fail criteria (p50, p90, p95, p99, avg, rate, count)
- **TTY-aware console output** with real-time progress bars and live metrics
- **Think time and pacing support** for realistic user behavior simulation
- **Multi-scenario support** with parallel execution and configurable start times
- **Variable extraction** from responses for request chaining

#### Lock-Free Metrics Collector
- **AtomicMetricsCollector** - Lock-free metrics collection providing 100x performance improvement:
  - Lock-free ring buffer for response time recording
  - Atomic counters for hot-path operations
  - Background flush routine for aggregated statistics
  - Supports 10,000+ RPS with minimal CPU overhead (<5%)

### Changed
- Performance engine completely rewritten for accuracy and scalability
- Rate limiter changed from token bucket to leaky bucket algorithm for smoother request distribution
- Default metrics collector changed to AtomicMetricsCollector for better performance

### Fixed
- Rate limiter accuracy - was generating 2x expected requests, now precisely matches target RPS
- Race conditions in executor Stop() methods
- VU tracking accuracy in metrics engine
- Connection pool exhaustion under high load

### Deprecated
- Legacy mutex-based `Collector` - use `AtomicMetricsCollector` instead

## [1.0.0] - 2025-01-01

### Added
- HTTP client with GET, POST, PUT, DELETE support
- Multiple output formats (Text, JSON, YAML, JUnit XML)
- Request timing metrics (DNS, TCP, TLS, TTFB)
- JSON configuration for request suites
- Environment variables and variable substitution
- Response validation (status, headers, body)
- JSON Schema validation
- Test suites with assertions
- Request chaining with variable extraction
- Cookie handling
- Basic authentication support
- Custom headers
- Request body from file or inline
- Verbose mode with detailed timing
- Configuration file examples
