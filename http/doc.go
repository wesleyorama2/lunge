// Package http provides an HTTP client library with detailed timing metrics
// and a fluent builder pattern for requests.
//
// This package is designed for programmatic use and provides:
//   - A configurable HTTP client with functional options
//   - Detailed timing information (DNS, TCP, TLS, TTFB)
//   - A fluent request builder pattern
//   - Response parsing utilities
//
// Basic Usage:
//
//	client := http.NewClient(
//	    http.WithBaseURL("https://api.example.com"),
//	    http.WithTimeout(30*time.Second),
//	    http.WithHeader("Authorization", "Bearer token"),
//	)
//
//	req := http.NewRequest("GET", "/users").
//	    WithQueryParam("limit", "10")
//
//	resp, err := client.Do(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Status: %d\n", resp.StatusCode)
//	fmt.Printf("TTFB: %v\n", resp.Timing.TimeToFirstByte)
//
// Auth Token Example:
//
//	client := http.NewClient(
//	    http.WithBaseURL("https://auth.example.com"),
//	)
//
//	req := http.NewRequest("POST", "/oauth/token").
//	    WithHeader("Content-Type", "application/x-www-form-urlencoded").
//	    WithBody("grant_type=client_credentials&client_id=xxx&client_secret=yyy")
//
//	resp, err := client.Do(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	var tokenResp struct {
//	    AccessToken string `json:"access_token"`
//	    ExpiresIn   int    `json:"expires_in"`
//	}
//	if err := resp.GetBodyAsJSON(&tokenResp); err != nil {
//	    log.Fatal(err)
//	}
//
// Thread Safety:
//
// Client is safe for concurrent use. Multiple goroutines may invoke methods
// on a Client simultaneously.
package http
