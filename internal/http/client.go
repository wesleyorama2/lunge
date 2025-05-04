package http

import (
	"context"
	"crypto/tls"
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
	var dnsDone, connectDone, tlsHandshakeDone bool
	var firstByteTime time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			timing.DNSLookupTime = time.Since(dnsStart)
			dnsDone = true
		},
		ConnectStart: func(network, addr string) {
			if dnsDone {
				connectStart = time.Now()
			}
		},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				timing.TCPConnectTime = time.Since(connectStart)
				connectDone = true
			}
		},
		TLSHandshakeStart: func() {
			if connectDone {
				tlsHandshakeStart = time.Now()
			}
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if err == nil {
				timing.TLSHandshakeTime = time.Since(tlsHandshakeStart)
				tlsHandshakeDone = true
			}
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Now()
			if tlsHandshakeDone {
				timing.TimeToFirstByte = firstByteTime.Sub(tlsHandshakeStart.Add(timing.TLSHandshakeTime))
			} else if connectDone {
				timing.TimeToFirstByte = firstByteTime.Sub(connectStart.Add(timing.TCPConnectTime))
			} else if dnsDone {
				timing.TimeToFirstByte = firstByteTime.Sub(dnsStart.Add(timing.DNSLookupTime))
			} else {
				timing.TimeToFirstByte = firstByteTime.Sub(timing.StartTime)
			}
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

	// Calculate content transfer time (total time minus time to first byte)
	if !firstByteTime.IsZero() {
		timing.ContentTransferTime = timing.TotalTime - timing.StartTime.Sub(firstByteTime)
	}

	// Create response
	resp := &Response{
		StatusCode:   httpResp.StatusCode,
		Status:       httpResp.Status,
		Headers:      httpResp.Header,
		Body:         httpResp.Body,
		ResponseTime: timing.TotalTime, // For backward compatibility
		Timing:       timing,
	}

	return resp, nil
}
