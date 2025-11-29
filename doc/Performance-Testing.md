# Performance Testing with Lunge

Lunge provides comprehensive performance and load testing capabilities through the `perf` command. This guide covers the performance engine and its various executor types, configuration options, and reporting features.

## Table of Contents

- [Quick Start](#quick-start)
- [Executors](#executors)
- [Configuration](#configuration)
- [CLI Usage](#cli-usage)
- [Thresholds](#thresholds)
- [Output and Reports](#output-and-reports)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

**Quick URL Test:**
```bash
lunge perf --url https://api.example.com/health \
  --vus 10 --duration 30s
```

**Config File Test:**
```bash
lunge perf -c test.yaml
```

**Ramping Test:**
```bash
lunge perf --url https://api.example.com/health \
  --executor ramping-vus \
  --stages "30s:10,2m:50,30s:0"
```

**Arrival Rate Test:**
```bash
lunge perf --url https://api.example.com/health \
  --executor constant-arrival-rate \
  --rate 100 --duration 5m --max-vus 200
```

---

## Executors

The performance engine supports four executor types, each designed for different load testing scenarios:

### 1. `constant-vus` - Fixed Virtual Users

Maintains a constant number of virtual users throughout the test. Each VU loops through the defined requests continuously.

**Best for:**
- Baseline performance testing
- Simulating consistent concurrent user load
- Quick validation tests

**Configuration:**
```yaml
scenarios:
  test:
    executor: constant-vus
    vus: 50            # Number of virtual users
    duration: 5m       # Test duration
    gracefulStop: 10s  # Wait time for iterations to finish
```

**CLI:**
```bash
lunge perf --url https://api.example.com/users \
  --executor constant-vus --vus 50 --duration 5m
```

### 2. `ramping-vus` - Variable Virtual Users

VU count changes through defined stages, allowing ramp-up, steady-state, and ramp-down patterns.

**Best for:**
- Stress testing with gradual load increase
- Finding breaking points
- Realistic user behavior simulation

**Configuration:**
```yaml
scenarios:
  test:
    executor: ramping-vus
    stages:
      - duration: 1m
        target: 10
        name: "ramp-up"
      - duration: 3m
        target: 50
        name: "steady"
      - duration: 30s
        target: 100
        name: "spike"
      - duration: 1m
        target: 0
        name: "ramp-down"
    gracefulStop: 30s
```

**CLI:**
```bash
lunge perf --url https://api.example.com/users \
  --executor ramping-vus \
  --stages "1m:10,3m:50,30s:100,1m:0"
```

### 3. `constant-arrival-rate` - Fixed Request Rate

Maintains a constant rate of new iterations per second, scaling VUs as needed to achieve the target rate.

**Best for:**
- API throughput testing
- SLA validation (e.g., "must handle 1000 RPS")
- Load testing with guaranteed request rate

**Configuration:**
```yaml
scenarios:
  test:
    executor: constant-arrival-rate
    rate: 100          # Iterations per second
    duration: 5m       # Test duration
    preAllocatedVUs: 50   # Initial VU pool
    maxVUs: 200        # Maximum VUs if needed
    gracefulStop: 30s
```

**CLI:**
```bash
lunge perf --url https://api.example.com/health \
  --executor constant-arrival-rate \
  --rate 100 --duration 5m \
  --pre-allocated-vus 50 --max-vus 200
```

### 4. `ramping-arrival-rate` - Variable Request Rate

Iteration rate changes through defined stages, useful for gradually increasing load while measuring throughput.

**Best for:**
- Finding throughput limits
- Gradual API capacity testing
- Variable load patterns

**Configuration:**
```yaml
scenarios:
  test:
    executor: ramping-arrival-rate
    preAllocatedVUs: 20
    maxVUs: 100
    stages:
      - duration: 2m
        target: 50
        name: "ramp-to-50rps"
      - duration: 5m
        target: 100
        name: "ramp-to-100rps"
      - duration: 2m
        target: 0
        name: "ramp-down"
    gracefulStop: 30s
```

**CLI:**
```bash
lunge perf --url https://api.example.com/health \
  --executor ramping-arrival-rate \
  --stages "2m:50,5m:100,2m:0" \
  --pre-allocated-vus 20 --max-vus 100
```

### Executor Comparison

| Executor | Load Control | VU Scaling | Use Case |
|----------|-------------|------------|----------|
| `constant-vus` | Fixed VUs | None | Baseline, quick tests |
| `ramping-vus` | Variable VUs | Manual via stages | Stress, spike tests |
| `constant-arrival-rate` | Fixed RPS | Auto-scales | API throughput |
| `ramping-arrival-rate` | Variable RPS | Auto-scales | Capacity finding |

## Configuration

The performance engine uses YAML (or JSON) configuration files for complex test scenarios.

### Complete Configuration Reference

```yaml
# Test identification
name: "API Performance Test"
description: "Load test for user management API"

# Global settings for all scenarios
settings:
  baseUrl: "https://api.example.com"   # Base URL for requests
  timeout: 30s                          # Default request timeout
  maxConnectionsPerHost: 100            # Connection pool size
  maxIdleConnsPerHost: 100              # Idle connection pool size
  insecureSkipVerify: false             # Skip TLS verification
  userAgent: "lunge/2.0"                # Default User-Agent
  headers:                              # Default headers for all requests
    Accept: "application/json"
    X-API-Version: "v2"

# Global variables (available to all scenarios)
variables:
  environment: "production"
  api_key: "${API_KEY:-default-key}"

# Scenario definitions
scenarios:
  # Scenario 1: Browse users
  browse_users:
    executor: constant-vus
    vus: 20
    duration: 5m
    gracefulStop: 10s
    startTime: 0s                       # When to start (relative to test start)
    tags:
      scenario_type: "browse"
    
    requests:
      - name: "List Users"
        method: GET
        url: "{{baseUrl}}/api/users"
        headers:
          Authorization: "Bearer {{api_key}}"
        timeout: 10s
        thinkTime: 500ms                # Wait after this request
        
        # Response assertions
        assertions:
          - type: status
            condition: eq
            value: "200"
          - type: duration
            condition: lt
            value: "1s"
        
        # Variable extraction
        extract:
          - name: "userId"
            source: body
            path: "$.data[0].id"

      - name: "Get User Details"
        method: GET
        url: "{{baseUrl}}/api/users/{{userId}}"
        headers:
          Authorization: "Bearer {{api_key}}"

  # Scenario 2: Create users
  create_users:
    executor: constant-arrival-rate
    rate: 10
    duration: 5m
    preAllocatedVUs: 5
    maxVUs: 20
    startTime: 30s                      # Start 30s after test begins
    
    requests:
      - name: "Create User"
        method: POST
        url: "{{baseUrl}}/api/users"
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{api_key}}"
        body: |
          {
            "name": "Test User {{iteration}}",
            "email": "user{{iteration}}@test.com"
          }

  # Scenario 3: Spike test
  spike_test:
    executor: ramping-vus
    startTime: 3m                       # Start after browse test stabilizes
    stages:
      - duration: 30s
        target: 100
        name: "spike"
      - duration: 1m
        target: 100
        name: "hold"
      - duration: 30s
        target: 0
        name: "recover"
    
    requests:
      - name: "Health Check"
        method: GET
        url: "{{baseUrl}}/health"

# Threshold definitions
thresholds:
  http_req_duration:
    - "p95 < 500ms"        # 95th percentile under 500ms
    - "p99 < 1s"           # 99th percentile under 1s
    - "avg < 200ms"        # Average under 200ms
  
  http_req_failed:
    - "rate < 0.01"        # Less than 1% failures
  
  http_reqs:
    - "rate > 100"         # At least 100 req/s throughput
    - "count > 10000"      # At least 10000 total requests
  
  # Custom thresholds for specific scenarios
  custom:
    browse_users_duration:
      - "p95 < 300ms"
    create_users_failed:
      - "rate < 0.001"

# Execution options
options:
  sequential: false        # Run scenarios in parallel (default)
  iterationsTimeout: 60s   # Max time for iteration completion
  setupTimeout: 30s        # Max setup time
  teardownTimeout: 30s     # Max teardown time
  noVUConnectionReuse: false  # Reuse connections between VUs
```

### Request Configuration

Each request in a scenario can be configured with:

```yaml
requests:
  - name: "Create User"          # Name for metrics
    method: POST                 # HTTP method
    url: "{{baseUrl}}/api/users" # URL with variable substitution
    
    # Headers
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer {{token}}"
    
    # Request body
    body: |
      {
        "name": "{{username}}",
        "email": "{{email}}"
      }
    
    # Timeout for this specific request
    timeout: 5s
    
    # Think time - wait after request completes
    thinkTime: 1s
    
    # Variable extraction from response
    extract:
      - name: "userId"
        source: body           # body, header, or status
        path: "$.id"           # JSONPath for body
      - name: "requestId"
        source: header
        path: "X-Request-ID"
    
    # Response assertions
    assertions:
      - type: status           # status, body, header, duration
        condition: eq          # eq, ne, gt, lt, gte, lte, contains, matches
        value: "201"
      - type: body
        path: "$.success"
        condition: eq
        value: "true"
      - type: duration
        condition: lt
        value: "500ms"
```

### Pacing Configuration

Control timing between iterations:

```yaml
scenarios:
  test:
    executor: constant-vus
    vus: 10
    duration: 5m
    
    pacing:
      type: constant           # none, constant, or random
      duration: 1s             # For constant pacing
    
    # OR for random pacing:
    pacing:
      type: random
      min: 500ms
      max: 2s
```

## CLI Usage

### Basic Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Configuration file | - |
| `--url` | - | URL to test (quick mode) | - |
| `--duration` | - | Test duration | 30s |
| `--verbose` | `-v` | Verbose output | false |
| `--timeout` | `-t` | Request timeout | 30s |

### Performance Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--executor` | Executor type | constant-vus |
| `--vus` | Number of VUs | 10 |
| `--stages` | Ramping stages (format: `duration:target,...`) | - |
| `--rate` | Iterations per second | - |
| `--max-vus` | Maximum VUs (arrival-rate) | - |
| `--pre-allocated-vus` | Pre-allocated VUs (arrival-rate) | - |
| `--html` | Generate HTML report | false |
| `--json` | Output results as JSON | false |
| `--quiet`, `-q` | Disable live progress | false |
| `--output` | Output file path | - |

### CLI Examples

```bash
# Basic load test with VUs
lunge perf --url https://api.example.com/health \
  --vus 50 --duration 5m

# Ramping VUs test
lunge perf --url https://api.example.com/health \
  --executor ramping-vus \
  --stages "1m:20,3m:50,1m:0"

# Constant arrival rate
lunge perf --url https://api.example.com/health \
  --executor constant-arrival-rate \
  --rate 100 --duration 5m \
  --max-vus 200

# With config file and HTML report
lunge perf -c test.yaml --html --output results.html

# JSON output for CI/CD
lunge perf -c test.yaml --json --output results.json

# Quiet mode (final summary only)
lunge perf -c test.yaml -q
```

## Thresholds

Thresholds define pass/fail criteria for your tests. They're specified in the config file:

```yaml
thresholds:
  # Request duration thresholds
  http_req_duration:
    - "p50 < 200ms"    # 50th percentile
    - "p90 < 400ms"    # 90th percentile
    - "p95 < 500ms"    # 95th percentile
    - "p99 < 1s"       # 99th percentile
    - "avg < 250ms"    # Average
    - "min < 50ms"     # Minimum
    - "max < 5s"       # Maximum
  
  # Error rate thresholds
  http_req_failed:
    - "rate < 0.01"    # Less than 1% error rate
  
  # Request count/rate thresholds
  http_reqs:
    - "count > 10000"  # Total requests
    - "rate > 100"     # Requests per second
```

### Threshold Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `<` | Less than | `p95 < 500ms` |
| `<=` | Less than or equal | `avg <= 200ms` |
| `>` | Greater than | `rate > 100` |
| `>=` | Greater than or equal | `count >= 1000` |

## Output and Reports

### Console Output

By default, the engine displays real-time progress:

```
════════════════════════════════════════════════════════════════
 API Performance Test                    constant-vus
════════════════════════════════════════════════════════════════

 Progress   [=====================>          ] 67% (2m00s/3m00s)
 VUs        50/50 (target)
 Stage      2/3 (steady)
 
 ─── Live Metrics ──────────────────────────────────────────────
 Requests   15,234 total (84.6/s)
 Errors     12 (0.08%)
 
 ─── Latency ───────────────────────────────────────────────────
 p50        145ms
 p95        312ms
 p99        567ms
 max        1.23s
```

### HTML Reports

Generate comprehensive HTML reports with `--html`:

```bash
lunge perf -c test.yaml --html --output report.html
```

HTML reports include:
- Summary statistics
- Request latency distribution charts
- RPS over time charts
- Error breakdown
- Threshold results
- Per-scenario metrics

### JSON Output

For CI/CD integration:

```bash
lunge perf -c test.yaml --json --output results.json
```

JSON output includes:
```json
{
  "name": "API Performance Test",
  "passed": true,
  "duration": "3m0s",
  "startTime": "2024-01-15T10:00:00Z",
  "endTime": "2024-01-15T10:03:00Z",
  "metrics": {
    "totalRequests": 15234,
    "successRequests": 15222,
    "failedRequests": 12,
    "errorRate": 0.0008,
    "rps": 84.6,
    "totalBytes": 45678900,
    "latency": {
      "min": "12ms",
      "max": "1.23s",
      "mean": "165ms",
      "p50": "145ms",
      "p90": "287ms",
      "p95": "312ms",
      "p99": "567ms"
    }
  },
  "scenarios": {
    "browse_users": {
      "executor": "constant-vus",
      "duration": "3m0s",
      "iterations": 8234
    }
  },
  "thresholds": [
    {
      "metric": "http_req_duration",
      "expression": "p95 < 500ms",
      "passed": true,
      "value": "312ms"
    }
  ]
}
```

---

## Best Practices

### 1. Start Small and Scale

```yaml
# Begin with low load
scenarios:
  initial:
    executor: constant-vus
    vus: 5
    duration: 1m
```

Then gradually increase based on results.

### 2. Use Ramping for Realistic Tests

```yaml
scenarios:
  realistic:
    executor: ramping-vus
    stages:
      - duration: 2m
        target: 10
        name: "warm-up"
      - duration: 5m
        target: 50
        name: "normal-load"
      - duration: 2m
        target: 0
        name: "cool-down"
```

### 3. Set Appropriate Thresholds

Based on SLOs/SLAs:

```yaml
thresholds:
  http_req_duration:
    - "p95 < 500ms"   # SLA: 95% under 500ms
    - "p99 < 1s"      # SLA: 99% under 1s
  http_req_failed:
    - "rate < 0.001"  # SLA: 99.9% availability
```

### 4. Use Think Time for Realistic Patterns

```yaml
requests:
  - name: "Browse Page"
    url: "{{baseUrl}}/products"
    thinkTime: 3s     # User reads the page

  - name: "Click Product"
    url: "{{baseUrl}}/products/{{productId}}"
    thinkTime: 5s     # User examines product
```

### 5. Monitor System Resources

Before running high-load tests, ensure:
- Sufficient CPU and memory on test machine
- Network bandwidth is not limiting
- Target system can handle the load

### 6. Use Multiple Scenarios for Complex Flows

```yaml
scenarios:
  browsers:
    executor: constant-vus
    vus: 100
    duration: 10m
    requests:
      - name: "Browse"
        url: "{{baseUrl}}/products"

  buyers:
    executor: constant-vus
    vus: 10
    duration: 10m
    requests:
      - name: "Add to Cart"
        url: "{{baseUrl}}/cart"
      - name: "Checkout"
        url: "{{baseUrl}}/checkout"
```

### 7. Use Arrival Rate for SLA Testing

When you need guaranteed throughput:

```yaml
scenarios:
  sla_test:
    executor: constant-arrival-rate
    rate: 1000        # Must handle 1000 RPS
    duration: 10m
    maxVUs: 500       # Scale up VUs as needed
```

---

## Troubleshooting

### High Error Rates

**Symptoms:** Error rate above threshold

**Solutions:**
1. Reduce concurrency or rate
2. Increase ramp-up duration
3. Check target server capacity
4. Verify timeouts are appropriate

```yaml
settings:
  timeout: 60s        # Increase timeout

scenarios:
  test:
    executor: ramping-vus
    stages:
      - duration: 5m   # Longer ramp-up
        target: 50
```

### Inconsistent Results

**Symptoms:** Large variance between test runs

**Solutions:**
1. Use warmup stages
2. Run longer tests
3. Ensure isolated test environment
4. Check for background processes

### Low Throughput

**Symptoms:** RPS lower than expected

**Solutions:**
1. Increase VUs for VU-based executors
2. Check connection pool settings
3. Reduce think time
4. Verify network capacity

```yaml
settings:
  maxConnectionsPerHost: 200
  maxIdleConnsPerHost: 200
```

### Memory Issues

**Symptoms:** Test machine runs out of memory

**Solutions:**
1. Reduce VU count
2. Limit response body collection
3. Use quiet mode (`-q`)
4. Run on machine with more RAM

### Connection Issues

**Symptoms:** Connection refused or timeout errors

**Solutions:**
1. Check target server connectivity
2. Verify firewall rules
3. Increase connection pool
4. Use keep-alive connections

---

## Examples

See the [examples directory](../examples/) for complete configurations:

- [`basic-load-test.yaml`](../examples/v2-basic-load-test.yaml) - Simple constant VUs test
- [`stress-test.yaml`](../examples/v2-stress-test.yaml) - Ramping VUs stress test
- [`api-throughput.yaml`](../examples/v2-api-throughput.yaml) - Constant arrival rate
- [`spike-test.yaml`](../examples/v2-spike-test.yaml) - Spike test pattern
- [`soak-test.yaml`](../examples/v2-soak-test.yaml) - Long duration soak test
- [`multi-scenario.yaml`](../examples/v2-multi-scenario.yaml) - Multiple scenarios

---

## See Also

- [Configuration Guide](Configuration.md) - General configuration options
- [Testing Guide](Testing.md) - Functional testing
- [Examples](Examples.md) - More examples
