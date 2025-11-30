// Package config provides configuration types and parsing for performance tests.
//
// This package defines the YAML/JSON schema for performance test configurations
// and provides utilities for loading and validating them.
//
// # Configuration Schema
//
// Test configurations use YAML or JSON format:
//
//	name: "API Load Test"
//	description: "Load test for user API endpoints"
//
//	settings:
//	  baseUrl: "https://api.example.com"
//	  timeout: 30s
//	  headers:
//	    Authorization: "Bearer {{token}}"
//
//	variables:
//	  token: "your-auth-token"
//
//	scenarios:
//	  smoke:
//	    executor: constant-vus
//	    vus: 5
//	    duration: 30s
//	    requests:
//	      - method: GET
//	        url: "{{baseUrl}}/health"
//
//	  load:
//	    executor: ramping-vus
//	    stages:
//	      - duration: 1m
//	        target: 10
//	      - duration: 5m
//	        target: 50
//	      - duration: 1m
//	        target: 0
//	    requests:
//	      - method: GET
//	        url: "{{baseUrl}}/users"
//
//	thresholds:
//	  http_req_duration:
//	    - "p95 < 500ms"
//	    - "avg < 200ms"
//	  http_req_failed:
//	    - "rate < 0.01"
//
// # Loading Configuration
//
//	cfg, err := config.LoadConfig("test.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate configuration
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Supported Executors
//
//   - constant-vus: Fixed number of VUs for a duration
//   - ramping-vus: VU count ramps according to stages
//   - constant-arrival-rate: Fixed iteration rate
//   - ramping-arrival-rate: Iteration rate ramps according to stages
package config
