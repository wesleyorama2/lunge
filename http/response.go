package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// TimingInfo stores detailed timing information for an HTTP request.
// All durations represent the time spent in each phase of the request.
type TimingInfo struct {
	// StartTime is when the request started
	StartTime time.Time

	// DNSLookupTime is the time spent looking up the DNS address
	DNSLookupTime time.Duration

	// TCPConnectTime is the time spent establishing a TCP connection
	TCPConnectTime time.Duration

	// TLSHandshakeTime is the time spent performing the TLS handshake (for HTTPS)
	TLSHandshakeTime time.Duration

	// TimeToFirstByte (TTFB) is the time from connection established to receiving the first byte
	TimeToFirstByte time.Duration

	// ContentTransferTime is the time spent reading the response body
	ContentTransferTime time.Duration

	// TotalTime is the total time from request start to completion
	TotalTime time.Duration
}

// Response represents an HTTP response with timing information.
type Response struct {
	// StatusCode is the HTTP status code (e.g., 200, 404, 500)
	StatusCode int

	// Status is the HTTP status string (e.g., "200 OK")
	Status string

	// Headers contains the response headers
	Headers http.Header

	// Body is the response body as an io.ReadCloser
	Body io.ReadCloser

	// ResponseTime is the total response time (for backward compatibility)
	// Prefer using Timing.TotalTime for more accurate measurements
	ResponseTime time.Duration

	// Timing contains detailed timing information
	Timing TimingInfo

	// Internal fields for caching
	rawBody []byte
	parsed  bool
}

// GetBody returns the response body as a byte array.
// The body is cached, so this method can be called multiple times.
func (r *Response) GetBody() ([]byte, error) {
	if r.parsed {
		return r.rawBody, nil
	}

	// Read the body
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Store the raw body
	r.rawBody = body
	r.parsed = true

	return body, nil
}

// GetBodyAsString returns the response body as a string.
func (r *Response) GetBodyAsString() (string, error) {
	body, err := r.GetBody()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// GetBodyAsJSON unmarshals the response body into the provided interface.
//
// Example:
//
//	var users []User
//	if err := resp.GetBodyAsJSON(&users); err != nil {
//	    log.Fatal(err)
//	}
func (r *Response) GetBodyAsJSON(v interface{}) error {
	body, err := r.GetBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// GetHeader returns the value of the specified header.
// Returns an empty string if the header is not present.
func (r *Response) GetHeader(key string) string {
	return r.Headers.Get(key)
}

// IsSuccess returns true if the response status code is in the 2xx range.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsRedirect returns true if the response status code is in the 3xx range.
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError returns true if the response status code is in the 4xx range.
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response status code is in the 5xx range.
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// IsError returns true if the response status code indicates an error (4xx or 5xx).
func (r *Response) IsError() bool {
	return r.IsClientError() || r.IsServerError()
}

// GetResponseTimeMillis returns the response time in milliseconds.
func (r *Response) GetResponseTimeMillis() int64 {
	return r.ResponseTime.Milliseconds()
}

// GetDNSLookupTimeMillis returns the DNS lookup time in milliseconds.
func (r *Response) GetDNSLookupTimeMillis() int64 {
	return r.Timing.DNSLookupTime.Milliseconds()
}

// GetTCPConnectTimeMillis returns the TCP connection time in milliseconds.
func (r *Response) GetTCPConnectTimeMillis() int64 {
	return r.Timing.TCPConnectTime.Milliseconds()
}

// GetTLSHandshakeTimeMillis returns the TLS handshake time in milliseconds.
func (r *Response) GetTLSHandshakeTimeMillis() int64 {
	return r.Timing.TLSHandshakeTime.Milliseconds()
}

// GetTimeToFirstByteMillis returns the time to first byte in milliseconds.
func (r *Response) GetTimeToFirstByteMillis() int64 {
	return r.Timing.TimeToFirstByte.Milliseconds()
}

// GetContentTransferTimeMillis returns the content transfer time in milliseconds.
func (r *Response) GetContentTransferTimeMillis() int64 {
	return r.Timing.ContentTransferTime.Milliseconds()
}

// GetTotalTimeMillis returns the total time in milliseconds.
func (r *Response) GetTotalTimeMillis() int64 {
	return r.Timing.TotalTime.Milliseconds()
}
