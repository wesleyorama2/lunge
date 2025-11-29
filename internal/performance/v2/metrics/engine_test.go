package metrics

import (
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	defer engine.Stop()

	// Check initial state
	snapshot := engine.GetSnapshot()
	if snapshot.TotalRequests != 0 {
		t.Errorf("Initial TotalRequests = %d, want 0", snapshot.TotalRequests)
	}
	if snapshot.CurrentPhase != PhaseInit {
		t.Errorf("Initial phase = %v, want %v", snapshot.CurrentPhase, PhaseInit)
	}
}

func TestEngine_RecordLatency(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Record some latencies
	engine.RecordLatency(10*time.Millisecond, "test-request", true, 1000)
	engine.RecordLatency(20*time.Millisecond, "test-request", true, 2000)
	engine.RecordLatency(30*time.Millisecond, "test-request", false, 500)

	snapshot := engine.GetSnapshot()

	if snapshot.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", snapshot.TotalRequests)
	}
	if snapshot.SuccessRequests != 2 {
		t.Errorf("SuccessRequests = %d, want 2", snapshot.SuccessRequests)
	}
	if snapshot.FailedRequests != 1 {
		t.Errorf("FailedRequests = %d, want 1", snapshot.FailedRequests)
	}
	if snapshot.TotalBytes != 3500 {
		t.Errorf("TotalBytes = %d, want 3500", snapshot.TotalBytes)
	}
}

func TestEngine_LatencyPercentiles(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Record latencies with known distribution
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	for _, lat := range latencies {
		engine.RecordLatency(lat, "", true, 100)
	}

	percentiles := engine.GetLatencyPercentiles()

	// P50 should be around 50ms (with some tolerance for HDR histogram binning)
	if percentiles.P50 < 40*time.Millisecond || percentiles.P50 > 60*time.Millisecond {
		t.Errorf("P50 = %v, want ~50ms (±10ms)", percentiles.P50)
	}

	// P99 should be close to 100ms
	if percentiles.P99 < 90*time.Millisecond || percentiles.P99 > 110*time.Millisecond {
		t.Errorf("P99 = %v, want ~100ms (±10ms)", percentiles.P99)
	}

	// Min should be 10ms
	if percentiles.Min < 9*time.Millisecond || percentiles.Min > 11*time.Millisecond {
		t.Errorf("Min = %v, want ~10ms", percentiles.Min)
	}

	// Max should be 100ms
	if percentiles.Max < 99*time.Millisecond || percentiles.Max > 101*time.Millisecond {
		t.Errorf("Max = %v, want ~100ms", percentiles.Max)
	}
}

func TestEngine_Phase(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Initial phase
	if engine.GetPhase() != PhaseInit {
		t.Errorf("Initial phase = %v, want %v", engine.GetPhase(), PhaseInit)
	}

	// Set phases
	phases := []Phase{PhaseWarmup, PhaseRampUp, PhaseSteady, PhaseRampDown, PhaseDone}
	for _, phase := range phases {
		engine.SetPhase(phase)
		if engine.GetPhase() != phase {
			t.Errorf("After SetPhase(%v), GetPhase() = %v", phase, engine.GetPhase())
		}
	}

	// Check phase history
	history := engine.GetPhaseHistory()
	if len(history) != len(phases) {
		t.Errorf("PhaseHistory length = %d, want %d", len(history), len(phases))
	}
}

func TestEngine_ActiveVUs(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	if engine.GetActiveVUs() != 0 {
		t.Errorf("Initial ActiveVUs = %d, want 0", engine.GetActiveVUs())
	}

	engine.SetActiveVUs(10)
	if engine.GetActiveVUs() != 10 {
		t.Errorf("After SetActiveVUs(10), GetActiveVUs() = %d, want 10", engine.GetActiveVUs())
	}

	engine.SetActiveVUs(5)
	if engine.GetActiveVUs() != 5 {
		t.Errorf("After SetActiveVUs(5), GetActiveVUs() = %d, want 5", engine.GetActiveVUs())
	}
}

func TestEngine_RequestStats(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Record latencies for different request names
	engine.RecordLatency(10*time.Millisecond, "login", true, 100)
	engine.RecordLatency(15*time.Millisecond, "login", true, 100)
	engine.RecordLatency(50*time.Millisecond, "get-profile", true, 500)
	engine.RecordLatency(60*time.Millisecond, "get-profile", true, 500)

	stats := engine.GetRequestStats()

	if len(stats) != 2 {
		t.Errorf("RequestStats length = %d, want 2", len(stats))
	}

	loginStats, ok := stats["login"]
	if !ok {
		t.Fatal("Missing 'login' stats")
	}
	if loginStats.Count != 2 {
		t.Errorf("login count = %d, want 2", loginStats.Count)
	}

	profileStats, ok := stats["get-profile"]
	if !ok {
		t.Fatal("Missing 'get-profile' stats")
	}
	if profileStats.Count != 2 {
		t.Errorf("get-profile count = %d, want 2", profileStats.Count)
	}
}

func TestEngine_Reset(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Record some data
	engine.RecordLatency(10*time.Millisecond, "test", true, 100)
	engine.RecordLatency(20*time.Millisecond, "test", false, 200)
	engine.SetPhase(PhaseSteady)
	engine.SetActiveVUs(5)

	// Verify data exists
	snapshot := engine.GetSnapshot()
	if snapshot.TotalRequests != 2 {
		t.Errorf("Before reset, TotalRequests = %d, want 2", snapshot.TotalRequests)
	}

	// Reset
	engine.Reset()

	// Verify data is cleared
	snapshot = engine.GetSnapshot()
	if snapshot.TotalRequests != 0 {
		t.Errorf("After reset, TotalRequests = %d, want 0", snapshot.TotalRequests)
	}
	if snapshot.SuccessRequests != 0 {
		t.Errorf("After reset, SuccessRequests = %d, want 0", snapshot.SuccessRequests)
	}
	if snapshot.FailedRequests != 0 {
		t.Errorf("After reset, FailedRequests = %d, want 0", snapshot.FailedRequests)
	}
	if snapshot.CurrentPhase != PhaseInit {
		t.Errorf("After reset, phase = %v, want %v", snapshot.CurrentPhase, PhaseInit)
	}
	if snapshot.ActiveVUs != 0 {
		t.Errorf("After reset, ActiveVUs = %d, want 0", snapshot.ActiveVUs)
	}
}

func TestEngine_Snapshot(t *testing.T) {
	engine := NewEngine()
	defer engine.Stop()

	// Record some data
	for i := 0; i < 100; i++ {
		success := i%10 != 0 // 10% failure rate
		engine.RecordLatency(time.Duration(i+1)*time.Millisecond, "", success, 100)
	}

	engine.SetPhase(PhaseSteady)
	engine.SetActiveVUs(10)

	snapshot := engine.GetSnapshot()

	// Check counters
	if snapshot.TotalRequests != 100 {
		t.Errorf("TotalRequests = %d, want 100", snapshot.TotalRequests)
	}
	if snapshot.SuccessRequests != 90 {
		t.Errorf("SuccessRequests = %d, want 90", snapshot.SuccessRequests)
	}
	if snapshot.FailedRequests != 10 {
		t.Errorf("FailedRequests = %d, want 10", snapshot.FailedRequests)
	}

	// Check error rate
	expectedErrorRate := 0.10
	if snapshot.ErrorRate < expectedErrorRate-0.01 || snapshot.ErrorRate > expectedErrorRate+0.01 {
		t.Errorf("ErrorRate = %v, want ~%v", snapshot.ErrorRate, expectedErrorRate)
	}

	// Check phase and VUs
	if snapshot.CurrentPhase != PhaseSteady {
		t.Errorf("CurrentPhase = %v, want %v", snapshot.CurrentPhase, PhaseSteady)
	}
	if snapshot.ActiveVUs != 10 {
		t.Errorf("ActiveVUs = %d, want 10", snapshot.ActiveVUs)
	}

	// Check latency stats exist
	if snapshot.Latency.Count != 100 {
		t.Errorf("Latency.Count = %d, want 100", snapshot.Latency.Count)
	}
}

func TestEngineWithConfig(t *testing.T) {
	config := EngineConfig{
		BucketInterval:   500 * time.Millisecond,
		MaxBuckets:       100,
		HistogramMin:     1,
		HistogramMax:     60000000, // 1 minute in microseconds
		HistogramSigFigs: 2,
	}

	engine := NewEngineWithConfig(config)
	if engine == nil {
		t.Fatal("NewEngineWithConfig() returned nil")
	}
	defer engine.Stop()

	// Record a latency
	engine.RecordLatency(10*time.Millisecond, "", true, 100)

	snapshot := engine.GetSnapshot()
	if snapshot.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", snapshot.TotalRequests)
	}
}
