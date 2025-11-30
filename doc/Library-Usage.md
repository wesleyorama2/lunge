# Using Lunge as a Library

Lunge can be used as a Go library in addition to its CLI functionality. This guide shows how to integrate Lunge into your Go applications.

## Installation

```bash
go get github.com/wesleyorama2/lunge
```

## Available Packages

### `github.com/wesleyorama2/lunge/http`

A fluent HTTP client with detailed timing metrics.

### `github.com/wesleyorama2/lunge/config`

Configuration loading and validation for Lunge JSON config files.

### `github.com/wesleyorama2/lunge/perf`

Performance testing library with subpackages:
- `perf/config` - YAML/JSON test configuration
- `perf/metrics` - High-performance metrics with HDR histograms
- `perf/executor` - Load generation strategies
- `perf/rate` - Rate limiting utilities

---

## HTTP Client Usage

The `http` package provides a clean, fluent API for making HTTP requests with detailed timing metrics.

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    lungehttp "github.com/wesleyorama2/lunge/http"
)

func main() {
    // Create a client
    client := lungehttp.NewClient(
        lungehttp.WithBaseURL("https://api.example.com"),
        lungehttp.WithTimeout(30*time.Second),
        lungehttp.WithHeader("Authorization", "Bearer your-token"),
    )
    
    // Make a request
    req := lungehttp.NewRequest("GET", "/users").
        WithQueryParam("limit", "10").
        WithHeader("Accept", "application/json")
    
    resp, err := client.Do(context.Background(), req)
    if err != nil {
        panic(err)
    }
    
    // Check response
    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Success: %v\n", resp.IsSuccess())
    
    // Access timing information
    fmt.Printf("DNS Lookup: %v\n", resp.Timing.DNSLookupTime)
    fmt.Printf("TCP Connect: %v\n", resp.Timing.TCPConnectTime)
    fmt.Printf("TLS Handshake: %v\n", resp.Timing.TLSHandshakeTime)
    fmt.Printf("Time to First Byte: %v\n", resp.Timing.TimeToFirstByte)
    fmt.Printf("Total Time: %v\n", resp.Timing.TotalTime)
    
    // Parse JSON response
    var users []map[string]interface{}
    if err := resp.GetBodyAsJSON(&users); err != nil {
        panic(err)
    }
    fmt.Printf("Got %d users\n", len(users))
}
```

### Auth Token Example

A common use case is getting an auth token and using it for subsequent requests:

```go
package main

import (
    "context"
    "fmt"
    
    lungehttp "github.com/wesleyorama2/lunge/http"
)

func main() {
    // Create auth client
    authClient := lungehttp.NewClient(
        lungehttp.WithBaseURL("https://auth.example.com"),
    )
    
    // Get auth token
    tokenReq := lungehttp.NewRequest("POST", "/oauth/token").
        WithFormData(map[string]string{
            "grant_type":    "client_credentials",
            "client_id":     "your-client-id",
            "client_secret": "your-client-secret",
        })
    
    resp, err := authClient.Do(context.Background(), tokenReq)
    if err != nil {
        panic(err)
    }
    
    var tokenResp struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := resp.GetBodyAsJSON(&tokenResp); err != nil {
        panic(err)
    }
    
    fmt.Printf("Got token (expires in %d seconds)\n", tokenResp.ExpiresIn)
    
    // Create API client with token
    apiClient := lungehttp.NewClient(
        lungehttp.WithBaseURL("https://api.example.com"),
        lungehttp.WithHeader("Authorization", "Bearer "+tokenResp.AccessToken),
    )
    
    // Use the authenticated client
    usersResp, _ := apiClient.Get(context.Background(), "/users")
    fmt.Printf("Users API Status: %d\n", usersResp.StatusCode)
}
```

### Convenience Methods

```go
// GET request
resp, err := client.Get(ctx, "/users")

// POST request with JSON body
resp, err := client.Post(ctx, "/users", map[string]string{
    "name": "John",
    "email": "john@example.com",
})

// PUT request
resp, err := client.Put(ctx, "/users/123", user)

// DELETE request
resp, err := client.Delete(ctx, "/users/123")

// PATCH request
resp, err := client.Patch(ctx, "/users/123", updates)
```

---

## Metrics Collection

The `perf/metrics` package provides high-performance metrics collection with HDR histograms.

### Basic Usage

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/wesleyorama2/lunge/perf/metrics"
)

func main() {
    // Create metrics engine
    engine := metrics.NewEngine()
    defer engine.Stop()
    
    // Record request latencies
    engine.RecordLatency(150*time.Millisecond, "GET /users", true, 1024)
    engine.RecordLatency(200*time.Millisecond, "GET /users", true, 2048)
    engine.RecordLatency(50*time.Millisecond, "GET /health", true, 64)
    engine.RecordLatency(500*time.Millisecond, "POST /users", false, 0) // Failed request
    
    // Get metrics snapshot
    snapshot := engine.GetSnapshot()
    
    fmt.Printf("Total Requests: %d\n", snapshot.TotalRequests)
    fmt.Printf("Success: %d\n", snapshot.SuccessRequests)
    fmt.Printf("Failed: %d\n", snapshot.FailedRequests)
    fmt.Printf("Error Rate: %.2f%%\n", snapshot.ErrorRate*100)
    fmt.Printf("RPS: %.2f\n", snapshot.RPS)
    
    fmt.Printf("\nLatency Percentiles:\n")
    fmt.Printf("  Min: %v\n", snapshot.Latency.Min)
    fmt.Printf("  P50: %v\n", snapshot.Latency.P50)
    fmt.Printf("  P90: %v\n", snapshot.Latency.P90)
    fmt.Printf("  P95: %v\n", snapshot.Latency.P95)
    fmt.Printf("  P99: %v\n", snapshot.Latency.P99)
    fmt.Printf("  Max: %v\n", snapshot.Latency.Max)
}
```

### Phase Tracking

```go
// Track test phases
engine.SetPhase(metrics.PhaseWarmup)
// ... warmup requests ...

engine.SetPhase(metrics.PhaseRampUp)
// ... ramp up phase ...

engine.SetPhase(metrics.PhaseSteady)
// ... main test phase ...

engine.SetPhase(metrics.PhaseRampDown)
// ... ramp down phase ...

engine.SetPhase(metrics.PhaseDone)
```

### Time-Series Data

```go
// Get time-series buckets
timeSeries := engine.GetTimeSeries()

for _, bucket := range timeSeries {
    fmt.Printf("[%s] RPS: %.2f, P95: %v, VUs: %d, Phase: %s\n",
        bucket.Timestamp.Format(time.RFC3339),
        bucket.IntervalRPS,
        bucket.LatencyP95,
        bucket.ActiveVUs,
        bucket.Phase)
}
```

---

## Rate Limiting

The `perf/rate` package provides rate limiting for load generation.

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/wesleyorama2/lunge/perf/rate"
)

func main() {
    // Create a rate limiter at 100 requests/second
    limiter := rate.NewLeakyBucket(100.0)
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    count := 0
    start := time.Now()
    
    for {
        if err := limiter.Wait(ctx); err != nil {
            break // Context cancelled or timed out
        }
        count++
        // Execute your request here
    }
    
    elapsed := time.Since(start)
    actualRPS := float64(count) / elapsed.Seconds()
    
    fmt.Printf("Executed %d requests in %v\n", count, elapsed)
    fmt.Printf("Actual RPS: %.2f\n", actualRPS)
}
```

### Dynamic Rate Changes

```go
// Start at 10 RPS
limiter := rate.NewLeakyBucket(10.0)

// Ramp up over time
go func() {
    for i := 1; i <= 10; i++ {
        time.Sleep(time.Second)
        limiter.SetRate(float64(i * 10)) // 10, 20, 30, ... 100 RPS
    }
}()
```

---

## Configuration Loading

The `config` package loads Lunge JSON configuration files.

```go
package main

import (
    "fmt"
    
    "github.com/wesleyorama2/lunge/config"
)

func main() {
    cfg, err := config.LoadConfig("lunge-config.json")
    if err != nil {
        panic(err)
    }
    
    // Validate configuration
    errors := config.ValidateConfig(cfg)
    if len(errors) > 0 {
        for _, e := range errors {
            fmt.Printf("Validation error: %s\n", e)
        }
        return
    }
    
    // Access environments
    for name, env := range cfg.Environments {
        fmt.Printf("Environment: %s -> %s\n", name, env.BaseURL)
    }
    
    // Access requests
    for name, req := range cfg.Requests {
        fmt.Printf("Request: %s -> %s %s\n", name, req.Method, req.URL)
    }
    
    // Process variables
    env := cfg.Environments["production"]
    for _, req := range cfg.Requests {
        url := config.ProcessEnvironment(req.URL, env.Vars)
        fmt.Printf("Resolved URL: %s\n", url)
    }
}
```

---

## Performance Test Configuration

The `perf/config` package handles YAML-based performance test configurations.

```go
package main

import (
    "fmt"
    
    perfconfig "github.com/wesleyorama2/lunge/perf/config"
)

func main() {
    cfg, err := perfconfig.LoadConfig("test.yaml")
    if err != nil {
        panic(err)
    }
    
    // Validate
    if err := cfg.Validate(); err != nil {
        panic(err)
    }
    
    // Apply defaults
    perfconfig.ApplyDefaults(cfg)
    
    fmt.Printf("Test: %s\n", cfg.Name)
    fmt.Printf("Base URL: %s\n", cfg.Settings.BaseURL)
    
    for name, scenario := range cfg.Scenarios {
        fmt.Printf("Scenario: %s (executor: %s)\n", name, scenario.Executor)
        for _, req := range scenario.Requests {
            fmt.Printf("  - %s %s\n", req.Method, req.URL)
        }
    }
}
```

---

## Complete Example: Custom Load Test Tool

Here's a complete example combining multiple packages:

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    lungehttp "github.com/wesleyorama2/lunge/http"
    "github.com/wesleyorama2/lunge/perf/metrics"
    "github.com/wesleyorama2/lunge/perf/rate"
)

func main() {
    // Configuration
    baseURL := "https://httpbin.org"
    vus := 5
    duration := 10 * time.Second
    targetRPS := 50.0
    
    // Create HTTP client
    client := lungehttp.NewClient(
        lungehttp.WithBaseURL(baseURL),
        lungehttp.WithTimeout(10*time.Second),
    )
    
    // Create metrics engine
    metricsEngine := metrics.NewEngine()
    defer metricsEngine.Stop()
    
    // Create rate limiter
    limiter := rate.NewLeakyBucket(targetRPS)
    
    // Context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    // Start VUs
    var wg sync.WaitGroup
    metricsEngine.SetPhase(metrics.PhaseSteady)
    metricsEngine.SetActiveVUs(vus)
    
    for i := 0; i < vus; i++ {
        wg.Add(1)
        go func(vuID int) {
            defer wg.Done()
            
            for {
                if err := limiter.Wait(ctx); err != nil {
                    return // Context done
                }
                
                // Execute request
                start := time.Now()
                resp, err := client.Get(ctx, "/get")
                latency := time.Since(start)
                
                // Record metrics
                success := err == nil && resp != nil && resp.IsSuccess()
                var bytes int64
                if resp != nil {
                    body, _ := resp.GetBody()
                    bytes = int64(len(body))
                }
                
                metricsEngine.RecordLatency(latency, "GET /get", success, bytes)
            }
        }(i)
    }
    
    // Wait for completion
    wg.Wait()
    metricsEngine.SetPhase(metrics.PhaseDone)
    
    // Print results
    snapshot := metricsEngine.GetSnapshot()
    
    fmt.Println("\n=== Test Results ===")
    fmt.Printf("Duration: %v\n", snapshot.Elapsed)
    fmt.Printf("Total Requests: %d\n", snapshot.TotalRequests)
    fmt.Printf("Success: %d\n", snapshot.SuccessRequests)
    fmt.Printf("Failed: %d\n", snapshot.FailedRequests)
    fmt.Printf("RPS: %.2f\n", snapshot.RPS)
    fmt.Printf("Error Rate: %.2f%%\n", snapshot.ErrorRate*100)
    fmt.Printf("\nLatency:\n")
    fmt.Printf("  P50: %v\n", snapshot.Latency.P50)
    fmt.Printf("  P95: %v\n", snapshot.Latency.P95)
    fmt.Printf("  P99: %v\n", snapshot.Latency.P99)
    fmt.Printf("  Max: %v\n", snapshot.Latency.Max)
}
```

---

## Thread Safety

| Package/Type | Thread-Safe | Notes |
|-------------|-------------|-------|
| `http.Client` | ✅ Yes | Can be shared across goroutines |
| `metrics.Engine` | ✅ Yes | Uses atomic operations and mutexes |
| `rate.LeakyBucket` | ✅ Yes | Uses mutex for rate changes |
| `config.Config` | ⚠️ Read-only | Safe for concurrent reads after loading |

---

## Migration from Internal Packages

If you were previously using internal packages (which shouldn't be imported), migrate to the public packages:

| Old (Internal) | New (Public) |
|---------------|--------------|
| `internal/http` | `github.com/wesleyorama2/lunge/http` |
| `internal/config` | `github.com/wesleyorama2/lunge/config` |
| `internal/performance/v2/metrics` | `github.com/wesleyorama2/lunge/perf/metrics` |
| `internal/performance/v2/rate` | `github.com/wesleyorama2/lunge/perf/rate` |
| `internal/performance/v2/config` | `github.com/wesleyorama2/lunge/perf/config` |

---

## Best Practices

1. **Reuse HTTP clients**: Create one `http.Client` and share it across goroutines
2. **Always call `metrics.Engine.Stop()`**: Use `defer` to ensure cleanup
3. **Set appropriate timeouts**: Configure timeouts on the HTTP client
4. **Use rate limiting**: For accurate load generation, use `rate.LeakyBucket`
5. **Monitor metrics in real-time**: Call `GetSnapshot()` periodically during tests