package rate

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewLeakyBucket(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		expected float64
	}{
		{"positive rate", 100.0, 100.0},
		{"zero rate defaults to 1", 0.0, 1.0},
		{"negative rate defaults to 1", -10.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := NewLeakyBucket(tt.rate)
			if lb.GetRate() != tt.expected {
				t.Errorf("GetRate() = %v, want %v", lb.GetRate(), tt.expected)
			}
		})
	}
}

func TestLeakyBucket_Next_ImmediateFirst(t *testing.T) {
	lb := NewLeakyBucket(100.0)

	// First call should return now or very close to it
	now := time.Now()
	nextTime := lb.Next()

	diff := nextTime.Sub(now)
	if diff > 10*time.Millisecond {
		t.Errorf("First Next() should be immediate, got delay of %v", diff)
	}
}

func TestLeakyBucket_Next_CorrectRate(t *testing.T) {
	rate := 100.0 // 100 per second = 10ms apart
	lb := NewLeakyBucket(rate)

	// Consume first token
	_ = lb.Next()

	// Second call should be ~10ms in the future
	next := lb.Next()
	expectedDelay := time.Duration(float64(time.Second) / rate)

	now := time.Now()
	actualDelay := next.Sub(now)

	// Allow 5ms tolerance
	if actualDelay < expectedDelay-5*time.Millisecond || actualDelay > expectedDelay+5*time.Millisecond {
		t.Errorf("Delay between calls = %v, want ~%v", actualDelay, expectedDelay)
	}
}

func TestLeakyBucket_Wait_RespectsContext(t *testing.T) {
	lb := NewLeakyBucket(1.0) // 1 per second = slow

	// Consume first token
	_ = lb.Next()

	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := lb.Wait(ctx)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("Wait() error = %v, want DeadlineExceeded", err)
	}

	// Should have cancelled quickly, not waited full second
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait() took %v, should have cancelled quickly", elapsed)
	}
}

func TestLeakyBucket_SetRate_NoAccumulation(t *testing.T) {
	lb := NewLeakyBucket(1000.0) // High rate

	// Consume some iterations quickly
	for i := 0; i < 5; i++ {
		_ = lb.Next()
	}

	// Change to low rate
	lb.SetRate(1.0)

	// Next call should NOT burst - should wait ~1s
	next := lb.Next()
	now := time.Now()
	delay := next.Sub(now)

	// Should be close to 1s, not immediate
	if delay < 500*time.Millisecond {
		t.Errorf("After SetRate, delay = %v, should be ~1s (no burst)", delay)
	}
}

func TestLeakyBucket_SetRate_UpdatesCorrectly(t *testing.T) {
	lb := NewLeakyBucket(100.0)

	if lb.GetRate() != 100.0 {
		t.Errorf("Initial rate = %v, want 100.0", lb.GetRate())
	}

	lb.SetRate(200.0)
	if lb.GetRate() != 200.0 {
		t.Errorf("After SetRate(200), rate = %v, want 200.0", lb.GetRate())
	}

	lb.SetRate(0) // Should default to 1.0
	if lb.GetRate() != 1.0 {
		t.Errorf("After SetRate(0), rate = %v, want 1.0", lb.GetRate())
	}
}

func TestLeakyBucket_ConcurrentAccess(t *testing.T) {
	lb := NewLeakyBucket(10000.0) // High rate for fast test

	var wg sync.WaitGroup
	numGoroutines := 10
	callsPerGoroutine := 100

	ctx := context.Background()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				_ = lb.Wait(ctx)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent test timed out")
	}

	stats := lb.Stats()
	expectedIterations := int64(numGoroutines * callsPerGoroutine)
	if stats.TotalIterations != expectedIterations {
		t.Errorf("TotalIterations = %d, want %d", stats.TotalIterations, expectedIterations)
	}
}

func TestLeakyBucket_Stats(t *testing.T) {
	lb := NewLeakyBucket(100.0)

	// Initial stats
	stats := lb.Stats()
	if stats.Rate != 100.0 {
		t.Errorf("Stats.Rate = %v, want 100.0", stats.Rate)
	}
	if stats.TotalIterations != 0 {
		t.Errorf("Stats.TotalIterations = %d, want 0", stats.TotalIterations)
	}

	// After some iterations
	for i := 0; i < 5; i++ {
		_ = lb.Next()
	}

	stats = lb.Stats()
	if stats.TotalIterations != 5 {
		t.Errorf("After 5 Next(), TotalIterations = %d, want 5", stats.TotalIterations)
	}
}

func TestLeakyBucket_Reset(t *testing.T) {
	lb := NewLeakyBucket(100.0)

	// Generate some iterations
	for i := 0; i < 10; i++ {
		_ = lb.Next()
	}

	stats := lb.Stats()
	if stats.TotalIterations != 10 {
		t.Errorf("Before reset, TotalIterations = %d, want 10", stats.TotalIterations)
	}

	lb.Reset()

	stats = lb.Stats()
	if stats.TotalIterations != 0 {
		t.Errorf("After reset, TotalIterations = %d, want 0", stats.TotalIterations)
	}
}

func TestLeakyBucketWithBurst(t *testing.T) {
	lb := NewLeakyBucketWithBurst(100.0, 5.0) // Allow 5 iteration burst

	if lb.GetMaxBurst() != 5.0 {
		t.Errorf("GetMaxBurst() = %v, want 5.0", lb.GetMaxBurst())
	}

	lb.SetMaxBurst(10.0)
	if lb.GetMaxBurst() != 10.0 {
		t.Errorf("After SetMaxBurst(10), GetMaxBurst() = %v, want 10.0", lb.GetMaxBurst())
	}
}
