package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"time"
)

// Client represents an HTTP client with customizable options
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
	timeout    time.Duration
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// NewClient creates a new HTTP client with the given options
func NewClient(options ...ClientOption) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: make(map[string]string),
	}

	// Apply options
	for _, option := range options {
		option(client)
	}

	return client
}

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithTimeout sets the timeout for the client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHeader adds a header to the client
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// Do executes an HTTP request and returns the response with detailed timing information
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	// Build the HTTP request
	httpReq, err := req.Build(c.baseURL)
	if err != nil {
		return nil, err
	}

	// Add client headers
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	// Initialize timing info
	timing := TimingInfo{
		StartTime: time.Now(),
	}

	// Create a trace to capture detailed timing information
	var dnsStart, connectStart, tlsHandshakeStart time.Time
	var dnsEnd, connectEnd, tlsHandshakeEnd time.Time
	var dnsDone, connectDone bool
	var firstByteTime time.Time
	var lastPhaseEnd time.Time // Tracks the end time of the last completed phase

	// Initialize lastPhaseEnd to the start time
	lastPhaseEnd = timing.StartTime

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsEnd = time.Now()
			timing.DNSLookupTime = dnsEnd.Sub(dnsStart)
			dnsDone = true
			lastPhaseEnd = dnsEnd // Update last phase end time
		},
		ConnectStart: func(network, addr string) {
			if dnsDone {
				connectStart = time.Now()
			}
		},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				connectEnd = time.Now()
				timing.TCPConnectTime = connectEnd.Sub(connectStart)
				connectDone = true
				lastPhaseEnd = connectEnd // Update last phase end time
			}
		},
		TLSHandshakeStart: func() {
			if connectDone {
				tlsHandshakeStart = time.Now()
			}
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if err == nil {
				tlsHandshakeEnd = time.Now()
				timing.TLSHandshakeTime = tlsHandshakeEnd.Sub(tlsHandshakeStart)
				lastPhaseEnd = tlsHandshakeEnd // Update last phase end time
			}
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Now()
			// Calculate time to first byte from the end of the last phase
			timing.TimeToFirstByte = firstByteTime.Sub(lastPhaseEnd)
		},
	}

	// Add the trace to the request context
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(ctx, trace))

	// Execute the request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// Calculate total response time
	timing.TotalTime = time.Since(timing.StartTime)

	// Read and close the body
	contentTransferStart := time.Now()
	bodyBytes, _ := ioutil.ReadAll(httpResp.Body)
	httpResp.Body.Close()

	// Calculate content transfer time - this is the time it took to read the body
	timing.ContentTransferTime = time.Since(contentTransferStart)

	// Create a new body reader from the bytes we read
	bodyReader := ioutil.NopCloser(bytes.NewReader(bodyBytes))

	// Create response
	resp := &Response{
		StatusCode:   httpResp.StatusCode,
		Status:       httpResp.Status,
		Headers:      httpResp.Header,
		Body:         bodyReader,
		ResponseTime: time.Since(timing.StartTime), // For backward compatibility
		Timing:       timing,
		rawBody:      bodyBytes, // Store the raw body so we don't need to read it again
		parsed:       true,      // Mark as already parsed
	}

	return resp, nil
}
