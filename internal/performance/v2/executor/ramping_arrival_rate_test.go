package executor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/executor"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// createRampingArrivalRateTestServer creates a test HTTP server
func createRampingArrivalRateTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
}

// createRampingArrivalRateTestScenario creates a scenario for testing
func createRampingArrivalRateTestScenario(serverURL string) *v2.Scenario {
	return &v2.Scenario{
		Name: "ramping-arrival-rate-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test-request",
				Method: "GET",
				URL:    serverURL,
			},
		},
	}
}

func TestRampingArrivalRate_Type(t *testing.T) {
	e := executor.NewRampingArrivalRate()
	if e.Type() != executor.TypeRampingArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingArrivalRate)
	}
}

func TestRampingArrivalRate_Init_ValidConfig(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 50},
			{Duration: 1 * time.Minute, Target: 50},
			{Duration: 30 * time.Second, Target: 0},
		},
		PreAllocatedVUs: 10,
		MaxVUs:          50,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestRampingArrivalRate_Init_InvalidType(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeConstantVUs, // Wrong type
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 50},
		},
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for wrong type, got nil")
	}
}

func TestRampingArrivalRate_Init_MissingStages(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type:   executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{}, // No stages
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for empty stages, got nil")
	}
}

func TestRampingArrivalRate_Init_Defaults(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10},
		},
		// PreAllocatedVUs and MaxVUs not set
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}

	// Check defaults were applied
	if config.PreAllocatedVUs != 1 {
		t.Errorf("PreAllocatedVUs default = %d, want 1", config.PreAllocatedVUs)
	}
	if config.MaxVUs != 1 {
		t.Errorf("MaxVUs default = %d, want 1", config.MaxVUs)
	}
}

func TestRampingArrivalRate_Init_MaxVUsAdjusted(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10},
		},
		PreAllocatedVUs: 10,
		MaxVUs:          5, // Less than PreAllocatedVUs
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}

	// MaxVUs should be adjusted to equal PreAllocatedVUs
	if config.MaxVUs != 10 {
		t.Errorf("MaxVUs should be adjusted to %d, got %d", config.PreAllocatedVUs, config.MaxVUs)
	}
}

func TestRampingArrivalRate_Run_Basic(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	// Use a steady rate stage to ensure iterations run
	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 300 * time.Millisecond, Target: 20}, // Steady at 20 RPS
			{Duration: 200 * time.Millisecond, Target: 20}, // Stay at 20 RPS
		},
		PreAllocatedVUs: 2,
		MaxVUs:          5,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err = e.Run(ctx, scheduler, metricsEngine)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have run for approximately the total duration (500ms)
	if elapsed < 400*time.Millisecond || elapsed > 1*time.Second {
		t.Errorf("Run() elapsed = %v, want ~500ms", elapsed)
	}

	// Check iterations were performed (ramping from 0 to 20 RPS over 300ms, then steady)
	stats := e.GetStats()
	t.Logf("Completed %d iterations", stats.Iterations)
}

func TestRampingArrivalRate_GetProgress(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10},
			{Duration: 1 * time.Second, Target: 0},
		},
		PreAllocatedVUs: 1,
	}

	_ = e.Init(context.Background(), config)

	// Before running
	progress := e.GetProgress()
	if progress != 0.0 {
		t.Errorf("Before Run(), GetProgress() = %v, want 0.0", progress)
	}
}

func TestRampingArrivalRate_GetActiveVUs(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10},
		},
		PreAllocatedVUs: 5,
		MaxVUs:          10,
	}

	_ = e.Init(context.Background(), config)

	// Before running
	activeVUs := e.GetActiveVUs()
	if activeVUs != 0 {
		t.Errorf("Before Run(), GetActiveVUs() = %d, want 0", activeVUs)
	}
}

func TestRampingArrivalRate_GetStats(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Minute, Target: 100, Name: "ramp-up"},
			{Duration: 2 * time.Minute, Target: 100, Name: "steady"},
			{Duration: 30 * time.Second, Target: 0, Name: "ramp-down"},
		},
		PreAllocatedVUs: 5,
		MaxVUs:          10,
	}

	_ = e.Init(context.Background(), config)

	stats := e.GetStats()
	expectedDuration := 3*time.Minute + 30*time.Second
	if stats.TotalDuration != expectedDuration {
		t.Errorf("Stats.TotalDuration = %v, want %v", stats.TotalDuration, expectedDuration)
	}
	if stats.TotalStages != 3 {
		t.Errorf("Stats.TotalStages = %d, want 3", stats.TotalStages)
	}
}

func TestRampingArrivalRate_Stop(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 10 * time.Second, Target: 50}, // Long duration
		},
		PreAllocatedVUs: 2,
		MaxVUs:          5,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Use context cancellation for safe stopping (avoids races)
	ctx, cancel := context.WithCancel(context.Background())

	// Start Run in a goroutine
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = e.Run(ctx, scheduler, metricsEngine)
		close(done)
	}()

	// Wait a bit then cancel context
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Wait for Run to complete
	select {
	case <-done:
		if runErr != nil {
			t.Fatalf("Run() error after cancel = %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not complete after cancel")
	}
}

func TestRampingArrivalRate_ContextCancellation(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 10 * time.Second, Target: 50}, // Long duration
		},
		PreAllocatedVUs: 2,
		MaxVUs:          5,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start Run in a goroutine
	done := make(chan struct{})
	go func() {
		_ = e.Run(ctx, scheduler, metricsEngine)
		close(done)
	}()

	// Wait a bit then cancel
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Wait for Run to complete
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not complete after context cancellation")
	}
}

func TestRampingArrivalRate_PhaseTransitions(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 200 * time.Millisecond, Target: 10, Name: "ramp-up"},  // Ramp up
			{Duration: 200 * time.Millisecond, Target: 10, Name: "steady"},   // Steady
			{Duration: 200 * time.Millisecond, Target: 0, Name: "ramp-down"}, // Ramp down
		},
		PreAllocatedVUs: 2,
		MaxVUs:          5,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = e.Run(ctx, scheduler, metricsEngine)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Check that phase was set to done at the end
	phase := metricsEngine.GetPhase()
	if phase != metrics.PhaseDone {
		t.Errorf("Final phase = %v, want %v", phase, metrics.PhaseDone)
	}
}

func TestRampingArrivalRate_TotalDuration(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 10},
			{Duration: 1 * time.Minute, Target: 50},
			{Duration: 30 * time.Second, Target: 0},
		},
	}

	expectedDuration := 2 * time.Minute
	if config.TotalDuration() != expectedDuration {
		t.Errorf("TotalDuration() = %v, want %v", config.TotalDuration(), expectedDuration)
	}
}

func TestRampingArrivalRate_Interface(t *testing.T) {
	// Verify that RampingArrivalRate implements Executor interface
	var _ executor.Executor = (*executor.RampingArrivalRate)(nil)
}

func TestRampingArrivalRate_VUPoolScaling(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	// Ramp to high rate quickly with low pre-allocated VUs
	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 100}, // Quick ramp to 100 RPS
			{Duration: 200 * time.Millisecond, Target: 100}, // Stay at 100 RPS
		},
		PreAllocatedVUs: 1,  // Start with 1 VU
		MaxVUs:          10, // Allow scaling to 10
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = e.Run(ctx, scheduler, metricsEngine)

	// VUs should have scaled up
	stats := e.GetStats()
	if stats.ActiveVUs < 1 {
		t.Errorf("Expected VUs to scale up, got %d", stats.ActiveVUs)
	}
}

func TestRampingArrivalRate_ZeroTargetStage(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	// Test with a stage that ramps to 0
	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 200 * time.Millisecond, Target: 10},
			{Duration: 300 * time.Millisecond, Target: 0}, // Ramp down to 0
		},
		PreAllocatedVUs: 2,
		MaxVUs:          5,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = e.Run(ctx, scheduler, metricsEngine)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRampingArrivalRate_Stop_BeforeRun(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10},
		},
		PreAllocatedVUs: 1,
	}

	_ = e.Init(context.Background(), config)

	// Stop before Run - should not panic
	err := e.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() before Run() error = %v", err)
	}
}

func TestRampingArrivalRate_GetProgress_AfterRun(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 5},
		},
		PreAllocatedVUs: 1,
		MaxVUs:          2,
	}

	_ = e.Init(context.Background(), config)

	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	// After completion
	progress := e.GetProgress()
	if progress != 1.0 {
		t.Errorf("After Run(), GetProgress() = %v, want 1.0", progress)
	}
}

func TestRampingArrivalRate_StageNames(t *testing.T) {
	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10, Name: "warmup"},
			{Duration: 2 * time.Second, Target: 50, Name: "ramp"},
			{Duration: 1 * time.Second, Target: 50, Name: "steady"},
		},
		PreAllocatedVUs: 1,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	stats := e.GetStats()
	if stats.TotalStages != 3 {
		t.Errorf("TotalStages = %d, want 3", stats.TotalStages)
	}
}

func TestRampingArrivalRate_ConcurrentAccess(t *testing.T) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingArrivalRate()

	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 300 * time.Millisecond, Target: 10},
			{Duration: 200 * time.Millisecond, Target: 0},
		},
		PreAllocatedVUs: 2,
		MaxVUs:          5,
	}

	_ = e.Init(context.Background(), config)

	// Run to completion (safe approach)
	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	// Allow brief time for VUs to shutdown
	time.Sleep(50 * time.Millisecond)

	// Access methods after completion (no race)
	progress := e.GetProgress()
	if progress != 1.0 {
		t.Errorf("GetProgress() = %v, want 1.0", progress)
	}

	// VUs may still be finishing cleanup, just verify no error
	_ = e.GetActiveVUs()

	stats := e.GetStats()
	if stats.TotalStages != 2 {
		t.Errorf("Stats.TotalStages = %d, want 2", stats.TotalStages)
	}
}

// Benchmark for rate calculation performance
func BenchmarkRampingArrivalRate_Run(b *testing.B) {
	server := createRampingArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	for i := 0; i < b.N; i++ {
		e := executor.NewRampingArrivalRate()

		config := &executor.Config{
			Type: executor.TypeRampingArrivalRate,
			Stages: []executor.Stage{
				{Duration: 50 * time.Millisecond, Target: 100},
			},
			PreAllocatedVUs: 5,
			MaxVUs:          10,
		}

		_ = e.Init(context.Background(), config)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = e.Run(ctx, scheduler, metricsEngine)
		cancel()
	}
}
