package v2

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// VUScheduler manages the lifecycle of Virtual Users.
//
// It provides:
// - VU pool management (spawning/ stopping VUs)
// - Shared HTTP client configuration
// - Graceful shutdown coordination
//
// The scheduler is used by executors to control VU counts.
type VUScheduler struct {
	// Scenario to execute
	scenario *Scenario

	// Metrics engine
	metrics *metrics.Engine

	// HTTP client configuration
	httpClientConfig HTTPClientConfig

	// Active VUs
	vus   map[int]*VirtualUser
	vusMu sync.RWMutex

	// VU ID counter
	nextVUID atomic.Int32

	// Shared HTTP client (if configured)
	sharedClient *http.Client

	// Shutdown coordination
	shutdownCh chan struct{}
	shutdownWg sync.WaitGroup
}

// HTTPClientConfig contains HTTP client configuration.
type HTTPClientConfig struct {
	// Timeout for HTTP requests
	Timeout time.Duration

	// MaxIdleConns controls the maximum number of idle connections
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum idle connections per host
	MaxIdleConnsPerHost int

	// MaxConnsPerHost limits the total connections per host
	MaxConnsPerHost int

	// IdleConnTimeout is how long idle connections are kept alive
	IdleConnTimeout time.Duration

	// DisableKeepAlives disables HTTP keep-alives
	DisableKeepAlives bool

	// DisableCompression disables automatic decompression
	DisableCompression bool

	// InsecureSkipVerify skips TLS certificate verification
	InsecureSkipVerify bool

	// UseSharedClient indicates whether VUs share a single HTTP client
	UseSharedClient bool
}

// DefaultHTTPClientConfig returns sensible defaults for load testing.
func DefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:             30 * time.Second,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     0, // Unlimited
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		InsecureSkipVerify:  false,
		UseSharedClient:     true, // Shared by default for connection pooling
	}
}

// NewVUScheduler creates a new VU scheduler.
func NewVUScheduler(scenario *Scenario, metricsEngine *metrics.Engine, httpConfig HTTPClientConfig) *VUScheduler {
	scheduler := &VUScheduler{
		scenario:         scenario,
		metrics:          metricsEngine,
		httpClientConfig: httpConfig,
		vus:              make(map[int]*VirtualUser),
		shutdownCh:       make(chan struct{}),
	}

	// Create shared HTTP client if configured
	if httpConfig.UseSharedClient {
		scheduler.sharedClient = scheduler.createHTTPClient()
	}

	return scheduler
}

// createHTTPClient creates an HTTP client with the configured settings.
func (s *VUScheduler) createHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        s.httpClientConfig.MaxIdleConns,
		MaxIdleConnsPerHost: s.httpClientConfig.MaxIdleConnsPerHost,
		MaxConnsPerHost:     s.httpClientConfig.MaxConnsPerHost,
		IdleConnTimeout:     s.httpClientConfig.IdleConnTimeout,
		DisableKeepAlives:   s.httpClientConfig.DisableKeepAlives,
		DisableCompression:  s.httpClientConfig.DisableCompression,
	}

	// Note: TLS configuration would be added here if InsecureSkipVerify is set
	// This requires crypto/tls import and more involved setup

	return &http.Client{
		Transport: transport,
		Timeout:   s.httpClientConfig.Timeout,
	}
}

// SpawnVU creates and returns a new Virtual User.
//
// The VU is registered with the scheduler but not started.
// The caller is responsible for running the VU.
func (s *VUScheduler) SpawnVU() *VirtualUser {
	id := int(s.nextVUID.Add(1))

	// Use shared client or create per-VU client
	var client *http.Client
	if s.httpClientConfig.UseSharedClient {
		client = s.sharedClient
	} else {
		client = s.createHTTPClient()
	}

	vu := NewVirtualUser(id, s.scenario, client, s.metrics)

	s.vusMu.Lock()
	s.vus[id] = vu
	s.vusMu.Unlock()

	return vu
}

// GetVU returns a VU by ID, or nil if not found.
func (s *VUScheduler) GetVU(id int) *VirtualUser {
	s.vusMu.RLock()
	defer s.vusMu.RUnlock()
	return s.vus[id]
}

// GetActiveVUs returns all currently active VUs.
func (s *VUScheduler) GetActiveVUs() []*VirtualUser {
	s.vusMu.RLock()
	defer s.vusMu.RUnlock()

	result := make([]*VirtualUser, 0, len(s.vus))
	for _, vu := range s.vus {
		if vu.GetState() != VUStateStopped {
			result = append(result, vu)
		}
	}
	return result
}

// GetActiveVUCount returns the count of non-stopped VUs.
func (s *VUScheduler) GetActiveVUCount() int {
	s.vusMu.RLock()
	defer s.vusMu.RUnlock()

	count := 0
	for _, vu := range s.vus {
		if vu.GetState() != VUStateStopped {
			count++
		}
	}
	return count
}

// StopVU requests a specific VU to stop.
func (s *VUScheduler) StopVU(id int) {
	s.vusMu.RLock()
	vu, exists := s.vus[id]
	s.vusMu.RUnlock()

	if exists {
		vu.RequestStop()
	}
}

// StopAllVUs requests all VUs to stop.
func (s *VUScheduler) StopAllVUs() {
	s.vusMu.RLock()
	defer s.vusMu.RUnlock()

	for _, vu := range s.vus {
		vu.RequestStop()
	}
}

// RemoveVU removes a VU from the scheduler.
// The VU should be stopped before calling this.
func (s *VUScheduler) RemoveVU(id int) {
	s.vusMu.Lock()
	defer s.vusMu.Unlock()

	if vu, exists := s.vus[id]; exists {
		vu.MarkStopped()
		delete(s.vus, id)
	}
}

// WaitForAllVUs waits for all VUs to stop with a timeout.
//
// Returns the number of VUs that did not stop within the timeout.
func (s *VUScheduler) WaitForAllVUs(timeout time.Duration) int {
	deadline := time.Now().Add(timeout)

	s.vusMu.RLock()
	vus := make([]*VirtualUser, 0, len(s.vus))
	for _, vu := range s.vus {
		vus = append(vus, vu)
	}
	s.vusMu.RUnlock()

	notStopped := 0
	for _, vu := range vus {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			// Timeout expired
			notStopped++
			continue
		}

		if !vu.WaitForStop(remaining) {
			notStopped++
		}
	}

	return notStopped
}

// RunVU runs a VU until it's stopped or the context is cancelled.
//
// This is a helper method for executors. It runs iterations continuously
// and handles the VU lifecycle automatically.
//
// Parameters:
//   - ctx: Context for cancellation
//   - vu: The VU to run
//   - pacing: Optional pacing between iterations
func (s *VUScheduler) RunVU(ctx context.Context, vu *VirtualUser, pacing time.Duration) {
	s.shutdownWg.Add(1)
	defer s.shutdownWg.Done()
	defer vu.MarkStopped()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.shutdownCh:
			return
		default:
		}

		// Check if VU was stopped
		if vu.GetState() == VUStateStopping || vu.GetState() == VUStateStopped {
			return
		}

		// Run one iteration
		err := vu.RunIteration(ctx)
		if err != nil {
			// Context cancelled or VU stopping
			if ctx.Err() != nil || vu.GetState() == VUStateStopping {
				return
			}
			// Log other errors but continue
		}

		// Apply pacing between iterations
		if pacing > 0 {
			select {
			case <-ctx.Done():
				return
			case <-s.shutdownCh:
				return
			case <-time.After(pacing):
			}
		}
	}
}

// Shutdown gracefully shuts down all VUs.
func (s *VUScheduler) Shutdown(timeout time.Duration) {
	// Signal shutdown
	close(s.shutdownCh)

	// Stop all VUs
	s.StopAllVUs()

	// Wait for VUs to finish with timeout
	done := make(chan struct{})
	go func() {
		s.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All VUs stopped
	case <-time.After(timeout):
		// Timeout expired
	}

	// Clean up shared client
	if s.sharedClient != nil {
		s.sharedClient.CloseIdleConnections()
	}
}

// UpdateMetrics updates the metrics engine with current VU count.
func (s *VUScheduler) UpdateMetrics() {
	count := s.GetActiveVUCount()
	s.metrics.SetActiveVUs(count)
}

// ScaleVUs adjusts the VU count to the target.
//
// This is a helper for ramping executors. It spawns or stops VUs
// as needed to reach the target count.
//
// Parameters:
//   - ctx: Context for spawning new VU goroutines
//   - target: Target number of VUs
//   - pacing: Pacing between iterations for new VUs
//   - onSpawn: Callback when a new VU is spawned (for goroutine management)
//
// Returns:
//   - Current VU count after adjustment
func (s *VUScheduler) ScaleVUs(ctx context.Context, target int, pacing time.Duration, onSpawn func(*VirtualUser)) int {
	current := s.GetActiveVUCount()

	if target > current {
		// Spawn new VUs
		for i := current; i < target; i++ {
			vu := s.SpawnVU()
			if onSpawn != nil {
				onSpawn(vu)
			}
		}
	} else if target < current {
		// Stop excess VUs
		excess := current - target
		stopped := 0

		s.vusMu.RLock()
		for _, vu := range s.vus {
			if stopped >= excess {
				break
			}
			if vu.GetState() != VUStateStopped && vu.GetState() != VUStateStopping {
				vu.RequestStop()
				stopped++
			}
		}
		s.vusMu.RUnlock()
	}

	s.UpdateMetrics()
	return s.GetActiveVUCount()
}
