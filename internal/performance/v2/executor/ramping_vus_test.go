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

// createRampingVUsTestServer creates a test HTTP server
func createRampingVUsTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
}

// createRampingVUsTestScenario creates a scenario for testing
func createRampingVUsTestScenario(serverURL string) *v2.Scenario {
	return &v2.Scenario{
		Name: "ramping-vus-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test-request",
				Method: "GET",
				URL:    serverURL,
			},
		},
	}
}

func TestNewRampingVUs(t *testing.T) {
	e := executor.NewRampingVUs()
	if e == nil {
		t.Fatal("NewRampingVUs() returned nil")
	}
}

func TestRampingVUs_Type(t *testing.T) {
	e := executor.NewRampingVUs()
	if e.Type() != executor.TypeRampingVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingVUs)
	}
}

func TestRampingVUs_Init_ValidConfig(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 10},
			{Duration: 1 * time.Minute, Target: 10},
			{Duration: 30 * time.Second, Target: 0},
		},
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestRampingVUs_Init_InvalidType(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeConstantVUs, // Wrong type
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 10},
		},
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for wrong type, got nil")
	}
}

func TestRampingVUs_Init_MissingStages(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type:   executor.TypeRampingVUs,
		Stages: []executor.Stage{}, // No stages
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for empty stages, got nil")
	}
}

func TestRampingVUs_Init_NilStages(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type:   executor.TypeRampingVUs,
		Stages: nil, // Nil stages
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for nil stages, got nil")
	}
}

func TestRampingVUs_Init_WithPacing(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 5},
		},
		Pacing: &executor.PacingConfig{
			Type:     executor.PacingConstant,
			Duration: 100 * time.Millisecond,
		},
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestRampingVUs_Init_WithRandomPacing(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 5},
		},
		Pacing: &executor.PacingConfig{
			Type: executor.PacingRandom,
			Min:  50 * time.Millisecond,
			Max:  200 * time.Millisecond,
		},
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestRampingVUs_Init_WithGracefulStop(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 5},
		},
		GracefulStop: 10 * time.Second,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestRampingVUs_Run_Basic(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 200 * time.Millisecond, Target: 3},
			{Duration: 200 * time.Millisecond, Target: 0},
		},
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

	// Should have run for approximately the total duration (400ms)
	if elapsed < 350*time.Millisecond || elapsed > 1*time.Second {
		t.Errorf("Run() elapsed = %v, want ~400ms", elapsed)
	}
}

func TestRampingVUs_Run_ThreeStages(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 150 * time.Millisecond, Target: 5, Name: "ramp-up"},
			{Duration: 150 * time.Millisecond, Target: 5, Name: "steady"},
			{Duration: 150 * time.Millisecond, Target: 0, Name: "ramp-down"},
		},
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

	stats := e.GetStats()
	t.Logf("Completed %d iterations over 3 stages", stats.Iterations)
}

func TestRampingVUs_Run_WithPacing(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 300 * time.Millisecond, Target: 2},
		},
		Pacing: &executor.PacingConfig{
			Type:     executor.PacingConstant,
			Duration: 50 * time.Millisecond,
		},
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

	stats := e.GetStats()
	t.Logf("Completed %d iterations with pacing", stats.Iterations)
}

func TestRampingVUs_GetProgress_BeforeRun(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 5},
		},
	}

	_ = e.Init(context.Background(), config)

	progress := e.GetProgress()
	if progress != 0.0 {
		t.Errorf("Before Run(), GetProgress() = %v, want 0.0", progress)
	}
}

func TestRampingVUs_GetProgress_AfterRun(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 1},
		},
	}

	_ = e.Init(context.Background(), config)

	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	progress := e.GetProgress()
	if progress != 1.0 {
		t.Errorf("After Run(), GetProgress() = %v, want 1.0", progress)
	}
}

func TestRampingVUs_GetActiveVUs_BeforeRun(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 5},
		},
	}

	_ = e.Init(context.Background(), config)

	activeVUs := e.GetActiveVUs()
	if activeVUs != 0 {
		t.Errorf("Before Run(), GetActiveVUs() = %d, want 0", activeVUs)
	}
}

func TestRampingVUs_GetActiveVUs_AfterRun(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 2},
		},
	}

	_ = e.Init(context.Background(), config)

	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	// After completion, active VUs should be 0
	activeVUs := e.GetActiveVUs()
	if activeVUs != 0 {
		t.Errorf("After Run(), GetActiveVUs() = %d, want 0", activeVUs)
	}
}

func TestRampingVUs_GetStats(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Minute, Target: 10, Name: "ramp-up"},
			{Duration: 2 * time.Minute, Target: 10, Name: "steady"},
			{Duration: 30 * time.Second, Target: 0, Name: "ramp-down"},
		},
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

func TestRampingVUs_GetStats_DuringRun(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 300 * time.Millisecond, Target: 3},
			{Duration: 200 * time.Millisecond, Target: 0},
		},
	}

	_ = e.Init(context.Background(), config)

	// Run to completion
	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	// Check stats after completion
	stats := e.GetStats()
	if stats.Elapsed < 400*time.Millisecond {
		t.Errorf("Stats.Elapsed = %v, want >= 400ms", stats.Elapsed)
	}
}

func TestRampingVUs_Stop(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 10 * time.Second, Target: 5}, // Long duration
		},
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Use context cancellation for safe stopping
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

func TestRampingVUs_Stop_BeforeRun(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 5},
		},
	}

	_ = e.Init(context.Background(), config)

	// Stop before Run - should not panic
	err := e.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() before Run() error = %v", err)
	}
}

func TestRampingVUs_ContextCancellation(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 10 * time.Second, Target: 5}, // Long duration
		},
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

func TestRampingVUs_Interface(t *testing.T) {
	// Verify that RampingVUs implements Executor interface
	var _ executor.Executor = (*executor.RampingVUs)(nil)
}

func TestRampingVUs_MetricsPhase(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 2},
			{Duration: 100 * time.Millisecond, Target: 0},
		},
	}

	_ = e.Init(context.Background(), config)

	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	// After run, phase should be Done
	phase := metricsEngine.GetPhase()
	if phase != metrics.PhaseDone {
		t.Errorf("After Run(), phase = %v, want %v", phase, metrics.PhaseDone)
	}
}

func TestRampingVUs_TotalDuration(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 10},
			{Duration: 1 * time.Minute, Target: 10},
			{Duration: 30 * time.Second, Target: 0},
		},
	}

	expectedDuration := 2 * time.Minute
	if config.TotalDuration() != expectedDuration {
		t.Errorf("TotalDuration() = %v, want %v", config.TotalDuration(), expectedDuration)
	}
}

func TestRampingVUs_SingleStage(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 200 * time.Millisecond, Target: 3},
		},
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx := context.Background()
	err = e.Run(ctx, scheduler, metricsEngine)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	stats := e.GetStats()
	if stats.TotalStages != 1 {
		t.Errorf("TotalStages = %d, want 1", stats.TotalStages)
	}
}

func TestRampingVUs_StageNames(t *testing.T) {
	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Second, Target: 10, Name: "warmup"},
			{Duration: 2 * time.Second, Target: 50, Name: "ramp"},
			{Duration: 1 * time.Second, Target: 50, Name: "steady"},
		},
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

func TestRampingVUs_ZeroTargetStage(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	// Test with a stage that ramps to 0
	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 2},
			{Duration: 200 * time.Millisecond, Target: 0}, // Ramp down to 0
		},
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

func TestRampingVUs_NoPacing(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 100 * time.Millisecond, Target: 2},
		},
		Pacing: &executor.PacingConfig{
			Type: executor.PacingNone,
		},
	}

	_ = e.Init(context.Background(), config)

	ctx := context.Background()
	err := e.Run(ctx, scheduler, metricsEngine)
	if err != nil {
		t.Fatalf("Run() with PacingNone error = %v", err)
	}
}

func TestRampingVUs_ConcurrentAccess(t *testing.T) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewRampingVUs()

	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 300 * time.Millisecond, Target: 3},
			{Duration: 200 * time.Millisecond, Target: 0},
		},
	}

	_ = e.Init(context.Background(), config)

	// Run to completion (safe approach)
	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	// Access methods after completion (no race)
	progress := e.GetProgress()
	if progress != 1.0 {
		t.Errorf("GetProgress() = %v, want 1.0", progress)
	}

	activeVUs := e.GetActiveVUs()
	if activeVUs != 0 {
		t.Errorf("GetActiveVUs() = %d, want 0", activeVUs)
	}

	stats := e.GetStats()
	if stats.TotalStages != 2 {
		t.Errorf("Stats.TotalStages = %d, want 2", stats.TotalStages)
	}
}

// Benchmark for ramping VUs execution
func BenchmarkRampingVUs_Run(b *testing.B) {
	server := createRampingVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createRampingVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	for i := 0; i < b.N; i++ {
		e := executor.NewRampingVUs()
		config := &executor.Config{
			Type: executor.TypeRampingVUs,
			Stages: []executor.Stage{
				{Duration: 50 * time.Millisecond, Target: 2},
				{Duration: 50 * time.Millisecond, Target: 0},
			},
		}
		_ = e.Init(context.Background(), config)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = e.Run(ctx, scheduler, metricsEngine)
		cancel()
	}
}
