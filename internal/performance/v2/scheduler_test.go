package v2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// createTestServer creates a test HTTP server
func createTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
}

// createSchedulerTestScenario creates a scenario for scheduler testing
func createSchedulerTestScenario(serverURL string) *v2.Scenario {
	return &v2.Scenario{
		Name: "scheduler-test",
		Requests: []*v2.RequestConfig{
			{
				Name:   "test-request",
				Method: "GET",
				URL:    serverURL,
			},
		},
	}
}

func TestDefaultHTTPClientConfig(t *testing.T) {
	config := v2.DefaultHTTPClientConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}
	if config.MaxIdleConns != 1000 {
		t.Errorf("MaxIdleConns = %d, want 1000", config.MaxIdleConns)
	}
	if config.MaxIdleConnsPerHost != 100 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 100", config.MaxIdleConnsPerHost)
	}
	if config.MaxConnsPerHost != 0 {
		t.Errorf("MaxConnsPerHost = %d, want 0", config.MaxConnsPerHost)
	}
	if config.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", config.IdleConnTimeout)
	}
	if config.DisableKeepAlives {
		t.Error("DisableKeepAlives should be false by default")
	}
	if config.DisableCompression {
		t.Error("DisableCompression should be false by default")
	}
	if config.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be false by default")
	}
	if !config.UseSharedClient {
		t.Error("UseSharedClient should be true by default")
	}
}

func TestNewVUScheduler(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	if scheduler == nil {
		t.Fatal("NewVUScheduler() returned nil")
	}

	// Check initial state
	if scheduler.GetActiveVUCount() != 0 {
		t.Errorf("Initial active VU count = %d, want 0", scheduler.GetActiveVUCount())
	}

	// Check GetActiveVUs returns empty slice
	activeVUs := scheduler.GetActiveVUs()
	if len(activeVUs) != 0 {
		t.Errorf("Initial GetActiveVUs() length = %d, want 0", len(activeVUs))
	}
}

func TestNewVUScheduler_WithoutSharedClient(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	httpConfig.UseSharedClient = false

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	if scheduler == nil {
		t.Fatal("NewVUScheduler() returned nil")
	}
}

func TestVUScheduler_SpawnVU(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn first VU
	vu1 := scheduler.SpawnVU()
	if vu1 == nil {
		t.Fatal("SpawnVU() returned nil")
	}
	if vu1.ID != 1 {
		t.Errorf("First VU ID = %d, want 1", vu1.ID)
	}

	// Spawn second VU
	vu2 := scheduler.SpawnVU()
	if vu2 == nil {
		t.Fatal("Second SpawnVU() returned nil")
	}
	if vu2.ID != 2 {
		t.Errorf("Second VU ID = %d, want 2", vu2.ID)
	}

	// Check active count
	if scheduler.GetActiveVUCount() != 2 {
		t.Errorf("Active VU count = %d, want 2", scheduler.GetActiveVUCount())
	}
}

func TestVUScheduler_SpawnVU_WithoutSharedClient(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	httpConfig.UseSharedClient = false

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn VUs - each should get its own client
	vu1 := scheduler.SpawnVU()
	vu2 := scheduler.SpawnVU()

	if vu1 == nil || vu2 == nil {
		t.Fatal("SpawnVU() returned nil")
	}

	// Each VU should have a different HTTP client when UseSharedClient=false
	if vu1.HTTPClient == vu2.HTTPClient {
		t.Error("VUs should have different HTTP clients when UseSharedClient=false")
	}
}

func TestVUScheduler_GetVU(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn a VU
	vu := scheduler.SpawnVU()

	// Get VU by ID
	retrieved := scheduler.GetVU(vu.ID)
	if retrieved != vu {
		t.Error("GetVU() did not return the same VU")
	}

	// Get non-existent VU
	nonExistent := scheduler.GetVU(999)
	if nonExistent != nil {
		t.Error("GetVU(999) should return nil")
	}
}

func TestVUScheduler_GetActiveVUs(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn multiple VUs
	scheduler.SpawnVU()
	scheduler.SpawnVU()
	scheduler.SpawnVU()

	activeVUs := scheduler.GetActiveVUs()
	if len(activeVUs) != 3 {
		t.Errorf("GetActiveVUs() length = %d, want 3", len(activeVUs))
	}
}

func TestVUScheduler_GetActiveVUCount(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Initially 0
	if scheduler.GetActiveVUCount() != 0 {
		t.Errorf("Initial count = %d, want 0", scheduler.GetActiveVUCount())
	}

	// Spawn VUs
	scheduler.SpawnVU()
	if scheduler.GetActiveVUCount() != 1 {
		t.Errorf("After 1 spawn count = %d, want 1", scheduler.GetActiveVUCount())
	}

	scheduler.SpawnVU()
	if scheduler.GetActiveVUCount() != 2 {
		t.Errorf("After 2 spawns count = %d, want 2", scheduler.GetActiveVUCount())
	}

	// Stop a VU
	vu := scheduler.SpawnVU()
	vu.MarkStopped()

	if scheduler.GetActiveVUCount() != 2 {
		t.Errorf("After stopping 1 VU count = %d, want 2", scheduler.GetActiveVUCount())
	}
}

func TestVUScheduler_StopVU(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	vu := scheduler.SpawnVU()

	// Stop the VU
	scheduler.StopVU(vu.ID)

	// VU should be in stopping state
	if vu.GetState() != v2.VUStateStopping {
		t.Errorf("VU state = %v, want %v", vu.GetState(), v2.VUStateStopping)
	}

	// Stop non-existent VU should not panic
	scheduler.StopVU(999)
}

func TestVUScheduler_StopAllVUs(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn multiple VUs
	vu1 := scheduler.SpawnVU()
	vu2 := scheduler.SpawnVU()
	vu3 := scheduler.SpawnVU()

	// Stop all
	scheduler.StopAllVUs()

	// All should be stopping
	if vu1.GetState() != v2.VUStateStopping {
		t.Errorf("VU1 state = %v, want %v", vu1.GetState(), v2.VUStateStopping)
	}
	if vu2.GetState() != v2.VUStateStopping {
		t.Errorf("VU2 state = %v, want %v", vu2.GetState(), v2.VUStateStopping)
	}
	if vu3.GetState() != v2.VUStateStopping {
		t.Errorf("VU3 state = %v, want %v", vu3.GetState(), v2.VUStateStopping)
	}
}

func TestVUScheduler_RemoveVU(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	vu := scheduler.SpawnVU()
	vuID := vu.ID

	// Verify VU exists
	if scheduler.GetVU(vuID) == nil {
		t.Error("VU should exist before removal")
	}

	// Remove the VU
	scheduler.RemoveVU(vuID)

	// Verify VU is removed
	if scheduler.GetVU(vuID) != nil {
		t.Error("VU should be nil after removal")
	}

	// VU should be marked as stopped
	if vu.GetState() != v2.VUStateStopped {
		t.Errorf("VU state = %v, want %v", vu.GetState(), v2.VUStateStopped)
	}

	// Remove non-existent VU should not panic
	scheduler.RemoveVU(999)
}

func TestVUScheduler_WaitForAllVUs(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn VUs
	vu1 := scheduler.SpawnVU()
	vu2 := scheduler.SpawnVU()

	// Mark VUs as stopping and then stopped
	go func() {
		time.Sleep(10 * time.Millisecond)
		vu1.RequestStop()
		vu1.MarkStopped()
		time.Sleep(10 * time.Millisecond)
		vu2.RequestStop()
		vu2.MarkStopped()
	}()

	// Wait should succeed
	notStopped := scheduler.WaitForAllVUs(500 * time.Millisecond)
	if notStopped != 0 {
		t.Errorf("WaitForAllVUs returned %d not stopped, want 0", notStopped)
	}
}

func TestVUScheduler_WaitForAllVUs_Timeout(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn VUs but don't stop them
	scheduler.SpawnVU()
	scheduler.SpawnVU()

	// Wait should timeout
	notStopped := scheduler.WaitForAllVUs(50 * time.Millisecond)
	if notStopped != 2 {
		t.Errorf("WaitForAllVUs returned %d not stopped, want 2", notStopped)
	}
}

func TestVUScheduler_RunVU(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	vu := scheduler.SpawnVU()

	// Run VU with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.RunVU(ctx, vu, 0) // No pacing
	}()

	wg.Wait()

	// Should have made at least 1 request
	if atomic.LoadInt32(&requestCount) < 1 {
		t.Error("RunVU should have made at least 1 request")
	}

	// VU should be stopped
	if vu.GetState() != v2.VUStateStopped {
		t.Errorf("VU state = %v, want %v", vu.GetState(), v2.VUStateStopped)
	}
}

func TestVUScheduler_RunVU_WithPacing(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	vu := scheduler.SpawnVU()

	// Run VU with pacing (50ms between iterations)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.RunVU(ctx, vu, 50*time.Millisecond)
	}()

	wg.Wait()

	// With 150ms runtime and 50ms pacing, we should have roughly 3 iterations
	count := atomic.LoadInt32(&requestCount)
	if count < 1 || count > 5 {
		t.Errorf("Request count = %d, expected 1-5 with 50ms pacing over 150ms", count)
	}
}

func TestVUScheduler_RunVU_StoppedVU(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	vu := scheduler.SpawnVU()

	// Pre-stop the VU
	vu.RequestStop()

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.RunVU(ctx, vu, 0)
	}()

	// Should return quickly since VU is already stopped
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("RunVU didn't return quickly for stopped VU")
	}
}

func TestVUScheduler_ScaleVUs_ScaleUp(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	ctx := context.Background()

	// Scale up to 5 VUs
	var spawned []*v2.VirtualUser
	count := scheduler.ScaleVUs(ctx, 5, 0, func(vu *v2.VirtualUser) {
		spawned = append(spawned, vu)
	})

	if count != 5 {
		t.Errorf("ScaleVUs returned %d, want 5", count)
	}

	if len(spawned) != 5 {
		t.Errorf("Spawned %d VUs, want 5", len(spawned))
	}

	if scheduler.GetActiveVUCount() != 5 {
		t.Errorf("Active VU count = %d, want 5", scheduler.GetActiveVUCount())
	}
}

func TestVUScheduler_ScaleVUs_ScaleDown(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	ctx := context.Background()

	// First scale up to 5
	scheduler.ScaleVUs(ctx, 5, 0, nil)

	// Then scale down to 2
	scheduler.ScaleVUs(ctx, 2, 0, nil)

	// Some VUs should be stopping
	activeVUs := scheduler.GetActiveVUs()
	stoppingCount := 0
	for _, vu := range activeVUs {
		if vu.GetState() == v2.VUStateStopping {
			stoppingCount++
		}
	}

	// Should have requested stop on 3 VUs
	if stoppingCount != 3 {
		t.Errorf("Stopping VU count = %d, want 3", stoppingCount)
	}
}

func TestVUScheduler_ScaleVUs_NoChange(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	ctx := context.Background()

	// Scale to 3
	scheduler.ScaleVUs(ctx, 3, 0, nil)

	// Scale to 3 again (no change)
	spawned := 0
	scheduler.ScaleVUs(ctx, 3, 0, func(vu *v2.VirtualUser) {
		spawned++
	})

	if spawned != 0 {
		t.Errorf("Spawned %d new VUs when scaling to same count", spawned)
	}
}

func TestVUScheduler_ScaleVUs_WithCallback(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	ctx := context.Background()

	// Track spawned VUs
	var spawnedIDs []int
	var mu sync.Mutex

	scheduler.ScaleVUs(ctx, 3, 0, func(vu *v2.VirtualUser) {
		mu.Lock()
		spawnedIDs = append(spawnedIDs, vu.ID)
		mu.Unlock()
	})

	mu.Lock()
	defer mu.Unlock()

	if len(spawnedIDs) != 3 {
		t.Errorf("Callback called %d times, want 3", len(spawnedIDs))
	}
}

func TestVUScheduler_Shutdown(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn some VUs
	vu1 := scheduler.SpawnVU()
	vu2 := scheduler.SpawnVU()

	// Start running VUs in background
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scheduler.RunVU(ctx, vu1, 0)
	}()
	go func() {
		defer wg.Done()
		scheduler.RunVU(ctx, vu2, 0)
	}()

	// Wait a bit then shutdown
	time.Sleep(50 * time.Millisecond)
	scheduler.Shutdown(1 * time.Second)

	// Wait for goroutines
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(2 * time.Second):
		t.Error("Shutdown did not complete in time")
	}
}

func TestVUScheduler_Shutdown_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn and run a VU
	vu := scheduler.SpawnVU()

	ctx := context.Background()
	go scheduler.RunVU(ctx, vu, 0)

	// Give VU time to start request
	time.Sleep(50 * time.Millisecond)

	// Shutdown with very short timeout
	start := time.Now()
	scheduler.Shutdown(100 * time.Millisecond)
	elapsed := time.Since(start)

	// Should have returned after timeout
	if elapsed > 200*time.Millisecond {
		t.Errorf("Shutdown took %v, expected ~100ms timeout", elapsed)
	}
}

func TestVUScheduler_UpdateMetrics(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn VUs
	scheduler.SpawnVU()
	scheduler.SpawnVU()
	scheduler.SpawnVU()

	// Update metrics
	scheduler.UpdateMetrics()

	// Check metrics engine has correct VU count
	if metricsEngine.GetActiveVUs() != 3 {
		t.Errorf("Metrics engine active VUs = %d, want 3", metricsEngine.GetActiveVUs())
	}
}

func TestVUScheduler_ConcurrentSpawn(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn VUs concurrently
	var wg sync.WaitGroup
	vuCount := 50

	for i := 0; i < vuCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scheduler.SpawnVU()
		}()
	}
	wg.Wait()

	if scheduler.GetActiveVUCount() != vuCount {
		t.Errorf("Active VU count = %d, want %d", scheduler.GetActiveVUCount(), vuCount)
	}
}

func TestVUScheduler_ConcurrentGetVU(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn some VUs
	vus := make([]*v2.VirtualUser, 10)
	for i := 0; i < 10; i++ {
		vus[i] = scheduler.SpawnVU()
	}

	// Concurrently get VUs
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			vuID := vus[idx%10].ID
			retrieved := scheduler.GetVU(vuID)
			if retrieved == nil {
				t.Errorf("GetVU(%d) returned nil", vuID)
			}
		}(i)
	}
	wg.Wait()
}

func TestHTTPClientConfig_Fields(t *testing.T) {
	config := v2.HTTPClientConfig{
		Timeout:             10 * time.Second,
		MaxIdleConns:        500,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     60 * time.Second,
		DisableKeepAlives:   true,
		DisableCompression:  true,
		InsecureSkipVerify:  true,
		UseSharedClient:     false,
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", config.Timeout)
	}
	if config.MaxIdleConns != 500 {
		t.Errorf("MaxIdleConns = %d, want 500", config.MaxIdleConns)
	}
	if config.MaxIdleConnsPerHost != 50 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 50", config.MaxIdleConnsPerHost)
	}
	if config.MaxConnsPerHost != 100 {
		t.Errorf("MaxConnsPerHost = %d, want 100", config.MaxConnsPerHost)
	}
	if config.IdleConnTimeout != 60*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 60s", config.IdleConnTimeout)
	}
	if !config.DisableKeepAlives {
		t.Error("DisableKeepAlives should be true")
	}
	if !config.DisableCompression {
		t.Error("DisableCompression should be true")
	}
	if !config.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
	if config.UseSharedClient {
		t.Error("UseSharedClient should be false")
	}
}

func TestVUScheduler_SharedClient(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createSchedulerTestScenario(server.URL)
	httpConfig := v2.DefaultHTTPClientConfig()
	httpConfig.UseSharedClient = true

	scheduler := v2.NewVUScheduler(scenario, metricsEngine, httpConfig)

	// Spawn multiple VUs
	vu1 := scheduler.SpawnVU()
	vu2 := scheduler.SpawnVU()

	// Both should share the same HTTP client
	if vu1.HTTPClient != vu2.HTTPClient {
		t.Error("VUs should share the same HTTP client when UseSharedClient=true")
	}
}
