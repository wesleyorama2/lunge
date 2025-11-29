package v2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	v2 "github.com/wesleyorama2/lunge/internal/performance/v2"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// Helper function to create a test scenario
func createTestScenario(serverURL string) *v2.Scenario {
	return &v2.Scenario{
		Name: "test-scenario",
		Variables: map[string]string{
			"baseURL": serverURL,
		},
		Requests: []*v2.RequestConfig{
			{
				Name:   "test-request",
				Method: "GET",
				URL:    serverURL,
			},
		},
	}
}

// Helper function to create a test VU
func createTestVU(scenario *v2.Scenario, metricsEngine *metrics.Engine) *v2.VirtualUser {
	client := &http.Client{Timeout: 5 * time.Second}
	return v2.NewVirtualUser(1, scenario, client, metricsEngine)
}

func TestNewVirtualUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createTestScenario(server.URL)
	vu := createTestVU(scenario, metricsEngine)

	// Verify fields are set correctly
	if vu.ID != 1 {
		t.Errorf("VU ID = %d, want 1", vu.ID)
	}
	if vu.Scenario == nil {
		t.Error("VU Scenario is nil")
	}
	if vu.HTTPClient == nil {
		t.Error("VU HTTPClient is nil")
	}
	if vu.Metrics == nil {
		t.Error("VU Metrics is nil")
	}
	if vu.GetState() != v2.VUStateIdle {
		t.Errorf("Initial VU state = %v, want %v", vu.GetState(), v2.VUStateIdle)
	}
	if vu.GetIteration() != 0 {
		t.Errorf("Initial iteration = %d, want 0", vu.GetIteration())
	}
}

func TestVUState_String(t *testing.T) {
	tests := []struct {
		state v2.VUState
		want  string
	}{
		{v2.VUStateIdle, "idle"},
		{v2.VUStateRunning, "running"},
		{v2.VUStateStopping, "stopping"},
		{v2.VUStateStopped, "stopped"},
		{v2.VUState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("VUState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVirtualUser_StateTransitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createTestScenario(server.URL)
	vu := createTestVU(scenario, metricsEngine)

	// Initial state is Idle
	if vu.GetState() != v2.VUStateIdle {
		t.Errorf("Initial state = %v, want %v", vu.GetState(), v2.VUStateIdle)
	}

	// Run an iteration (transitions to Running then back to Idle)
	ctx := context.Background()
	err := vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("RunIteration() error = %v", err)
	}

	// After iteration completes, state should be Idle
	if vu.GetState() != v2.VUStateIdle {
		t.Errorf("After iteration state = %v, want %v", vu.GetState(), v2.VUStateIdle)
	}

	// Request stop
	vu.RequestStop()
	if vu.GetState() != v2.VUStateStopping {
		t.Errorf("After RequestStop state = %v, want %v", vu.GetState(), v2.VUStateStopping)
	}

	// Mark stopped
	vu.MarkStopped()
	if vu.GetState() != v2.VUStateStopped {
		t.Errorf("After MarkStopped state = %v, want %v", vu.GetState(), v2.VUStateStopped)
	}
}

func TestVirtualUser_RunIteration(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 123, "name": "test"}`))
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "test-scenario",
		Requests: []*v2.RequestConfig{
			{
				Name:   "request-1",
				Method: "GET",
				URL:    server.URL + "/first",
			},
			{
				Name:   "request-2",
				Method: "GET",
				URL:    server.URL + "/second",
			},
		},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	// Run one iteration
	err := vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("RunIteration() error = %v", err)
	}

	// Verify both requests were made
	if requestCount != 2 {
		t.Errorf("Request count = %d, want 2", requestCount)
	}

	// Verify iteration counter incremented
	if vu.GetIteration() != 1 {
		t.Errorf("Iteration count = %d, want 1", vu.GetIteration())
	}

	// Run another iteration
	err = vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("Second RunIteration() error = %v", err)
	}

	if vu.GetIteration() != 2 {
		t.Errorf("Iteration count after second iteration = %d, want 2", vu.GetIteration())
	}
}

func TestVirtualUser_RunIteration_WithThinkTime(t *testing.T) {
	requestTimes := make([]time.Time, 0)
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	thinkTime := 100 * time.Millisecond
	scenario := &v2.Scenario{
		Name: "test-scenario",
		Requests: []*v2.RequestConfig{
			{
				Name:      "request-1",
				Method:    "GET",
				URL:       server.URL + "/first",
				ThinkTime: thinkTime,
			},
			{
				Name:   "request-2",
				Method: "GET",
				URL:    server.URL + "/second",
			},
		},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	startTime := time.Now()
	err := vu.RunIteration(ctx)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("RunIteration() error = %v", err)
	}

	// Should have taken at least the think time
	if elapsed < thinkTime {
		t.Errorf("Elapsed time = %v, want >= %v (think time)", elapsed, thinkTime)
	}

	mu.Lock()
	reqTimesLen := len(requestTimes)
	mu.Unlock()

	// Verify both requests were made
	if reqTimesLen != 2 {
		t.Errorf("Request count = %d, want 2", reqTimesLen)
	}

	// Verify think time was applied between requests
	mu.Lock()
	if len(requestTimes) >= 2 {
		timeBetweenRequests := requestTimes[1].Sub(requestTimes[0])
		if timeBetweenRequests < thinkTime-10*time.Millisecond {
			t.Errorf("Time between requests = %v, want >= %v", timeBetweenRequests, thinkTime)
		}
	}
	mu.Unlock()
}

func TestVirtualUser_RunIteration_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createTestScenario(server.URL)
	vu := createTestVU(scenario, metricsEngine)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := vu.RunIteration(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestVirtualUser_RunIteration_StoppedVU(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createTestScenario(server.URL)
	vu := createTestVU(scenario, metricsEngine)

	// Stop the VU
	vu.RequestStop()
	vu.MarkStopped()

	ctx := context.Background()
	err := vu.RunIteration(ctx)
	if err == nil {
		t.Error("Expected error when VU is stopped")
	}
}

func TestVirtualUser_Variables(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name:      "test",
		Variables: map[string]string{},
	}
	vu := createTestVU(scenario, metricsEngine)

	// Test SetData and GetData
	vu.SetData("key1", "value1")
	vu.SetData("key2", 123)
	vu.SetData("key3", true)

	val1, ok := vu.GetData("key1")
	if !ok || val1 != "value1" {
		t.Errorf("GetData(key1) = %v, %v, want value1, true", val1, ok)
	}

	val2, ok := vu.GetData("key2")
	if !ok || val2 != 123 {
		t.Errorf("GetData(key2) = %v, %v, want 123, true", val2, ok)
	}

	val3, ok := vu.GetData("key3")
	if !ok || val3 != true {
		t.Errorf("GetData(key3) = %v, %v, want true, true", val3, ok)
	}

	// Test non-existent key
	_, ok = vu.GetData("nonexistent")
	if ok {
		t.Error("GetData(nonexistent) should return false")
	}

	// Test ClearData
	vu.ClearData("key1")
	_, ok = vu.GetData("key1")
	if ok {
		t.Error("After ClearData, key1 should not exist")
	}
}

func TestVirtualUser_RequestStop(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{Name: "test"}
	vu := createTestVU(scenario, metricsEngine)

	// Initial state is idle
	if vu.GetState() != v2.VUStateIdle {
		t.Errorf("Initial state = %v, want %v", vu.GetState(), v2.VUStateIdle)
	}

	// Request stop
	vu.RequestStop()
	if vu.GetState() != v2.VUStateStopping {
		t.Errorf("After RequestStop state = %v, want %v", vu.GetState(), v2.VUStateStopping)
	}

	// Calling RequestStop again should be safe
	vu.RequestStop()
	if vu.GetState() != v2.VUStateStopping {
		t.Errorf("After second RequestStop state = %v, want %v", vu.GetState(), v2.VUStateStopping)
	}
}

func TestVirtualUser_MarkStopped(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{Name: "test"}
	vu := createTestVU(scenario, metricsEngine)

	// Mark stopped directly
	vu.MarkStopped()
	if vu.GetState() != v2.VUStateStopped {
		t.Errorf("After MarkStopped state = %v, want %v", vu.GetState(), v2.VUStateStopped)
	}

	// Calling MarkStopped again should be safe
	vu.MarkStopped()
	if vu.GetState() != v2.VUStateStopped {
		t.Errorf("After second MarkStopped state = %v, want %v", vu.GetState(), v2.VUStateStopped)
	}
}

func TestVirtualUser_WaitForStop(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{Name: "test"}
	vu := createTestVU(scenario, metricsEngine)

	// WaitForStop should timeout when VU is not stopped
	stopped := vu.WaitForStop(50 * time.Millisecond)
	if stopped {
		t.Error("WaitForStop should return false when VU is not stopped")
	}

	// Mark VU as stopped in a goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		vu.MarkStopped()
	}()

	// Now WaitForStop should return true
	stopped = vu.WaitForStop(100 * time.Millisecond)
	if !stopped {
		t.Error("WaitForStop should return true when VU is stopped")
	}
}

func TestVirtualUser_ExtractVariables(t *testing.T) {
	// Test extracting from header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-12345")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 123, "name": "test"}`))
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "test-extract",
		Requests: []*v2.RequestConfig{
			{
				Name:   "extract-test",
				Method: "GET",
				URL:    server.URL,
				Extract: []v2.ExtractConfig{
					{
						Name:   "requestId",
						Source: "header",
						Path:   "X-Request-Id",
					},
					{
						Name:   "statusCode",
						Source: "status",
					},
					{
						Name:   "responseBody",
						Source: "body",
					},
				},
			},
		},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	err := vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("RunIteration() error = %v", err)
	}

	// Check extracted header value
	requestId, ok := vu.GetData("requestId")
	if !ok || requestId != "req-12345" {
		t.Errorf("Extracted requestId = %v, %v, want req-12345, true", requestId, ok)
	}

	// Check extracted status
	statusCode, ok := vu.GetData("statusCode")
	if !ok || statusCode != "200" {
		t.Errorf("Extracted statusCode = %v, %v, want 200, true", statusCode, ok)
	}

	// Check extracted body
	body, ok := vu.GetData("responseBody")
	if !ok {
		t.Error("responseBody not extracted")
	}
	if body == "" {
		t.Error("responseBody should not be empty")
	}
}

func TestVirtualUser_HTTPRequestWithHeaders(t *testing.T) {
	var receivedHeaders http.Header
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "test-headers",
		Variables: map[string]string{
			"authToken": "Bearer test-token-123",
		},
		Requests: []*v2.RequestConfig{
			{
				Name:   "header-test",
				Method: "GET",
				URL:    server.URL,
				Headers: map[string]string{
					"Authorization": "{{authToken}}",
					"Content-Type":  "application/json",
					"X-Custom":      "custom-value",
				},
			},
		},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	err := vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("RunIteration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify headers were sent
	if receivedHeaders.Get("Authorization") != "Bearer test-token-123" {
		t.Errorf("Authorization header = %q, want %q", receivedHeaders.Get("Authorization"), "Bearer test-token-123")
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header = %q, want %q", receivedHeaders.Get("Content-Type"), "application/json")
	}
	if receivedHeaders.Get("X-Custom") != "custom-value" {
		t.Errorf("X-Custom header = %q, want %q", receivedHeaders.Get("X-Custom"), "custom-value")
	}
}

func TestVirtualUser_HTTPRequestWithBody(t *testing.T) {
	var receivedBody string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		mu.Lock()
		receivedBody = string(body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "test-body",
		Variables: map[string]string{
			"userName": "testuser",
		},
		Requests: []*v2.RequestConfig{
			{
				Name:   "body-test",
				Method: "POST",
				URL:    server.URL,
				Body:   `{"username": "{{userName}}", "active": true}`,
			},
		},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	err := vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("RunIteration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	expectedBody := `{"username": "testuser", "active": true}`
	if receivedBody != expectedBody {
		t.Errorf("Request body = %q, want %q", receivedBody, expectedBody)
	}
}

func TestVirtualUser_HTTPRequestFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := createTestScenario(server.URL)
	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	err := vu.RunIteration(ctx)
	// Iteration should complete even with failed requests (500s are still counted)
	if err != nil {
		t.Errorf("RunIteration() error = %v (should complete even on 500)", err)
	}

	// Check metrics recorded the failure
	snapshot := metricsEngine.GetSnapshot()
	if snapshot.FailedRequests != 1 {
		t.Errorf("FailedRequests = %d, want 1", snapshot.FailedRequests)
	}
}

func TestVirtualUser_ConcurrentAccess(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{Name: "test"}
	vu := createTestVU(scenario, metricsEngine)

	// Concurrent SetData/GetData access
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			vu.SetData("key", idx)
		}(i)
		go func() {
			defer wg.Done()
			vu.GetData("key")
		}()
	}
	wg.Wait()

	// No race conditions should occur
}

func TestVirtualUser_StopDuringIteration(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name: "test-stop",
		Requests: []*v2.RequestConfig{
			{Name: "req1", Method: "GET", URL: server.URL},
			{Name: "req2", Method: "GET", URL: server.URL},
			{Name: "req3", Method: "GET", URL: server.URL},
		},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	// Run iteration in goroutine
	go func() {
		time.Sleep(25 * time.Millisecond) // Stop during first request
		vu.RequestStop()
	}()

	err := vu.RunIteration(ctx)
	// Should complete gracefully (either erroring or returning nil for graceful stop)
	_ = err

	// Verify VU is stopping or stopped
	state := vu.GetState()
	if state != v2.VUStateStopping && state != v2.VUStateStopped && state != v2.VUStateIdle {
		t.Errorf("State after stop = %v", state)
	}
}

func TestRequestResult_Fields(t *testing.T) {
	result := &v2.RequestResult{
		VUID:          1,
		Iteration:     5,
		RequestName:   "test-request",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(100 * time.Millisecond),
		Duration:      100 * time.Millisecond,
		StatusCode:    200,
		BytesReceived: 1024,
		Error:         nil,
		ResponseBody:  []byte(`{"status": "ok"}`),
	}

	if result.VUID != 1 {
		t.Errorf("VUID = %d, want 1", result.VUID)
	}
	if result.Iteration != 5 {
		t.Errorf("Iteration = %d, want 5", result.Iteration)
	}
	if result.RequestName != "test-request" {
		t.Errorf("RequestName = %s, want test-request", result.RequestName)
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", result.Duration)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if result.BytesReceived != 1024 {
		t.Errorf("BytesReceived = %d, want 1024", result.BytesReceived)
	}
}

func TestScenario_Fields(t *testing.T) {
	scenario := &v2.Scenario{
		Name: "test-scenario",
		Variables: map[string]string{
			"var1": "value1",
			"var2": "value2",
		},
		Requests: []*v2.RequestConfig{
			{
				Name:   "request-1",
				Method: "GET",
				URL:    "https://api.example.com/users",
			},
		},
	}

	if scenario.Name != "test-scenario" {
		t.Errorf("Name = %s, want test-scenario", scenario.Name)
	}
	if len(scenario.Variables) != 2 {
		t.Errorf("Variables length = %d, want 2", len(scenario.Variables))
	}
	if len(scenario.Requests) != 1 {
		t.Errorf("Requests length = %d, want 1", len(scenario.Requests))
	}
}

func TestRequestConfig_Fields(t *testing.T) {
	config := &v2.RequestConfig{
		Name:      "test-request",
		Method:    "POST",
		URL:       "https://api.example.com/users",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      `{"name": "test"}`,
		Timeout:   5 * time.Second,
		ThinkTime: 1 * time.Second,
		Extract: []v2.ExtractConfig{
			{Name: "userId", Source: "body", Path: "$.id"},
		},
	}

	if config.Name != "test-request" {
		t.Errorf("Name = %s, want test-request", config.Name)
	}
	if config.Method != "POST" {
		t.Errorf("Method = %s, want POST", config.Method)
	}
	if config.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", config.Timeout)
	}
	if config.ThinkTime != 1*time.Second {
		t.Errorf("ThinkTime = %v, want 1s", config.ThinkTime)
	}
	if len(config.Extract) != 1 {
		t.Errorf("Extract length = %d, want 1", len(config.Extract))
	}
}

func TestExtractConfig_Fields(t *testing.T) {
	config := v2.ExtractConfig{
		Name:   "userId",
		Source: "body",
		Path:   "$.id",
		Regex:  `"id":\s*(\d+)`,
	}

	if config.Name != "userId" {
		t.Errorf("Name = %s, want userId", config.Name)
	}
	if config.Source != "body" {
		t.Errorf("Source = %s, want body", config.Source)
	}
	if config.Path != "$.id" {
		t.Errorf("Path = %s, want $.id", config.Path)
	}
	if config.Regex != `"id":\s*(\d+)` {
		t.Errorf("Regex = %s, want \"id\":\\s*(\\d+)", config.Regex)
	}
}

func TestVirtualUser_NilScenario(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	client := &http.Client{Timeout: 5 * time.Second}
	vu := v2.NewVirtualUser(1, nil, client, metricsEngine)

	// VU with nil scenario should have zero requests
	if vu.Scenario != nil {
		t.Error("Expected nil scenario")
	}
}

func TestVirtualUser_EmptyScenario(t *testing.T) {
	metricsEngine := metrics.NewEngine()
	defer metricsEngine.Stop()

	scenario := &v2.Scenario{
		Name:     "empty",
		Requests: []*v2.RequestConfig{},
	}

	vu := createTestVU(scenario, metricsEngine)
	ctx := context.Background()

	// An empty scenario should complete successfully
	err := vu.RunIteration(ctx)
	if err != nil {
		t.Errorf("RunIteration() with empty scenario error = %v", err)
	}
}
