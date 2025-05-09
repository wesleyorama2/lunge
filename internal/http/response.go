package http

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// TimingInfo stores detailed timing information for an HTTP request
type TimingInfo struct {
	// Time when the request started
	StartTime time.Time

	// DNS lookup time
	DNSLookupTime time.Duration

	// TCP connection time
	TCPConnectTime time.Duration

	// TLS handshake time (for HTTPS)
	TLSHandshakeTime time.Duration

	// Time to first byte (TTFB)
	TimeToFirstByte time.Duration

	// Content transfer time
	ContentTransferTime time.Duration

	// Total response time
	TotalTime time.Duration
}

// Response represents an HTTP response
type Response struct {
	StatusCode   int
	Status       string
	Headers      http.Header
	Body         io.ReadCloser
	ResponseTime time.Duration
	Timing       TimingInfo
	rawBody      []byte
	parsed       bool
}

// GetBody returns the response body as a byte array
func (r *Response) GetBody() ([]byte, error) {
	if r.parsed {
		return r.rawBody, nil
	}

	// Read the body
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Store the raw body
	r.rawBody = body
	r.parsed = true

	return body, nil
}

// GetBodyAsString returns the response body as a string
func (r *Response) GetBodyAsString() (string, error) {
	body, err := r.GetBody()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// GetBodyAsJSON unmarshals the response body into the provided interface
func (r *Response) GetBodyAsJSON(v interface{}) error {
	body, err := r.GetBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// GetHeader returns the value of the specified header
func (r *Response) GetHeader(key string) string {
	return r.Headers.Get(key)
}

// IsSuccess returns true if the response status code is in the 2xx range
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsRedirect returns true if the response status code is in the 3xx range
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError returns true if the response status code is in the 4xx range
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response status code is in the 5xx range
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// GetResponseTimeMillis returns the response time in milliseconds
func (r *Response) GetResponseTimeMillis() int64 {
	return r.ResponseTime.Milliseconds()
}

// GetDNSLookupTimeMillis returns the DNS lookup time in milliseconds
func (r *Response) GetDNSLookupTimeMillis() int64 {
	return r.Timing.DNSLookupTime.Milliseconds()
}

// GetTCPConnectTimeMillis returns the TCP connection time in milliseconds
func (r *Response) GetTCPConnectTimeMillis() int64 {
	return r.Timing.TCPConnectTime.Milliseconds()
}

// GetTLSHandshakeTimeMillis returns the TLS handshake time in milliseconds
func (r *Response) GetTLSHandshakeTimeMillis() int64 {
	return r.Timing.TLSHandshakeTime.Milliseconds()
}

// GetTimeToFirstByteMillis returns the time to first byte in milliseconds
func (r *Response) GetTimeToFirstByteMillis() int64 {
	return r.Timing.TimeToFirstByte.Milliseconds()
}

// GetContentTransferTimeMillis returns the content transfer time in milliseconds
func (r *Response) GetContentTransferTimeMillis() int64 {
	return r.Timing.ContentTransferTime.Milliseconds()
}

// GetTotalTimeMillis returns the total time in milliseconds
func (r *Response) GetTotalTimeMillis() int64 {
	return r.Timing.TotalTime.Milliseconds()
}
