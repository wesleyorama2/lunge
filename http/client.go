package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

// Client represents an HTTP client with customizable options.
// Client is safe for concurrent use by multiple goroutines.
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// NewClient creates a new HTTP client with the given options.
//
// Example:
//
//	client := http.NewClient(
//	    http.WithBaseURL("https://api.example.com"),
//	    http.WithTimeout(30*time.Second),
//	    http.WithHeader("Authorization", "Bearer token"),
//	)
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

// WithBaseURL sets the base URL for all requests made by this client.
// The base URL is prepended to the path specified in each Request.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithTimeout sets the timeout for all requests made by this client.
// The default timeout is 30 seconds.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHeader adds a default header to all requests made by this client.
// Headers set on individual requests will override these defaults.
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithHTTPClient sets a custom *http.Client for this client.
// Use this for advanced configuration like custom transports or TLS settings.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithInsecureSkipVerify disables TLS certificate verification.
// WARNING: This should only be used for testing purposes.
func WithInsecureSkipVerify() ClientOption {
	return func(c *Client) {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		c.httpClient.Transport = transport
	}
}

// Do executes an HTTP request and returns the response with detailed timing information.
// The request is built with the client's base URL and headers, and the provided Request
// configuration is applied on top.
//
// Example:
//
//	req := http.NewRequest("GET", "/users").
//	    WithQueryParam("limit", "10")
//
//	resp, err := client.Do(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Status: %d, TTFB: %v\n", resp.StatusCode, resp.Timing.TimeToFirstByte)
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	// Build the HTTP request
	httpReq, err := req.Build(c.baseURL)
	if err != nil {
		return nil, err
	}

	// Add client headers (request headers can override these)
	for key, value := range c.headers {
		if httpReq.Header.Get(key) == "" {
			httpReq.Header.Set(key, value)
		}
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
	bodyBytes, _ := io.ReadAll(httpResp.Body)
	httpResp.Body.Close()

	// Calculate content transfer time - this is the time it took to read the body
	timing.ContentTransferTime = time.Since(contentTransferStart)

	// Create a new body reader from the bytes we read
	bodyReader := io.NopCloser(bytes.NewReader(bodyBytes))

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

// Get is a convenience method for making GET requests.
func (c *Client) Get(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, NewRequest("GET", path))
}

// Post is a convenience method for making POST requests with a body.
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, NewRequest("POST", path).WithBody(body))
}

// Put is a convenience method for making PUT requests with a body.
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, NewRequest("PUT", path).WithBody(body))
}

// Delete is a convenience method for making DELETE requests.
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, NewRequest("DELETE", path))
}

// Patch is a convenience method for making PATCH requests with a body.
func (c *Client) Patch(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, NewRequest("PATCH", path).WithBody(body))
}
