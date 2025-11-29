# Performance Engine Technical Specification

This document provides detailed Go code designs and interfaces for implementing the new performance engine.

---

## 1. Executor System

### 1.1 Executor Interface

```go
// executor/executor.go

package executor

import (
    "context"
    "time"
)

// ExecutorType identifies the type of executor
type ExecutorType string

const (
    ExecutorConstantVUs        ExecutorType = "constant-vus"
    ExecutorRampingVUs         ExecutorType = "ramping-vus"
    ExecutorConstantArrivalRate ExecutorType = "constant-arrival-rate"
    ExecutorRampingArrivalRate  ExecutorType = "ramping-arrival-rate"
    ExecutorPerVUIterations    ExecutorType = "per-vu-iterations"
    ExecutorSharedIterations   ExecutorType = "shared-iterations"
)

// Executor defines the interface for load generation strategies
type Executor interface {
    // Type returns the executor type
    Type() ExecutorType
    
    // Init initializes the executor with configuration
    Init(ctx context.Context, config *ExecutorConfig) error
    
    // Run starts the executor and blocks until completion
    Run(ctx context.Context, engine *Engine) error
    
    // GetProgress returns current progress (0.0 to 1.0)
    GetProgress() float64
    
    // GetActiveVUs returns current active VU count
    GetActiveVUs() int
    
    // GetStats returns executor-specific stats
    GetStats() *ExecutorStats
    
    // Stop gracefully stops the executor
    Stop(ctx context.Context) error
}

// ExecutorConfig contains configuration for an executor
type ExecutorConfig struct {
    Name          string            `json:"name" yaml:"name"`
    Type          ExecutorType      `json:"type" yaml:"type"`
    
    // VU-based executors
    VUs           int               `json:"vus,omitempty" yaml:"vus,omitempty"`
    Duration      time.Duration     `json:"duration,omitempty" yaml:"duration,omitempty"`
    Iterations    int64             `json:"iterations,omitempty" yaml:"iterations,omitempty"`
    
    // Arrival-rate executors
    Rate          float64           `json:"rate,omitempty" yaml:"rate,omitempty"`           // iterations/second
    PreAllocatedVUs int             `json:"preAllocatedVUs,omitempty" yaml:"preAllocatedVUs,omitempty"`
    MaxVUs        int               `json:"maxVUs,omitempty" yaml:"maxVUs,omitempty"`
    
    // Stages (for ramping executors)
    Stages        []Stage           `json:"stages,omitempty" yaml:"stages,omitempty"`
    
    // Graceful stop timeout
    GracefulStop  time.Duration     `json:"gracefulStop,omitempty" yaml:"gracefulStop,omitempty"`
    
    // Optional pacing between iterations
    Pacing        *PacingConfig     `json:"pacing,omitempty" yaml:"pacing,omitempty"`
}

// Stage defines a stage in ramping executors
type Stage struct {
    Duration time.Duration `json:"duration" yaml:"duration"`
    Target   int           `json:"target" yaml:"target"` // VU count or RPS depending on executor
    Name     string        `json:"name,omitempty" yaml:"name,omitempty"`
}

// PacingConfig controls time between iterations
type PacingConfig struct {
    Type     PacingType    `json:"type" yaml:"type"`
    Duration time.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`
    Min      time.Duration `json:"min,omitempty" yaml:"min,omitempty"`
    Max      time.Duration `json:"max,omitempty" yaml:"max,omitempty"`
}

type PacingType string

const (
    PacingNone     PacingType = "none"
    PacingConstant PacingType = "constant"
    PacingRandom   PacingType = "random"
)

// ExecutorStats contains real-time executor statistics
type ExecutorStats struct {
    StartTime       time.Time     `json:"startTime"`
    CurrentTime     time.Time     `json:"currentTime"`
    Elapsed         time.Duration `json:"elapsed"`
    TotalDuration   time.Duration `json:"totalDuration"`
    ActiveVUs       int           `json:"activeVUs"`
    TargetVUs       int           `json:"targetVUs"`
    Iterations      int64         `json:"iterations"`
    CurrentStage    int           `json:"currentStage"`
    CurrentRate     float64       `json:"currentRate"` // For arrival-rate executors
    TargetRate      float64       `json:"targetRate"`
}
```

### 1.2 Constant VUs Executor

```go
// executor/constant_vus.go

package executor

import (
    "context"
    "sync"
    "sync/atomic"
    "time"
)

// ConstantVUsExecutor runs a fixed number of VUs for a duration
type ConstantVUsExecutor struct {
    config      *ExecutorConfig
    vuScheduler *VUScheduler
    metrics     *MetricsEngine
    
    // State
    startTime   time.Time
    activeVUs   atomic.Int32
    iterations  atomic.Int64
    running     atomic.Bool
    
    mu          sync.RWMutex
}

func NewConstantVUsExecutor() *ConstantVUsExecutor {
    return &ConstantVUsExecutor{}
}

func (e *ConstantVUsExecutor) Type() ExecutorType {
    return ExecutorConstantVUs
}

func (e *ConstantVUsExecutor) Init(ctx context.Context, config *ExecutorConfig) error {
    e.config = config
    
    // Validate config
    if config.VUs <= 0 {
        return fmt.Errorf("vus must be > 0, got %d", config.VUs)
    }
    if config.Duration <= 0 {
        return fmt.Errorf("duration must be > 0, got %v", config.Duration)
    }
    
    return nil
}

func (e *ConstantVUsExecutor) Run(ctx context.Context, engine *Engine) error {
    e.running.Store(true)
    e.startTime = time.Now()
    e.metrics = engine.Metrics
    e.vuScheduler = engine.VUScheduler
    
    // Create context with duration timeout
    runCtx, cancel := context.WithTimeout(ctx, e.config.Duration)
    defer cancel()
    
    // Spawn all VUs
    var wg sync.WaitGroup
    for i := 0; i < e.config.VUs; i++ {
        wg.Add(1)
        go func(vuID int) {
            defer wg.Done()
            e.runVU(runCtx, vuID)
        }(i)
    }
    
    // Wait for all VUs to complete
    wg.Wait()
    e.running.Store(false)
    
    return nil
}

func (e *ConstantVUsExecutor) runVU(ctx context.Context, vuID int) {
    e.activeVUs.Add(1)
    defer e.activeVUs.Add(-1)
    
    vu := e.vuScheduler.GetVU(vuID)
    
    // Run iterations until context cancelled
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Execute one iteration
            err := vu.RunIteration(ctx)
            if err != nil {
                // Log error but continue
            }
            e.iterations.Add(1)
            
            // Apply pacing between iterations
            if e.config.Pacing != nil {
                e.applyPacing(ctx)
            }
        }
    }
}

func (e *ConstantVUsExecutor) applyPacing(ctx context.Context) {
    if e.config.Pacing == nil || e.config.Pacing.Type == PacingNone {
        return
    }
    
    var wait time.Duration
    switch e.config.Pacing.Type {
    case PacingConstant:
        wait = e.config.Pacing.Duration
    case PacingRandom:
        // Random duration between min and max
        diff := e.config.Pacing.Max - e.config.Pacing.Min
        wait = e.config.Pacing.Min + time.Duration(rand.Int63n(int64(diff)))
    }
    
    select {
    case <-ctx.Done():
    case <-time.After(wait):
    }
}

func (e *ConstantVUsExecutor) GetProgress() float64 {
    if !e.running.Load() {
        return 1.0
    }
    elapsed := time.Since(e.startTime)
    return min(float64(elapsed)/float64(e.config.Duration), 1.0)
}

func (e *ConstantVUsExecutor) GetActiveVUs() int {
    return int(e.activeVUs.Load())
}

func (e *ConstantVUsExecutor) GetStats() *ExecutorStats {
    return &ExecutorStats{
        StartTime:     e.startTime,
        CurrentTime:   time.Now(),
        Elapsed:       time.Since(e.startTime),
        TotalDuration: e.config.Duration,
        ActiveVUs:     int(e.activeVUs.Load()),
        TargetVUs:     e.config.VUs,
        Iterations:    e.iterations.Load(),
    }
}

func (e *ConstantVUsExecutor) Stop(ctx context.Context) error {
    // Context cancellation handles this
    return nil
}
```

### 1.3 Ramping VUs Executor

```go
// executor/ramping_vus.go

package executor

// RampingVUsExecutor ramps VU count up and down according to stages
type RampingVUsExecutor struct {
    config      *ExecutorConfig
    vuScheduler *VUScheduler
    metrics     *MetricsEngine
    
    startTime    time.Time
    activeVUs    atomic.Int32
    targetVUs    atomic.Int32
    iterations   atomic.Int64
    currentStage atomic.Int32
    running      atomic.Bool
    
    mu           sync.RWMutex
    vus          []*VirtualUser
    vuMu         sync.Mutex
}

func (e *RampingVUsExecutor) Run(ctx context.Context, engine *Engine) error {
    e.running.Store(true)
    e.startTime = time.Now()
    e.metrics = engine.Metrics
    e.vuScheduler = engine.VUScheduler
    
    // Calculate total duration from stages
    var totalDuration time.Duration
    for _, stage := range e.config.Stages {
        totalDuration += stage.Duration
    }
    
    runCtx, cancel := context.WithTimeout(ctx, totalDuration)
    defer cancel()
    
    // Start VU controller (adjusts VU count smoothly)
    go e.vuController(runCtx)
    
    // Wait for completion
    <-runCtx.Done()
    
    // Graceful shutdown - wait for VUs to finish current iteration
    e.gracefulShutdown()
    
    e.running.Store(false)
    return nil
}

func (e *RampingVUsExecutor) vuController(ctx context.Context) {
    ticker := time.NewTicker(100 * time.Millisecond) // Adjust VUs every 100ms
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            targetVUs := e.calculateTargetVUs()
            e.targetVUs.Store(int32(targetVUs))
            e.adjustVUs(ctx, targetVUs)
        }
    }
}

func (e *RampingVUsExecutor) calculateTargetVUs() int {
    elapsed := time.Since(e.startTime)
    
    // Find current stage and interpolate target
    var stageStart time.Duration
    for i, stage := range e.config.Stages {
        stageEnd := stageStart + stage.Duration
        
        if elapsed < stageEnd {
            e.currentStage.Store(int32(i))
            
            // Calculate progress within this stage
            stageProgress := float64(elapsed-stageStart) / float64(stage.Duration)
            
            // Get previous stage target (or 0 for first stage)
            prevTarget := 0
            if i > 0 {
                prevTarget = e.config.Stages[i-1].Target
            }
            
            // Linear interpolation between previous and current target
            return prevTarget + int(float64(stage.Target-prevTarget)*stageProgress)
        }
        
        stageStart = stageEnd
    }
    
    // Past all stages - return last target
    if len(e.config.Stages) > 0 {
        return e.config.Stages[len(e.config.Stages)-1].Target
    }
    return 0
}

func (e *RampingVUsExecutor) adjustVUs(ctx context.Context, targetVUs int) {
    e.vuMu.Lock()
    defer e.vuMu.Unlock()
    
    currentVUs := len(e.vus)
    
    if targetVUs > currentVUs {
        // Spawn new VUs
        for i := currentVUs; i < targetVUs; i++ {
            vu := e.vuScheduler.SpawnVU(i)
            e.vus = append(e.vus, vu)
            go e.runVU(ctx, vu)
        }
    } else if targetVUs < currentVUs {
        // Gracefully stop excess VUs
        for i := currentVUs - 1; i >= targetVUs; i-- {
            e.vus[i].RequestStop()
            e.vus = e.vus[:i]
        }
    }
}

func (e *RampingVUsExecutor) gracefulShutdown() {
    e.vuMu.Lock()
    defer e.vuMu.Unlock()
    
    // Request all VUs to stop
    for _, vu := range e.vus {
        vu.RequestStop()
    }
    
    // Wait with timeout
    gracefulStop := e.config.GracefulStop
    if gracefulStop == 0 {
        gracefulStop = 30 * time.Second
    }
    
    deadline := time.Now().Add(gracefulStop)
    for _, vu := range e.vus {
        remaining := time.Until(deadline)
        if remaining <= 0 {
            break
        }
        vu.WaitForStop(remaining)
    }
}
```

### 1.4 Constant Arrival Rate Executor

```go
// executor/constant_rate.go

package executor

// ConstantArrivalRateExecutor maintains a fixed iteration rate (open model)
type ConstantArrivalRateExecutor struct {
    config       *ExecutorConfig
    vuScheduler  *VUScheduler
    metrics      *MetricsEngine
    rateLimiter  *LeakyBucket
    
    startTime    time.Time
    activeVUs    atomic.Int32
    iterations   atomic.Int64
    running      atomic.Bool
    
    vuPool       chan *VirtualUser
}

func (e *ConstantArrivalRateExecutor) Run(ctx context.Context, engine *Engine) error {
    e.running.Store(true)
    e.startTime = time.Now()
    e.metrics = engine.Metrics
    e.vuScheduler = engine.VUScheduler
    
    // Initialize leaky bucket with target rate
    e.rateLimiter = NewLeakyBucket(e.config.Rate)
    
    // Pre-allocate VU pool
    e.vuPool = make(chan *VirtualUser, e.config.MaxVUs)
    for i := 0; i < e.config.PreAllocatedVUs; i++ {
        vu := e.vuScheduler.SpawnVU(i)
        e.vuPool <- vu
    }
    
    runCtx, cancel := context.WithTimeout(ctx, e.config.Duration)
    defer cancel()
    
    // Run iteration scheduler
    e.runScheduler(runCtx)
    
    e.running.Store(false)
    return nil
}

func (e *ConstantArrivalRateExecutor) runScheduler(ctx context.Context) {
    for {
        // Get next scheduled iteration time from leaky bucket
        nextTime := e.rateLimiter.Next()
        
        // Wait until scheduled time
        waitDuration := time.Until(nextTime)
        if waitDuration > 0 {
            select {
            case <-ctx.Done():
                return
            case <-time.After(waitDuration):
            }
        }
        
        // Check if we should still be running
        select {
        case <-ctx.Done():
            return
        default:
        }
        
        // Get a VU from the pool (or spawn new one if needed)
        vu := e.getVU(ctx)
        if vu == nil {
            continue // Context cancelled
        }
        
        // Run iteration asynchronously
        go func(vu *VirtualUser) {
            e.activeVUs.Add(1)
            defer e.activeVUs.Add(-1)
            
            err := vu.RunIteration(ctx)
            if err != nil {
                // Log error
            }
            e.iterations.Add(1)
            
            // Return VU to pool
            e.returnVU(vu)
        }(vu)
    }
}

func (e *ConstantArrivalRateExecutor) getVU(ctx context.Context) *VirtualUser {
    select {
    case vu := <-e.vuPool:
        return vu
    default:
        // Pool empty - spawn new VU if under max
        if int(e.activeVUs.Load()) < e.config.MaxVUs {
            return e.vuScheduler.SpawnVU(int(e.iterations.Load()))
        }
        // At max VUs - wait for one to become available
        select {
        case <-ctx.Done():
            return nil
        case vu := <-e.vuPool:
            return vu
        }
    }
}

func (e *ConstantArrivalRateExecutor) returnVU(vu *VirtualUser) {
    select {
    case e.vuPool <- vu:
    default:
        // Pool full, discard VU
        vu.Cleanup()
    }
}
```

---

## 2. Virtual User (VU) Model

```go
// vu.go

package performance

import (
    "context"
    "net/http"
    "sync"
    "sync/atomic"
    "time"
)

// VUState represents the lifecycle state of a VU
type VUState int32

const (
    VUStateIdle VUState = iota
    VUStateRunning
    VUStateStopping
    VUStateStopped
)

// VirtualUser represents a single simulated user
type VirtualUser struct {
    ID          int
    Scenario    *Scenario
    HTTPClient  *http.Client
    Metrics     *MetricsEngine
    
    // Lifecycle management
    state       atomic.Int32  // VUState
    stopCh      chan struct{}
    doneCh      chan struct{}
    
    // Iteration tracking
    iteration   atomic.Int64
    
    // Per-VU data (for variable scope)
    data        map[string]interface{}
    dataMu      sync.RWMutex
    
    // Timing
    lastIterStart time.Time
}

func NewVirtualUser(id int, scenario *Scenario, httpClient *http.Client, metrics *MetricsEngine) *VirtualUser {
    return &VirtualUser{
        ID:         id,
        Scenario:   scenario,
        HTTPClient: httpClient,
        Metrics:    metrics,
        stopCh:     make(chan struct{}),
        doneCh:     make(chan struct{}),
        data:       make(map[string]interface{}),
    }
}

// RunIteration executes a single iteration of the scenario
func (vu *VirtualUser) RunIteration(ctx context.Context) error {
    if vu.GetState() != VUStateRunning && vu.GetState() != VUStateIdle {
        return fmt.Errorf("VU %d not in runnable state", vu.ID)
    }
    
    vu.state.Store(int32(VUStateRunning))
    vu.lastIterStart = time.Now()
    vu.iteration.Add(1)
    
    // Execute all requests in the scenario
    for i, req := range vu.Scenario.Requests {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-vu.stopCh:
            return nil // Graceful stop
        default:
        }
        
        // Execute request
        result, err := vu.executeRequest(ctx, req)
        if err != nil {
            // Record error but continue with next request
            vu.Metrics.RecordError(err)
            continue
        }
        
        // Record metrics
        vu.Metrics.RecordRequest(result)
        
        // Apply think time between requests (if configured)
        if req.ThinkTime > 0 && i < len(vu.Scenario.Requests)-1 {
            vu.applyThinkTime(ctx, req.ThinkTime)
        }
    }
    
    vu.state.Store(int32(VUStateIdle))
    return nil
}

func (vu *VirtualUser) executeRequest(ctx context.Context, req *RequestConfig) (*RequestResult, error) {
    startTime := time.Now()
    
    // Build HTTP request
    httpReq, err := vu.buildRequest(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to build request: %w", err)
    }
    
    // Execute request with timing
    resp, err := vu.HTTPClient.Do(httpReq)
    endTime := time.Now()
    
    result := &RequestResult{
        VUID:       vu.ID,
        Iteration:  vu.iteration.Load(),
        RequestName: req.Name,
        StartTime:  startTime,
        EndTime:    endTime,
        Duration:   endTime.Sub(startTime),
    }
    
    if err != nil {
        result.Error = err
        return result, nil // Return result with error for metrics
    }
    
    defer resp.Body.Close()
    
    // Read response body for byte counting
    body, _ := io.ReadAll(resp.Body)
    result.StatusCode = resp.StatusCode
    result.BytesReceived = int64(len(body))
    
    // Check assertions
    if req.Assertions != nil {
        result.AssertionResults = vu.checkAssertions(req.Assertions, resp, body)
    }
    
    return result, nil
}

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
    
    // Add headers
    for key, value := range req.Headers {
        httpReq.Header.Set(key, vu.resolveVariables(value))
    }
    
    return httpReq, nil
}

func (vu *VirtualUser) resolveVariables(input string) string {
    // Replace {{varName}} with values from vu.data and scenario.Variables
    // Implementation uses regex or template engine
    return input // TODO: implement
}

func (vu *VirtualUser) applyThinkTime(ctx context.Context, duration time.Duration) {
    select {
    case <-ctx.Done():
    case <-vu.stopCh:
    case <-time.After(duration):
    }
}

// Lifecycle methods

func (vu *VirtualUser) GetState() VUState {
    return VUState(vu.state.Load())
}

func (vu *VirtualUser) RequestStop() {
    if vu.state.CompareAndSwap(int32(VUStateRunning), int32(VUStateStopping)) ||
       vu.state.CompareAndSwap(int32(VUStateIdle), int32(VUStateStopping)) {
        close(vu.stopCh)
    }
}

func (vu *VirtualUser) WaitForStop(timeout time.Duration) bool {
    select {
    case <-vu.doneCh:
        return true
    case <-time.After(timeout):
        return false
    }
}

func (vu *VirtualUser) Cleanup() {
    vu.state.Store(int32(VUStateStopped))
    close(vu.doneCh)
}

// Data methods for variable scope

func (vu *VirtualUser) SetData(key string, value interface{}) {
    vu.dataMu.Lock()
    defer vu.dataMu.Unlock()
    vu.data[key] = value
}

func (vu *VirtualUser) GetData(key string) (interface{}, bool) {
    vu.dataMu.RLock()
    defer vu.dataMu.RUnlock()
    val, ok := vu.data[key]
    return val, ok
}
```

---

## 3. Metrics Engine with HDR Histogram

```go
// metrics/engine.go

package metrics

import (
    "context"
    "sync"
    "sync/atomic"
    "time"
    
    "github.com/HdrHistogram/hdrhistogram-go"
)

// MetricsEngine collects and aggregates performance metrics
type MetricsEngine struct {
    // HDR Histograms for latency (lock-free reads, locked writes)
    latencyHist     *hdrhistogram.Histogram  // Overall latency
    latencyHistMu   sync.Mutex
    
    // Per-request-name histograms
    requestHists    map[string]*hdrhistogram.Histogram
    requestHistsMu  sync.RWMutex
    
    // Atomic counters
    totalRequests   atomic.Int64
    successRequests atomic.Int64
    failedRequests  atomic.Int64
    totalBytes      atomic.Int64
    
    // Time-bucketed metrics
    bucketStore     *TimeBucketStore
    
    // Current phase
    currentPhase    Phase
    phaseMu         sync.RWMutex
    
    // Background emitter
    emitterCtx      context.Context
    emitterCancel   context.CancelFunc
    
    // Start time
    startTime       time.Time
}

type Phase string

const (
    PhaseWarmup    Phase = "warmup"
    PhaseRampUp    Phase = "ramp-up"
    PhaseSteady    Phase = "steady"
    PhaseRampDown  Phase = "ramp-down"
    PhaseCooldown  Phase = "cooldown"
)

// NewMetricsEngine creates a new metrics engine
func NewMetricsEngine() *MetricsEngine {
    ctx, cancel := context.WithCancel(context.Background())
    
    me := &MetricsEngine{
        // HDR histogram: 1μs to 1 hour, 3 significant figures
        latencyHist:    hdrhistogram.New(1, 3600000000, 3),
        requestHists:   make(map[string]*hdrhistogram.Histogram),
        bucketStore:    NewTimeBucketStore(3600), // Keep 1 hour of data
        emitterCtx:     ctx,
        emitterCancel:  cancel,
        startTime:      time.Now(),
        currentPhase:   PhaseWarmup,
    }
    
    // Start background emitter
    go me.runEmitter()
    
    return me
}

// RecordRequest records a completed request
func (me *MetricsEngine) RecordRequest(result *RequestResult) {
    // Update counters atomically
    me.totalRequests.Add(1)
    me.totalBytes.Add(result.BytesReceived)
    
    if result.Error != nil || result.StatusCode >= 400 {
        me.failedRequests.Add(1)
    } else {
        me.successRequests.Add(1)
    }
    
    // Record latency in histogram
    latencyMicros := result.Duration.Microseconds()
    
    me.latencyHistMu.Lock()
    me.latencyHist.RecordValue(latencyMicros)
    me.latencyHistMu.Unlock()
    
    // Record per-request-name histogram
    if result.RequestName != "" {
        me.recordRequestHistogram(result.RequestName, latencyMicros)
    }
    
    // Update current bucket
    me.bucketStore.RecordRequest(result)
}

func (me *MetricsEngine) recordRequestHistogram(name string, latencyMicros int64) {
    me.requestHistsMu.RLock()
    hist, exists := me.requestHists[name]
    me.requestHistsMu.RUnlock()
    
    if !exists {
        me.requestHistsMu.Lock()
        // Double-check after acquiring write lock
        if hist, exists = me.requestHists[name]; !exists {
            hist = hdrhistogram.New(1, 3600000000, 3)
            me.requestHists[name] = hist
        }
        me.requestHistsMu.Unlock()
    }
    
    hist.RecordValue(latencyMicros)
}

// runEmitter runs the background time-bucket emitter
func (me *MetricsEngine) runEmitter() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-me.emitterCtx.Done():
            return
        case <-ticker.C:
            me.emitBucket()
        }
    }
}

func (me *MetricsEngine) emitBucket() {
    me.phaseMu.RLock()
    phase := me.currentPhase
    me.phaseMu.RUnlock()
    
    // Get current histogram percentiles
    me.latencyHistMu.Lock()
    p50 := time.Duration(me.latencyHist.ValueAtQuantile(50)) * time.Microsecond
    p95 := time.Duration(me.latencyHist.ValueAtQuantile(95)) * time.Microsecond
    p99 := time.Duration(me.latencyHist.ValueAtQuantile(99)) * time.Microsecond
    me.latencyHistMu.Unlock()
    
    bucket := &TimeBucket{
        Timestamp:   time.Now(),
        Requests:    me.totalRequests.Load(),
        Successes:   me.successRequests.Load(),
        Failures:    me.failedRequests.Load(),
        Bytes:       me.totalBytes.Load(),
        LatencyP50:  p50,
        LatencyP95:  p95,
        LatencyP99:  p99,
        Phase:       phase,
    }
    
    me.bucketStore.AddBucket(bucket)
}

// SetPhase updates the current phase
func (me *MetricsEngine) SetPhase(phase Phase) {
    me.phaseMu.Lock()
    defer me.phaseMu.Unlock()
    me.currentPhase = phase
}

// GetSnapshot returns a point-in-time snapshot of metrics
func (me *MetricsEngine) GetSnapshot() *MetricsSnapshot {
    me.latencyHistMu.Lock()
    defer me.latencyHistMu.Unlock()
    
    elapsed := time.Since(me.startTime).Seconds()
    totalReqs := me.totalRequests.Load()
    
    return &MetricsSnapshot{
        TotalRequests:      totalReqs,
        SuccessfulRequests: me.successRequests.Load(),
        FailedRequests:     me.failedRequests.Load(),
        TotalBytes:         me.totalBytes.Load(),
        
        Latency: LatencyStats{
            Min:    time.Duration(me.latencyHist.Min()) * time.Microsecond,
            Max:    time.Duration(me.latencyHist.Max()) * time.Microsecond,
            Mean:   time.Duration(me.latencyHist.Mean()) * time.Microsecond,
            StdDev: time.Duration(me.latencyHist.StdDev()) * time.Microsecond,
            P50:    time.Duration(me.latencyHist.ValueAtQuantile(50)) * time.Microsecond,
            P90:    time.Duration(me.latencyHist.ValueAtQuantile(90)) * time.Microsecond,
            P95:    time.Duration(me.latencyHist.ValueAtQuantile(95)) * time.Microsecond,
            P99:    time.Duration(me.latencyHist.ValueAtQuantile(99)) * time.Microsecond,
        },
        
        Throughput: ThroughputStats{
            RequestsPerSecond: float64(totalReqs) / elapsed,
            BytesPerSecond:    float64(me.totalBytes.Load()) / elapsed,
        },
        
        ErrorRate: float64(me.failedRequests.Load()) / float64(max(totalReqs, 1)),
        
        Elapsed:   time.Since(me.startTime),
        Timestamp: time.Now(),
    }
}

// GetTimeSeries returns time-bucketed data
func (me *MetricsEngine) GetTimeSeries() []*TimeBucket {
    return me.bucketStore.GetBuckets()
}

// Stop stops the metrics engine
func (me *MetricsEngine) Stop() {
    me.emitterCancel()
    // Emit final bucket
    me.emitBucket()
}

// Reset resets all metrics
func (me *MetricsEngine) Reset() {
    me.latencyHistMu.Lock()
    me.latencyHist.Reset()
    me.latencyHistMu.Unlock()
    
    me.requestHistsMu.Lock()
    me.requestHists = make(map[string]*hdrhistogram.Histogram)
    me.requestHistsMu.Unlock()
    
    me.totalRequests.Store(0)
    me.successRequests.Store(0)
    me.failedRequests.Store(0)
    me.totalBytes.Store(0)
    
    me.bucketStore.Reset()
    me.startTime = time.Now()
}
```

### 3.1 Time Bucket Store

```go
// metrics/bucket.go

package metrics

import (
    "sync"
    "time"
)

// TimeBucket represents metrics for a 1-second interval
type TimeBucket struct {
    Timestamp  time.Time     `json:"timestamp"`
    Requests   int64         `json:"requests"`
    Successes  int64         `json:"successes"`
    Failures   int64         `json:"failures"`
    Bytes      int64         `json:"bytes"`
    LatencyP50 time.Duration `json:"latencyP50"`
    LatencyP95 time.Duration `json:"latencyP95"`
    LatencyP99 time.Duration `json:"latencyP99"`
    ActiveVUs  int           `json:"activeVUs"`
    Phase      Phase         `json:"phase"`
    
    // Delta values (requests in this interval, not cumulative)
    IntervalRequests int64   `json:"intervalRequests"`
    IntervalRPS      float64 `json:"intervalRPS"`
}

// TimeBucketStore stores time-bucketed metrics
type TimeBucketStore struct {
    buckets     []*TimeBucket
    maxBuckets  int
    mu          sync.RWMutex
    
    // For delta calculation
    lastRequests int64
    lastBytes    int64
    lastTime     time.Time
    
    // For recording within current bucket
    currentBucket *currentBucketData
    currentMu     sync.Mutex
}

type currentBucketData struct {
    requests    int64
    successes   int64
    failures    int64
    bytes       int64
    startTime   time.Time
}

func NewTimeBucketStore(maxBuckets int) *TimeBucketStore {
    return &TimeBucketStore{
        buckets:    make([]*TimeBucket, 0, maxBuckets),
        maxBuckets: maxBuckets,
        lastTime:   time.Now(),
        currentBucket: &currentBucketData{
            startTime: time.Now(),
        },
    }
}

// RecordRequest records a request into the current bucket
func (tbs *TimeBucketStore) RecordRequest(result *RequestResult) {
    tbs.currentMu.Lock()
    defer tbs.currentMu.Unlock()
    
    tbs.currentBucket.requests++
    tbs.currentBucket.bytes += result.BytesReceived
    
    if result.Error != nil || result.StatusCode >= 400 {
        tbs.currentBucket.failures++
    } else {
        tbs.currentBucket.successes++
    }
}

// AddBucket adds a new bucket and calculates deltas
func (tbs *TimeBucketStore) AddBucket(bucket *TimeBucket) {
    tbs.mu.Lock()
    defer tbs.mu.Unlock()
    
    // Calculate interval metrics
    tbs.currentMu.Lock()
    bucket.IntervalRequests = tbs.currentBucket.requests
    interval := time.Since(tbs.currentBucket.startTime).Seconds()
    if interval > 0 {
        bucket.IntervalRPS = float64(tbs.currentBucket.requests) / interval
    }
    // Reset current bucket
    tbs.currentBucket = &currentBucketData{
        startTime: time.Now(),
    }
    tbs.currentMu.Unlock()
    
    // Ring buffer behavior
    if len(tbs.buckets) >= tbs.maxBuckets {
        // Remove oldest
        tbs.buckets = tbs.buckets[1:]
    }
    
    tbs.buckets = append(tbs.buckets, bucket)
}

// GetBuckets returns a copy of all buckets
func (tbs *TimeBucketStore) GetBuckets() []*TimeBucket {
    tbs.mu.RLock()
    defer tbs.mu.RUnlock()
    
    result := make([]*TimeBucket, len(tbs.buckets))
    copy(result, tbs.buckets)
    return result
}

// GetBucketsForPhase returns buckets for a specific phase
func (tbs *TimeBucketStore) GetBucketsForPhase(phase Phase) []*TimeBucket {
    tbs.mu.RLock()
    defer tbs.mu.RUnlock()
    
    result := make([]*TimeBucket, 0)
    for _, b := range tbs.buckets {
        if b.Phase == phase {
            result = append(result, b)
        }
    }
    return result
}

// Reset clears all buckets
func (tbs *TimeBucketStore) Reset() {
    tbs.mu.Lock()
    defer tbs.mu.Unlock()
    
    tbs.buckets = make([]*TimeBucket, 0, tbs.maxBuckets)
    tbs.lastRequests = 0
    tbs.lastBytes = 0
    tbs.lastTime = time.Now()
}
```

---

## 4. Leaky Bucket Rate Limiter

```go
// rate/leaky_bucket.go

package rate

import (
    "sync"
    "time"
)

// LeakyBucket implements the leaky bucket algorithm for rate limiting
// Unlike token bucket, it focuses on "when to execute next" rather than
// "how many tokens available"
type LeakyBucket struct {
    rate        float64     // Iterations per second
    lastDrip    time.Time   // Last iteration timestamp
    accumulated float64     // Accumulated iterations (fractional)
    maxBurst    float64     // Maximum burst (typically 1.0 for strict timing)
    mu          sync.Mutex
}

// NewLeakyBucket creates a new leaky bucket
func NewLeakyBucket(rate float64) *LeakyBucket {
    if rate <= 0 {
        rate = 1.0
    }
    return &LeakyBucket{
        rate:     rate,
        lastDrip: time.Now(),
        maxBurst: 1.0,
    }
}

// Next returns when the next iteration should start
// This method is designed to be called repeatedly in a loop
func (lb *LeakyBucket) Next() time.Time {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(lb.lastDrip).Seconds()
    
    // Accumulate iterations based on elapsed time
    lb.accumulated += elapsed * lb.rate
    
    // Cap at max burst
    if lb.accumulated > lb.maxBurst {
        lb.accumulated = lb.maxBurst
    }
    
    lb.lastDrip = now
    
    if lb.accumulated >= 1.0 {
        // Can execute immediately
        lb.accumulated -= 1.0
        return now
    }
    
    // Calculate wait time for next iteration
    deficit := 1.0 - lb.accumulated
    waitSeconds := deficit / lb.rate
    lb.accumulated = 0 // Will be consumed when wait is over
    
    return now.Add(time.Duration(waitSeconds * float64(time.Second)))
}

// SetRate updates the rate (for ramping)
func (lb *LeakyBucket) SetRate(rate float64) {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    if rate <= 0 {
        rate = 1.0
    }
    
    // Don't carry over accumulated iterations when rate changes
    // This prevents bursting during ramp-down
    lb.rate = rate
    lb.accumulated = 0
    lb.lastDrip = time.Now()
}

// GetRate returns the current rate
func (lb *LeakyBucket) GetRate() float64 {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    return lb.rate
}

// SetMaxBurst sets the maximum burst size
func (lb *LeakyBucket) SetMaxBurst(burst float64) {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    lb.maxBurst = burst
}
```

---

## 5. Configuration Schema

```go
// config/schema.go

package config

import (
    "time"
    
    "github.com/wesleyorama2/lunge/internal/performance/executor"
)

// TestConfig is the root configuration structure
type TestConfig struct {
    Name        string            `json:"name" yaml:"name"`
    Description string            `json:"description,omitempty" yaml:"description,omitempty"`
    
    Settings    GlobalSettings    `json:"settings,omitempty" yaml:"settings,omitempty"`
    Variables   map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`
    Scenarios   map[string]*ScenarioConfig `json:"scenarios" yaml:"scenarios"`
    Thresholds  ThresholdsConfig  `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`
    Output      OutputConfig      `json:"output,omitempty" yaml:"output,omitempty"`
}

// GlobalSettings contains global test settings
type GlobalSettings struct {
    BaseURL               string        `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
    Timeout               Duration      `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    Keepalive             bool          `json:"keepalive,omitempty" yaml:"keepalive,omitempty"`
    MaxConnectionsPerHost int           `json:"maxConnectionsPerHost,omitempty" yaml:"maxConnectionsPerHost,omitempty"`
    InsecureSkipVerify    bool          `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
    UserAgent             string        `json:"userAgent,omitempty" yaml:"userAgent,omitempty"`
}

// ScenarioConfig defines a test scenario
type ScenarioConfig struct {
    Executor   executor.ExecutorType `json:"executor" yaml:"executor"`
    
    // VU-based executors
    VUs        int                   `json:"vus,omitempty" yaml:"vus,omitempty"`
    Duration   Duration              `json:"duration,omitempty" yaml:"duration,omitempty"`
    Iterations int64                 `json:"iterations,omitempty" yaml:"iterations,omitempty"`
    
    // Arrival-rate executors
    Rate           float64           `json:"rate,omitempty" yaml:"rate,omitempty"`
    PreAllocatedVUs int              `json:"preAllocatedVUs,omitempty" yaml:"preAllocatedVUs,omitempty"`
    MaxVUs         int               `json:"maxVUs,omitempty" yaml:"maxVUs,omitempty"`
    
    // Stages (for ramping executors)
    Stages         []StageConfig     `json:"stages,omitempty" yaml:"stages,omitempty"`
    
    // Graceful stop
    GracefulStop   Duration          `json:"gracefulStop,omitempty" yaml:"gracefulStop,omitempty"`
    
    // Pacing
    Pacing         *PacingConfig     `json:"pacing,omitempty" yaml:"pacing,omitempty"`
    
    // Requests in this scenario
    Requests       []RequestConfig   `json:"requests" yaml:"requests"`
}

// StageConfig defines a stage in ramping executors
type StageConfig struct {
    Duration Duration `json:"duration" yaml:"duration"`
    Target   int      `json:"target" yaml:"target"`
    Name     string   `json:"name,omitempty" yaml:"name,omitempty"`
}

// PacingConfig controls think time between iterations
type PacingConfig struct {
    Type     string   `json:"type" yaml:"type"`        // "none", "constant", "random"
    Duration Duration `json:"duration,omitempty" yaml:"duration,omitempty"`
    Min      Duration `json:"min,omitempty" yaml:"min,omitempty"`
    Max      Duration `json:"max,omitempty" yaml:"max,omitempty"`
}

// RequestConfig defines a single HTTP request
type RequestConfig struct {
    Name       string            `json:"name,omitempty" yaml:"name,omitempty"`
    Method     string            `json:"method" yaml:"method"`
    URL        string            `json:"url" yaml:"url"`
    Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
    Body       string            `json:"body,omitempty" yaml:"body,omitempty"`
    Timeout    Duration          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    ThinkTime  Duration          `json:"thinkTime,omitempty" yaml:"thinkTime,omitempty"`
    Assertions []AssertionConfig `json:"assertions,omitempty" yaml:"assertions,omitempty"`
    
    // Variable extraction
    Extract    []ExtractConfig   `json:"extract,omitempty" yaml:"extract,omitempty"`
}

// AssertionConfig defines an assertion on the response
type AssertionConfig struct {
    Type     string `json:"type" yaml:"type"`           // "status", "responseTime", "body", "header"
    Expected string `json:"expected,omitempty" yaml:"expected,omitempty"`
    Contains string `json:"contains,omitempty" yaml:"contains,omitempty"`
    Max      Duration `json:"max,omitempty" yaml:"max,omitempty"` // For responseTime
}

// ExtractConfig defines variable extraction from response
type ExtractConfig struct {
    Name     string `json:"name" yaml:"name"`
    Source   string `json:"source" yaml:"source"`   // "body", "header", "status"
    Path     string `json:"path,omitempty" yaml:"path,omitempty"` // JSONPath for body
    Regex    string `json:"regex,omitempty" yaml:"regex,omitempty"`
}

// ThresholdsConfig defines pass/fail criteria
type ThresholdsConfig struct {
    // Global thresholds
    HTTPReqDuration []string          `json:"http_req_duration,omitempty" yaml:"http_req_duration,omitempty"`
    HTTPReqFailed   []string          `json:"http_req_failed,omitempty" yaml:"http_req_failed,omitempty"`
    
    // Per-scenario thresholds
    Scenarios       map[string]map[string][]string `json:"scenarios,omitempty" yaml:"scenarios,omitempty"`
}

// OutputConfig defines output destinations
type OutputConfig struct {
    Console ConsoleOutputConfig `json:"console,omitempty" yaml:"console,omitempty"`
    HTML    HTMLOutputConfig    `json:"html,omitempty" yaml:"html,omitempty"`
    JSON    JSONOutputConfig    `json:"json,omitempty" yaml:"json,omitempty"`
}

type ConsoleOutputConfig struct {
    Enabled bool `json:"enabled" yaml:"enabled"`
    Summary bool `json:"summary" yaml:"summary"`
}

type HTMLOutputConfig struct {
    Enabled bool   `json:"enabled" yaml:"enabled"`
    Path    string `json:"path" yaml:"path"`
}

type JSONOutputConfig struct {
    Enabled bool   `json:"enabled" yaml:"enabled"`
    Path    string `json:"path" yaml:"path"`
}

// Duration is a custom type for JSON/YAML duration parsing
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
    // Handle both string ("30s") and number (30) formats
    // Implementation details...
    return nil
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
    // Similar to JSON
    return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
    return json.Marshal(time.Duration(d).String())
}
```

---

## 6. Implementation Checklist

### Phase 1: Core Foundation
- [ ] Create `internal/performance/v2/` directory structure
- [ ] Implement `MetricsEngine` with HDR histogram
- [ ] Implement `TimeBucketStore`
- [ ] Implement `VirtualUser` struct and lifecycle
- [ ] Implement `VUScheduler`
- [ ] Implement `LeakyBucket` rate limiter
- [ ] Implement `ConstantVUsExecutor`
- [ ] Unit tests for all core components

### Phase 2: Executors
- [ ] Implement `RampingVUsExecutor`
- [ ] Implement `ConstantArrivalRateExecutor`
- [ ] Implement `RampingArrivalRateExecutor`
- [ ] Implement `PerVUIterationsExecutor`
- [ ] Implement `SharedIterationsExecutor`
- [ ] Integration tests for executors

### Phase 3: Configuration & CLI
- [ ] Implement config schema and parser
- [ ] Implement config validator
- [ ] Update CLI for new executor flags
- [ ] Update CLI for stages syntax
- [ ] Migration examples

### Phase 4: Reporting
- [ ] Update HTML report template
- [ ] Implement threshold checking
- [ ] Console progress display
- [ ] JSON output format

### Phase 5: Polish
- [ ] End-to-end tests
- [ ] Performance benchmarks
- [ ] Documentation
- [ ] Example configs