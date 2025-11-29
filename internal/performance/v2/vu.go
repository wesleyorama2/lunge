// Package v2 provides the next-generation performance testing engine.
package v2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// VUState represents the lifecycle state of a Virtual User.
type VUState int32

const (
	// VUStateIdle indicates the VU is ready but not currently running.
	VUStateIdle VUState = iota
	// VUStateRunning indicates the VU is actively running iterations.
	VUStateRunning
	// VUStateStopping indicates the VU has been requested to stop.
	VUStateStopping
	// VUStateStopped indicates the VU has fully stopped.
	VUStateStopped
)

func (s VUState) String() string {
	switch s {
	case VUStateIdle:
		return "idle"
	case VUStateRunning:
		return "running"
	case VUStateStopping:
		return "stopping"
	case VUStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// VirtualUser represents a single simulated user executing test iterations.
//
// Each VU has its own:
// - HTTP client (for connection pooling within the VU)
// - Variable scope (for extracted values and state)
// - Iteration counter
// - Lifecycle management
//
// VUs are created by the VUScheduler and run iterations defined by a Scenario.
type VirtualUser struct {
	// Unique identifier for this VU
	ID int

	// Scenario defines what requests to execute
	Scenario *Scenario

	// HTTP client for this VU (may be shared or per-VU)
	HTTPClient *http.Client

	// Metrics engine for recording results
	Metrics *metrics.Engine

	// Lifecycle state (atomic for lock-free reads)
	state atomic.Int32

	// Stop signal
	stopCh chan struct{}

	// Done signal (closed when VU fully stops)
	doneCh chan struct{}

	// Iteration counter
	iteration atomic.Int64

	// Per-VU variable scope
	data   map[string]interface{}
	dataMu sync.RWMutex

	// Last iteration timing
	lastIterStart time.Time
	lastIterEnd   time.Time
}

// NewVirtualUser creates a new Virtual User.
//
// Parameters:
//   - id: Unique identifier for this VU
//   - scenario: The scenario defining what requests to execute
//   - httpClient: HTTP client for making requests
//   - metricsEngine: Metrics engine for recording results
func NewVirtualUser(id int, scenario *Scenario, httpClient *http.Client, metricsEngine *metrics.Engine) *VirtualUser {
	return &VirtualUser{
		ID:         id,
		Scenario:   scenario,
		HTTPClient: httpClient,
		Metrics:    metricsEngine,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
		data:       make(map[string]interface{}),
	}
}

// GetState returns the current VU state.
func (vu *VirtualUser) GetState() VUState {
	return VUState(vu.state.Load())
}

// GetIteration returns the current iteration number.
func (vu *VirtualUser) GetIteration() int64 {
	return vu.iteration.Load()
}

// RunIteration executes a single iteration of the scenario.
//
// An iteration consists of executing all requests defined in the scenario,
// optionally with think time between requests.
//
// Returns:
//   - nil if the iteration completed successfully
//   - error if the iteration was cancelled or encountered a fatal error
func (vu *VirtualUser) RunIteration(ctx context.Context) error {
	// Check if we should run
	currentState := vu.GetState()
	if currentState == VUStateStopping || currentState == VUStateStopped {
		return fmt.Errorf("VU %d is stopping or stopped", vu.ID)
	}

	// Transition to running
	vu.state.Store(int32(VUStateRunning))
	vu.lastIterStart = time.Now()
	vu.iteration.Add(1)

	// Execute all requests in the scenario
	for i, req := range vu.Scenario.Requests {
		// Check for stop signal
		select {
		case <-ctx.Done():
			vu.lastIterEnd = time.Now()
			return ctx.Err()
		case <-vu.stopCh:
			vu.lastIterEnd = time.Now()
			return nil // Graceful stop
		default:
		}

		// Execute the request
		result := vu.executeRequest(ctx, req)

		// Record metrics
		success := result.Error == nil && result.StatusCode < 400
		vu.Metrics.RecordLatency(result.Duration, req.Name, success, result.BytesReceived)

		// Apply think time between requests (not after the last one)
		if req.ThinkTime > 0 && i < len(vu.Scenario.Requests)-1 {
			vu.applyThinkTime(ctx, req.ThinkTime)
		}
	}

	vu.lastIterEnd = time.Now()
	vu.state.Store(int32(VUStateIdle))
	return nil
}

// executeRequest executes a single HTTP request and returns the result.
func (vu *VirtualUser) executeRequest(ctx context.Context, req *RequestConfig) *RequestResult {
	startTime := time.Now()

	result := &RequestResult{
		VUID:        vu.ID,
		Iteration:   vu.iteration.Load(),
		RequestName: req.Name,
		StartTime:   startTime,
	}

	// Build the HTTP request
	httpReq, err := vu.buildRequest(ctx, req)
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		result.Error = fmt.Errorf("failed to build request: %w", err)
		return result
	}

	// Execute the request
	resp, err := vu.HTTPClient.Do(httpReq)
	endTime := time.Now()

	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime)

	if err != nil {
		result.Error = err
		return result
	}

	defer resp.Body.Close()

	// Read response body for byte counting and assertions
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response body: %w", err)
		result.StatusCode = resp.StatusCode
		return result
	}

	result.StatusCode = resp.StatusCode
	result.BytesReceived = int64(len(body))
	result.ResponseBody = body

	// Extract variables if configured
	if len(req.Extract) > 0 {
		vu.extractVariables(req.Extract, resp, body)
	}

	return result
}

// buildRequest builds an HTTP request from the configuration.
func (vu *VirtualUser) buildRequest(ctx context.Context, req *RequestConfig) (*http.Request, error) {
	// Resolve variables in URL
	url := vu.resolveVariables(req.URL)

	// Build request body
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(vu.resolveVariables(req.Body))
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, body)
	if err != nil {
		return nil, err
	}

	// Add headers with variable resolution
	for key, value := range req.Headers {
		httpReq.Header.Set(key, vu.resolveVariables(value))
	}

	return httpReq, nil
}

// resolveVariables replaces {{varName}} placeholders with values.
func (vu *VirtualUser) resolveVariables(input string) string {
	result := input

	// First, resolve from VU-local data
	vu.dataMu.RLock()
	for key, value := range vu.data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	vu.dataMu.RUnlock()

	// Then, resolve from scenario variables
	if vu.Scenario != nil {
		for key, value := range vu.Scenario.Variables {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	return result
}

// extractVariables extracts values from the response and stores them in VU data.
func (vu *VirtualUser) extractVariables(extracts []ExtractConfig, resp *http.Response, body []byte) {
	for _, extract := range extracts {
		var value string

		switch extract.Source {
		case "header":
			value = resp.Header.Get(extract.Path)
		case "status":
			value = fmt.Sprintf("%d", resp.StatusCode)
		case "body":
			// Simple JSONPath-like extraction (basic implementation)
			// TODO: Implement proper JSONPath support
			value = string(body)
		}

		if value != "" {
			vu.SetData(extract.Name, value)
		}
	}
}

// applyThinkTime waits for the specified duration or until stopped.
func (vu *VirtualUser) applyThinkTime(ctx context.Context, duration time.Duration) {
	select {
	case <-ctx.Done():
	case <-vu.stopCh:
	case <-time.After(duration):
	}
}

// RequestStop signals the VU to stop after completing the current iteration.
func (vu *VirtualUser) RequestStop() {
	currentState := VUState(vu.state.Load())
	if currentState == VUStateStopped {
		return
	}

	// Try to transition to stopping state
	if vu.state.CompareAndSwap(int32(VUStateRunning), int32(VUStateStopping)) ||
		vu.state.CompareAndSwap(int32(VUStateIdle), int32(VUStateStopping)) {
		close(vu.stopCh)
	}
}

// WaitForStop waits for the VU to stop with a timeout.
//
// Returns true if the VU stopped within the timeout, false otherwise.
func (vu *VirtualUser) WaitForStop(timeout time.Duration) bool {
	select {
	case <-vu.doneCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

// MarkStopped marks the VU as fully stopped.
// Should be called by the scheduler when the VU goroutine exits.
func (vu *VirtualUser) MarkStopped() {
	vu.state.Store(int32(VUStateStopped))
	select {
	case <-vu.doneCh:
		// Already closed
	default:
		close(vu.doneCh)
	}
}

// SetData stores a value in the VU's variable scope.
func (vu *VirtualUser) SetData(key string, value interface{}) {
	vu.dataMu.Lock()
	defer vu.dataMu.Unlock()
	vu.data[key] = value
}

// GetData retrieves a value from the VU's variable scope.
func (vu *VirtualUser) GetData(key string) (interface{}, bool) {
	vu.dataMu.RLock()
	defer vu.dataMu.RUnlock()
	val, ok := vu.data[key]
	return val, ok
}

// ClearData removes a value from the VU's variable scope.
func (vu *VirtualUser) ClearData(key string) {
	vu.dataMu.Lock()
	defer vu.dataMu.Unlock()
	delete(vu.data, key)
}

// RequestResult contains the result of a single HTTP request.
type RequestResult struct {
	VUID          int           `json:"vuId"`
	Iteration     int64         `json:"iteration"`
	RequestName   string        `json:"requestName"`
	StartTime     time.Time     `json:"startTime"`
	EndTime       time.Time     `json:"endTime"`
	Duration      time.Duration `json:"duration"`
	StatusCode    int           `json:"statusCode"`
	BytesReceived int64         `json:"bytesReceived"`
	Error         error         `json:"error,omitempty"`
	ResponseBody  []byte        `json:"-"` // Not serialized
}

// Scenario defines what a VU executes during each iteration.
type Scenario struct {
	// Name of the scenario
	Name string `json:"name" yaml:"name"`

	// Variables available to all requests
	Variables map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`

	// Requests to execute in order
	Requests []*RequestConfig `json:"requests" yaml:"requests"`
}

// RequestConfig defines a single HTTP request.
type RequestConfig struct {
	// Name for this request (used in metrics)
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// HTTP method
	Method string `json:"method" yaml:"method"`

	// URL (supports variable substitution)
	URL string `json:"url" yaml:"url"`

	// Headers
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`

	// Body (supports variable substitution)
	Body string `json:"body,omitempty" yaml:"body,omitempty"`

	// Timeout for this specific request (optional)
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Think time after this request
	ThinkTime time.Duration `json:"thinkTime,omitempty" yaml:"thinkTime,omitempty"`

	// Variable extraction from response
	Extract []ExtractConfig `json:"extract,omitempty" yaml:"extract,omitempty"`
}

// ExtractConfig defines how to extract variables from a response.
type ExtractConfig struct {
	// Name of the variable to store
	Name string `json:"name" yaml:"name"`

	// Source: "body", "header", "status"
	Source string `json:"source" yaml:"source"`

	// Path: header name, or JSONPath for body
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Regex pattern (optional, for body extraction)
	Regex string `json:"regex,omitempty" yaml:"regex,omitempty"`
}
