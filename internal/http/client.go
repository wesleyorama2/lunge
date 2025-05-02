package http

import (
	"context"
	"net/http"
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

// Do executes an HTTP request and returns the response
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

	// Add context
	httpReq = httpReq.WithContext(ctx)

	// Record start time for response time calculation
	startTime := time.Now()

	// Execute the request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// Calculate response time
	responseTime := time.Since(startTime)

	// Create response
	resp := &Response{
		StatusCode:   httpResp.StatusCode,
		Status:       httpResp.Status,
		Headers:      httpResp.Header,
		Body:         httpResp.Body,
		ResponseTime: responseTime,
	}

	return resp, nil
}
