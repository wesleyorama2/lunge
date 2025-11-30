// Package executor provides load generation strategies for performance testing.
//
// Executors control HOW load is generated - whether by managing a pool
// of virtual users (VUs) or by controlling iteration rates. Each executor
// implements a different strategy suitable for different testing scenarios.
//
// # Available Executors
//
//   - constant-vus: Runs a fixed number of VUs for a duration
//   - ramping-vus: Ramps VU count up and down according to stages
//   - constant-arrival-rate: Maintains a fixed iteration rate (RPS)
//   - ramping-arrival-rate: Ramps iteration rate up and down
//
// # Executor Selection Guide
//
// VU-based (constant-vus, ramping-vus):
//   - Simulates a fixed number of concurrent users
//   - Throughput varies based on response time
//   - Good for capacity testing and user simulation
//
// Arrival-rate (constant-arrival-rate, ramping-arrival-rate):
//   - Maintains fixed throughput regardless of response time
//   - VU count scales automatically to meet target rate
//   - Good for SLA testing and stress testing
//
// # Example: constant-vus
//
//	scenario:
//	  executor: constant-vus
//	  vus: 10
//	  duration: 5m
//
// # Example: ramping-vus
//
//	scenario:
//	  executor: ramping-vus
//	  stages:
//	    - duration: 1m
//	      target: 5    # Ramp up to 5 VUs
//	    - duration: 3m
//	      target: 20   # Ramp up to 20 VUs
//	    - duration: 1m
//	      target: 0    # Ramp down to 0
//
// # Example: constant-arrival-rate
//
//	scenario:
//	  executor: constant-arrival-rate
//	  rate: 100           # 100 iterations per second
//	  duration: 5m
//	  preAllocatedVUs: 10 # Start with 10 VUs
//	  maxVUs: 50          # Scale up to 50 VUs if needed
//
// # Example: ramping-arrival-rate
//
//	scenario:
//	  executor: ramping-arrival-rate
//	  preAllocatedVUs: 10
//	  maxVUs: 100
//	  stages:
//	    - duration: 1m
//	      target: 50   # Ramp up to 50 RPS
//	    - duration: 3m
//	      target: 200  # Ramp up to 200 RPS
//	    - duration: 1m
//	      target: 0    # Ramp down to 0
package executor
