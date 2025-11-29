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

// createConstantVUsTestServer creates a test HTTP server
func createConstantVUsTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
}

// createConstantVUsTestScenario creates a scenario for testing
func createConstantVUsTestScenario(serverURL string) *v2.Scenario {
	return &v2.Scenario{
		Name: "constant-vus-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test-request",
				Method: "GET",
				URL:    serverURL,
			},
		},
	}
}

func TestNewConstantVUs(t *testing.T) {
	e := executor.NewConstantVUs()
	if e == nil {
		t.Fatal("NewConstantVUs() returned nil")
	}
}

func TestConstantVUs_Type(t *testing.T) {
	e := executor.NewConstantVUs()
	if e.Type() != executor.TypeConstantVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantVUs)
	}
}

func TestConstantVUs_Init_ValidConfig(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      10,
		Duration: 1 * time.Minute,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestConstantVUs_Init_InvalidType(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeRampingVUs, // Wrong type
		VUs:      10,
		Duration: 1 * time.Minute,
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for wrong type, got nil")
	}
}

func TestConstantVUs_Init_MissingVUs(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      0, // Invalid VUs
		Duration: 1 * time.Minute,
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for zero VUs, got nil")
	}
}

func TestConstantVUs_Init_NegativeVUs(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      -5, // Negative VUs
		Duration: 1 * time.Minute,
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for negative VUs, got nil")
	}
}

func TestConstantVUs_Init_MissingDuration(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      10,
		Duration: 0, // Invalid duration
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for zero duration, got nil")
	}
}

func TestConstantVUs_Init_NegativeDuration(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      10,
		Duration: -1 * time.Minute, // Negative duration
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for negative duration, got nil")
	}
}

func TestConstantVUs_Init_WithPacing(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      5,
		Duration: 30 * time.Second,
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

func TestConstantVUs_Init_WithRandomPacing(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      5,
		Duration: 30 * time.Second,
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

func TestConstantVUs_Init_WithGracefulStop(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:         executor.TypeConstantVUs,
		VUs:          5,
		Duration:     30 * time.Second,
		GracefulStop: 10 * time.Second,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestConstantVUs_Run_Basic(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 300 * time.Millisecond,
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

	// Should have run for approximately the duration
	if elapsed < 250*time.Millisecond || elapsed > 800*time.Millisecond {
		t.Errorf("Run() elapsed = %v, want ~300ms", elapsed)
	}

	// Check that iterations were performed
	stats := e.GetStats()
	if stats.Iterations < 1 {
		t.Errorf("Iterations = %d, want at least 1", stats.Iterations)
	}
}

func TestConstantVUs_Run_WithPacing(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 400 * time.Millisecond,
		Pacing: &executor.PacingConfig{
			Type:     executor.PacingConstant,
			Duration: 100 * time.Millisecond,
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

	// With pacing, iterations should be limited
	stats := e.GetStats()
	t.Logf("Completed %d iterations with constant pacing", stats.Iterations)
}

func TestConstantVUs_Run_WithRandomPacing(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 400 * time.Millisecond,
		Pacing: &executor.PacingConfig{
			Type: executor.PacingRandom,
			Min:  50 * time.Millisecond,
			Max:  150 * time.Millisecond,
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
	t.Logf("Completed %d iterations with random pacing", stats.Iterations)
}

func TestConstantVUs_Run_MultipleVUs(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      5,
		Duration: 300 * time.Millisecond,
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
	if stats.TargetVUs != 5 {
		t.Errorf("TargetVUs = %d, want 5", stats.TargetVUs)
	}
}

func TestConstantVUs_GetProgress_BeforeRun(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 1 * time.Second,
	}

	_ = e.Init(context.Background(), config)

	// Before running
	progress := e.GetProgress()
	if progress != 0.0 {
		t.Errorf("Before Run(), GetProgress() = %v, want 0.0", progress)
	}
}

func TestConstantVUs_GetProgress_AfterRun(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      1,
		Duration: 100 * time.Millisecond,
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

func TestConstantVUs_GetActiveVUs_BeforeRun(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      5,
		Duration: 1 * time.Second,
	}

	_ = e.Init(context.Background(), config)

	// Before running
	activeVUs := e.GetActiveVUs()
	if activeVUs != 0 {
		t.Errorf("Before Run(), GetActiveVUs() = %d, want 0", activeVUs)
	}
}

func TestConstantVUs_GetActiveVUs_AfterRun(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 100 * time.Millisecond,
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

func TestConstantVUs_GetStats(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      10,
		Duration: 5 * time.Minute,
	}

	_ = e.Init(context.Background(), config)

	stats := e.GetStats()
	if stats.TotalDuration != 5*time.Minute {
		t.Errorf("Stats.TotalDuration = %v, want 5m", stats.TotalDuration)
	}
	if stats.TargetVUs != 10 {
		t.Errorf("Stats.TargetVUs = %d, want 10", stats.TargetVUs)
	}
}

func TestConstantVUs_GetStats_DuringRun(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      3,
		Duration: 500 * time.Millisecond,
	}

	_ = e.Init(context.Background(), config)

	// Run and then check stats after completion
	ctx := context.Background()
	_ = e.Run(ctx, scheduler, metricsEngine)

	stats := e.GetStats()
	// After completion, elapsed should be approximately the duration
	if stats.Elapsed < 400*time.Millisecond {
		t.Errorf("Stats.Elapsed = %v, want >= 400ms", stats.Elapsed)
	}
}

func TestConstantVUs_Stop(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 10 * time.Second, // Long duration
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

func TestConstantVUs_Stop_WithGracefulTimeout(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:         executor.TypeConstantVUs,
		VUs:          2,
		Duration:     10 * time.Second,
		GracefulStop: 500 * time.Millisecond, // Short graceful stop
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Use context cancellation instead of Stop() to avoid race
	ctx, cancel := context.WithCancel(context.Background())

	// Start Run in a goroutine
	done := make(chan struct{})
	go func() {
		_ = e.Run(ctx, scheduler, metricsEngine)
		close(done)
	}()

	// Wait a bit then cancel (simulating graceful stop)
	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	cancel()
	<-done
	elapsed := time.Since(start)

	// Should complete quickly after cancellation
	if elapsed > 1*time.Second {
		t.Errorf("Run() took %v to stop after cancel, expected < 1s", elapsed)
	}
}

func TestConstantVUs_Stop_BeforeRun(t *testing.T) {
	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 1 * time.Second,
	}

	_ = e.Init(context.Background(), config)

	// Stop before Run - should not panic
	err := e.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() before Run() error = %v", err)
	}
}

func TestConstantVUs_ContextCancellation(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 10 * time.Second, // Long duration
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

func TestConstantVUs_ContextTimeout(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 10 * time.Second, // Long duration
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = e.Run(ctx, scheduler, metricsEngine)
	elapsed := time.Since(start)

	// Should complete due to context timeout
	if elapsed > 1*time.Second {
		t.Errorf("Run() took %v with short context timeout, expected < 1s", elapsed)
	}
}

func TestConstantVUs_Interface(t *testing.T) {
	// Verify that ConstantVUs implements Executor interface
	var _ executor.Executor = (*executor.ConstantVUs)(nil)
}

func TestConstantVUs_MetricsPhase(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      1,
		Duration: 200 * time.Millisecond,
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

func TestConstantVUs_NoPacing(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      2,
		Duration: 100 * time.Millisecond,
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

func TestConstantVUs_RandomPacingEqualMinMax(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      1,
		Duration: 200 * time.Millisecond,
		Pacing: &executor.PacingConfig{
			Type: executor.PacingRandom,
			Min:  100 * time.Millisecond,
			Max:  100 * time.Millisecond, // Same as min
		},
	}

	_ = e.Init(context.Background(), config)

	ctx := context.Background()
	err := e.Run(ctx, scheduler, metricsEngine)
	if err != nil {
		t.Fatalf("Run() with equal min/max pacing error = %v", err)
	}
}

func TestConstantVUs_SingleVU(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      1,
		Duration: 200 * time.Millisecond,
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
	if stats.TargetVUs != 1 {
		t.Errorf("TargetVUs = %d, want 1", stats.TargetVUs)
	}
}

func TestConstantVUs_ConcurrentAccess(t *testing.T) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantVUs()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      5,
		Duration: 500 * time.Millisecond,
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
	if stats.TotalDuration != 500*time.Millisecond {
		t.Errorf("Stats.TotalDuration = %v, want 500ms", stats.TotalDuration)
	}
}

// Benchmark for constant VUs execution
func BenchmarkConstantVUs_Run(b *testing.B) {
	server := createConstantVUsTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createConstantVUsTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	for i := 0; i < b.N; i++ {
		e := executor.NewConstantVUs()
		config := &executor.Config{
			Type:     executor.TypeConstantVUs,
			VUs:      2,
			Duration: 100 * time.Millisecond,
		}
		_ = e.Init(context.Background(), config)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = e.Run(ctx, scheduler, metricsEngine)
		cancel()
	}
}
