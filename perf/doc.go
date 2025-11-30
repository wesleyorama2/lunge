// Package perf provides a performance testing library for load testing HTTP APIs.
//
// This package provides high-level APIs for running performance tests programmatically,
// along with subpackages for more granular control:
//
//   - perf/config: Test configuration parsing and validation
//   - perf/metrics: High-performance metrics collection with HDR histograms
//   - perf/executor: Load generation strategies (constant VUs, ramping, arrival rate)
//   - perf/rate: Rate limiting utilities
//
// # Quick Start
//
// For simple use cases, use the high-level RunTest function:
//
//	cfg, _ := config.LoadConfig("test.yaml")
//	result, _ := perf.RunTest(context.Background(), cfg)
//
//	fmt.Printf("Requests: %d\n", result.Metrics.TotalRequests)
//	fmt.Printf("P95: %v\n", result.Metrics.Latency.P95)
//	fmt.Printf("Passed: %v\n", result.Passed)
//
// # Custom Test Configuration
//
// You can also build test configurations programmatically:
//
//	cfg := &config.TestConfig{
//	    Name: "My API Test",
//	    Settings: config.GlobalSettings{
//	        BaseURL: "https://api.example.com",
//	        Timeout: 30 * time.Second,
//	    },
//	    Scenarios: map[string]*config.ScenarioConfig{
//	        "smoke": {
//	            Executor: "constant-vus",
//	            VUs:      5,
//	            Duration: "30s",
//	            Requests: []config.RequestConfig{
//	                {Method: "GET", URL: "{{baseUrl}}/health"},
//	            },
//	        },
//	    },
//	}
//
// # Metrics Collection
//
// For custom metrics collection, use the metrics subpackage directly:
//
//	engine := metrics.NewEngine()
//	defer engine.Stop()
//
//	// Record latencies as requests complete
//	engine.RecordLatency(150*time.Millisecond, "GET /users", true, 1024)
//
//	// Get current metrics
//	snapshot := engine.GetSnapshot()
//	fmt.Printf("RPS: %.2f\n", snapshot.RPS)
//
// # Rate Limiting
//
// For rate-limited operations, use the rate subpackage:
//
//	limiter := rate.NewLeakyBucket(100.0) // 100 req/sec
//
//	for i := 0; i < 1000; i++ {
//	    if err := limiter.Wait(ctx); err != nil {
//	        break // Context cancelled
//	    }
//	    // Execute request
//	}
package perf
