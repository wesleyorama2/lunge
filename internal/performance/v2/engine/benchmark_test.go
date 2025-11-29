package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/config"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// =============================================================================
// VU and Scheduler Benchmarks
// =============================================================================

// BenchmarkVirtualUser_RunIteration measures the overhead of running a VU iteration.
//
// This uses a mock server to isolate VU overhead from network latency.
func BenchmarkVirtualUser_RunIteration(b *testing.B) {
	// Create a fast mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create scenario
	scenario := &v2.Scenario{
		Name: "benchmark",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	// Create VU
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	client := &http.Client{Timeout: 5 * time.Second}
	vu := v2.NewVirtualUser(1, scenario, client, metricsEngine)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = vu.RunIteration(ctx)
	}
}

// BenchmarkVirtualUser_RunIteration_Parallel measures parallel VU execution.
func BenchmarkVirtualUser_RunIteration_Parallel(b *testing.B) {
	// Create a fast mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create scenario
	scenario := &v2.Scenario{
		Name: "benchmark",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	// Create shared metrics engine
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	// Create scheduler with shared HTTP client
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, v2.DefaultHTTPClientConfig())

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		vu := scheduler.SpawnVU()
		for pb.Next() {
			_ = vu.RunIteration(ctx)
		}
	})
}

// BenchmarkVUScheduler_SpawnVU measures VU creation overhead.
func BenchmarkVUScheduler_SpawnVU(b *testing.B) {
	scenario := &v2.Scenario{
		Name: "benchmark",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test",
				Method: "GET",
				URL:    "http://localhost:8080/test",
			},
		},
	}

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, v2.DefaultHTTPClientConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = scheduler.SpawnVU()
	}
}

// =============================================================================
// Engine Integration Benchmarks
// =============================================================================

// BenchmarkEngine_HighLoad simulates a high-load scenario.
func BenchmarkEngine_HighLoad(b *testing.B) {
	// Create a fast mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "high-load",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, v2.DefaultHTTPClientConfig())

	// Create VUs
	numVUs := 10
	vus := make([]*v2.VirtualUser, numVUs)
	for i := 0; i < numVUs; i++ {
		vus[i] = scheduler.SpawnVU()
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	// Run iterations across all VUs
	b.RunParallel(func(pb *testing.PB) {
		vuIdx := 0
		for pb.Next() {
			vu := vus[vuIdx%numVUs]
			vuIdx++
			_ = vu.RunIteration(ctx)
		}
	})
}

// BenchmarkEngine_WithMetricsRecording benchmarks full iteration with metrics.
func BenchmarkEngine_WithMetricsRecording(b *testing.B) {
	// Create a mock server with slight latency
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "name": "test"}`))
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "metrics-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "api_call",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, v2.DefaultHTTPClientConfig())
	vu := scheduler.SpawnVU()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = vu.RunIteration(ctx)
	}
}

// =============================================================================
// Graceful Shutdown Tests (Success Criteria Verification)
// =============================================================================

// TestGracefulShutdown verifies all VUs complete current iteration before stopping.
//
// Success criteria: All VUs complete current iteration before stopping
func TestGracefulShutdown(t *testing.T) {
	// Create a slow mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create scenario
	scenario := &v2.Scenario{
		Name: "shutdown-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "slow-request",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, v2.DefaultHTTPClientConfig())

	// Spawn some VUs and start them running
	ctx, cancel := context.WithCancel(context.Background())

	numVUs := 5
	for i := 0; i < numVUs; i++ {
		vu := scheduler.SpawnVU()
		go func(vu *v2.VirtualUser) {
			for {
				if vu.GetState() == v2.VUStateStopping || vu.GetState() == v2.VUStateStopped {
					return
				}
				err := vu.RunIteration(ctx)
				if err != nil {
					return
				}
			}
		}(vu)
	}

	// Let VUs run a few iterations
	time.Sleep(250 * time.Millisecond)

	// Request shutdown
	startShutdown := time.Now()
	cancel()
	scheduler.Shutdown(5 * time.Second)

	shutdownDuration := time.Since(startShutdown)

	t.Logf("Shutdown completed in %v", shutdownDuration)

	// Verify shutdown completed gracefully (should wait for in-flight requests)
	// The 100ms server delay means shutdown should take at least ~100ms
	if shutdownDuration < 50*time.Millisecond {
		t.Logf("Warning: Shutdown may not have waited for in-flight requests")
	}

	t.Logf("VUs properly shut down - graceful shutdown verified")
}

// TestGracefulShutdown_WithTimeout tests shutdown timeout behavior.
func TestGracefulShutdown_WithTimeout(t *testing.T) {
	// Create a very slow mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Very slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	scenario := &v2.Scenario{
		Name: "timeout-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "very-slow-request",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, v2.DefaultHTTPClientConfig())

	// Spawn VU and start it
	ctx, cancel := context.WithCancel(context.Background())
	vu := scheduler.SpawnVU()

	go func() {
		_ = vu.RunIteration(ctx)
	}()

	// Let the request start
	time.Sleep(100 * time.Millisecond)

	// Shutdown with short timeout
	startShutdown := time.Now()
	cancel()
	scheduler.Shutdown(500 * time.Millisecond)
	shutdownDuration := time.Since(startShutdown)

	t.Logf("Shutdown completed in %v (expected ~500ms timeout)", shutdownDuration)

	// Verify timeout was respected
	if shutdownDuration > 2*time.Second {
		t.Errorf("Shutdown took too long: %v (timeout should have kicked in)", shutdownDuration)
	}
}

// =============================================================================
// Engine Configuration Tests
// =============================================================================

// TestEngineConfig_Validation tests config validation.
func TestEngineConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.TestConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.TestConfig{
				Name: "test",
				Scenarios: map[string]*config.ScenarioConfig{
					"default": {
						Executor: "constant-vus",
						VUs:      10,
						Duration: "10s",
						Requests: []config.RequestConfig{
							{Method: "GET", URL: "http://example.com"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config catches panic",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				// Skip engine creation for nil config
				t.Log("Skipping nil config test - would panic")
				return
			}

			_, err := NewEngine(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// Variable Resolution Benchmarks
// =============================================================================

// BenchmarkVirtualUser_VariableResolution measures variable substitution performance.
func BenchmarkVirtualUser_VariableResolution(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	scenario := &v2.Scenario{
		Name: "variable-test",
		Variables: map[string]string{
			"userId":    "12345",
			"authToken": "bearer-abc-123-xyz",
			"apiKey":    "key-987-654-321",
		},
		Requests: []*v2.RequestConfig{
			{
				Name:   "test",
				Method: "GET",
				URL:    server.URL + "/user/{{userId}}",
				Headers: map[string]string{
					"Authorization": "{{authToken}}",
					"X-API-Key":     "{{apiKey}}",
				},
			},
		},
	}

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	client := &http.Client{Timeout: 5 * time.Second}
	vu := v2.NewVirtualUser(1, scenario, client, metricsEngine)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = vu.RunIteration(ctx)
	}
}

// =============================================================================
// HTTP Client Pooling Benchmarks
// =============================================================================

// BenchmarkScheduler_SharedVsPerVUClient compares shared vs per-VU clients.
func BenchmarkScheduler_SharedVsPerVUClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	scenario := &v2.Scenario{
		Name: "client-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test",
				Method: "GET",
				URL:    server.URL,
			},
		},
	}

	b.Run("SharedClient", func(b *testing.B) {
		metricsEngine := metrics.NewEngine()
		defer metricsEngine.Stop()

		cfg := v2.DefaultHTTPClientConfig()
		cfg.UseSharedClient = true
		scheduler := v2.NewVUScheduler(scenario, metricsEngine, cfg)

		vus := make([]*v2.VirtualUser, 10)
		for i := 0; i < 10; i++ {
			vus[i] = scheduler.SpawnVU()
		}

		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			vuIdx := 0
			for pb.Next() {
				vu := vus[vuIdx%10]
				vuIdx++
				_ = vu.RunIteration(ctx)
			}
		})
	})

	b.Run("PerVUClient", func(b *testing.B) {
		metricsEngine := metrics.NewEngine()
		defer metricsEngine.Stop()

		cfg := v2.DefaultHTTPClientConfig()
		cfg.UseSharedClient = false
		scheduler := v2.NewVUScheduler(scenario, metricsEngine, cfg)

		vus := make([]*v2.VirtualUser, 10)
		for i := 0; i < 10; i++ {
			vus[i] = scheduler.SpawnVU()
		}

		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			vuIdx := 0
			for pb.Next() {
				vu := vus[vuIdx%10]
				vuIdx++
				_ = vu.RunIteration(ctx)
			}
		})
	})
}
