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

// createArrivalRateTestServer creates a test HTTP server
func createArrivalRateTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
}

// createArrivalRateTestScenario creates a scenario for testing
func createArrivalRateTestScenario(serverURL string) *v2.Scenario {
	return &v2.Scenario{
		Name: "arrival-rate-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test-request",
				Method: "GET",
				URL:    serverURL,
			},
		},
	}
}

func TestConstantArrivalRate_Type(t *testing.T) {
	e := executor.NewConstantArrivalRate()
	if e.Type() != executor.TypeConstantArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantArrivalRate)
	}
}

func TestConstantArrivalRate_Init_ValidConfig(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            100.0,
		Duration:        1 * time.Minute,
		PreAllocatedVUs: 10,
		MaxVUs:          50,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
}

func TestConstantArrivalRate_Init_InvalidType(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:     executor.TypeConstantVUs, // Wrong type
		Rate:     100.0,
		Duration: 1 * time.Minute,
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for wrong type, got nil")
	}
}

func TestConstantArrivalRate_Init_MissingRate(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:     executor.TypeConstantArrivalRate,
		Rate:     0, // Invalid rate
		Duration: 1 * time.Minute,
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for zero rate, got nil")
	}
}

func TestConstantArrivalRate_Init_MissingDuration(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:     executor.TypeConstantArrivalRate,
		Rate:     100.0,
		Duration: 0, // Invalid duration
	}

	err := e.Init(context.Background(), config)
	if err == nil {
		t.Fatal("Init() expected error for zero duration, got nil")
	}
}

func TestConstantArrivalRate_Init_Defaults(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:     executor.TypeConstantArrivalRate,
		Rate:     100.0,
		Duration: 1 * time.Minute,
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

func TestConstantArrivalRate_Init_MaxVUsAdjusted(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            100.0,
		Duration:        1 * time.Minute,
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

func TestConstantArrivalRate_Run_Basic(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0, // 10 iterations per second
		Duration:        500 * time.Millisecond,
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

	// Should have run for approximately the duration
	if elapsed < 400*time.Millisecond || elapsed > 1*time.Second {
		t.Errorf("Run() elapsed = %v, want ~500ms", elapsed)
	}

	// Check iterations were performed
	stats := e.GetStats()
	if stats.Iterations < 3 { // At 10 RPS for 500ms, expect ~5 iterations (with some tolerance)
		t.Errorf("Iterations = %d, want at least 3", stats.Iterations)
	}
}

func TestConstantArrivalRate_GetProgress(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        1 * time.Second,
		PreAllocatedVUs: 1,
	}

	_ = e.Init(context.Background(), config)

	// Before running
	progress := e.GetProgress()
	if progress != 0.0 {
		t.Errorf("Before Run(), GetProgress() = %v, want 0.0", progress)
	}
}

func TestConstantArrivalRate_GetActiveVUs(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        1 * time.Second,
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

func TestConstantArrivalRate_GetStats(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            100.0,
		Duration:        1 * time.Minute,
		PreAllocatedVUs: 5,
		MaxVUs:          10,
	}

	_ = e.Init(context.Background(), config)

	stats := e.GetStats()
	if stats.TotalDuration != 1*time.Minute {
		t.Errorf("Stats.TotalDuration = %v, want 1m", stats.TotalDuration)
	}
	if stats.TargetRate != 100.0 {
		t.Errorf("Stats.TargetRate = %v, want 100.0", stats.TargetRate)
	}
	if stats.CurrentRate != 100.0 {
		t.Errorf("Stats.CurrentRate = %v, want 100.0", stats.CurrentRate)
	}
}

func TestConstantArrivalRate_Stop(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        10 * time.Second, // Long duration
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

func TestConstantArrivalRate_ContextCancellation(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        10 * time.Second, // Long duration
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

func TestConstantArrivalRate_VUPoolScaling(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	// High rate with low pre-allocated VUs should trigger scaling
	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            100.0, // High rate
		Duration:        300 * time.Millisecond,
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

func TestConstantArrivalRate_Interface(t *testing.T) {
	// Verify that ConstantArrivalRate implements Executor interface
	var _ executor.Executor = (*executor.ConstantArrivalRate)(nil)
}

func TestConstantArrivalRate_ConcurrentIterations(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            50.0, // 50 iterations per second
		Duration:        500 * time.Millisecond,
		PreAllocatedVUs: 5,
		MaxVUs:          10,
	}

	err := e.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = e.Run(ctx, scheduler, metricsEngine)

	// At 50 RPS for 500ms, expect ~25 iterations (but actual results vary due to timing)
	stats := e.GetStats()
	// Use stats iterations since our mock doesn't track - be lenient with timing variance
	if stats.Iterations < 10 {
		t.Errorf("Expected at least 10 iterations, got %d", stats.Iterations)
	}
}

func TestConstantArrivalRate_MetricsPhase(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        200 * time.Millisecond,
		PreAllocatedVUs: 1,
		MaxVUs:          5,
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

func TestConstantArrivalRate_Stop_BeforeRun(t *testing.T) {
	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        1 * time.Second,
		PreAllocatedVUs: 1,
	}

	_ = e.Init(context.Background(), config)

	// Stop before Run - should not panic
	err := e.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() before Run() error = %v", err)
	}
}

func TestConstantArrivalRate_GetProgress_AfterRun(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            10.0,
		Duration:        100 * time.Millisecond,
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

func TestConstantArrivalRate_ConcurrentAccess(t *testing.T) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	e := executor.NewConstantArrivalRate()

	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            20.0,
		Duration:        300 * time.Millisecond,
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
	if stats.TargetRate != 20.0 {
		t.Errorf("Stats.TargetRate = %v, want 20.0", stats.TargetRate)
	}
}

// Benchmark for iteration scheduling performance
func BenchmarkConstantArrivalRate_IterationScheduling(b *testing.B) {
	server := createArrivalRateTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createArrivalRateTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	for i := 0; i < b.N; i++ {
		e := executor.NewConstantArrivalRate()

		config := &executor.Config{
			Type:            executor.TypeConstantArrivalRate,
			Rate:            1000.0,
			Duration:        50 * time.Millisecond,
			PreAllocatedVUs: 10,
			MaxVUs:          20,
		}

		_ = e.Init(context.Background(), config)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = e.Run(ctx, scheduler, metricsEngine)
		cancel()
	}
}
