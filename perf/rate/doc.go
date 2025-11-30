// Package rate provides rate limiting implementations for load testing.
//
// The primary implementation is the LeakyBucket, which provides smooth
// rate limiting suitable for load generation scenarios.
//
// # Leaky Bucket Algorithm
//
// Unlike token bucket which focuses on "how many tokens are available",
// leaky bucket focuses on "when should the next iteration execute".
// This approach provides smoother rate limiting without bursting issues
// during rate changes (ramp-up/down).
//
// # Basic Usage
//
//	limiter := rate.NewLeakyBucket(100.0) // 100 iterations per second
//
//	for {
//	    if err := limiter.Wait(ctx); err != nil {
//	        break // Context cancelled
//	    }
//	    // Execute iteration
//	}
//
// # Dynamic Rate Changes
//
// The leaky bucket supports smooth rate changes during execution,
// which is useful for ramping tests:
//
//	limiter := rate.NewLeakyBucket(10.0) // Start at 10 RPS
//
//	go func() {
//	    for i := 1; i <= 10; i++ {
//	        time.Sleep(time.Second)
//	        limiter.SetRate(float64(i * 10)) // Ramp up
//	    }
//	}()
//
// # Thread Safety
//
// All methods on LeakyBucket are safe for concurrent use from multiple goroutines.
package rate
