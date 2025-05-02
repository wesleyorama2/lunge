package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Request represents an HTTP request
type Request struct {
	Method      string
	Path        string
	QueryParams url.Values
	Headers     map[string]string
	Body        interface{}
}

// NewRequest creates a new HTTP request
func NewRequest(method, path string) *Request {
	return &Request{
		Method:      method,
		Path:        path,
		QueryParams: make(url.Values),
		Headers:     make(map[string]string),
	}
}

// WithHeader adds a header to the request
func (r *Request) WithHeader(key, value string) *Request {
	r.Headers[key] = value
	return r
}

// WithQueryParam adds a query parameter to the request
func (r *Request) WithQueryParam(key, value string) *Request {
	r.QueryParams.Add(key, value)
	return r
}

// WithQueryParams adds multiple query parameters to the request
func (r *Request) WithQueryParams(params map[string]string) *Request {
	for key, value := range params {
		r.QueryParams.Add(key, value)
	}
	return r
}

// WithBody sets the body of the request
func (r *Request) WithBody(body interface{}) *Request {
	r.Body = body
	return r
}

// Build constructs an http.Request from the Request
func (r *Request) Build(baseURL string) (*http.Request, error) {
	// Construct the URL
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Join the base URL path with the request path
	if reqURL.Path == "" {
		reqURL.Path = r.Path
	} else {
		reqURL.Path = strings.TrimRight(reqURL.Path, "/") + "/" + strings.TrimLeft(r.Path, "/")
	}

	// Add query parameters
	query := reqURL.Query()
	for key, values := range r.QueryParams {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	reqURL.RawQuery = query.Encode()

	// Prepare the body
	var bodyReader io.Reader
	if r.Body != nil {
		switch body := r.Body.(type) {
		case string:
			bodyReader = strings.NewReader(body)
		case []byte:
			bodyReader = bytes.NewReader(body)
		case io.Reader:
			bodyReader = body
		default:
			// Assume JSON for other types
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonBody)
			// Set Content-Type to application/json if not already set
			if _, ok := r.Headers["Content-Type"]; !ok {
				r.Headers["Content-Type"] = "application/json"
			}
		}
	}

	// Create the HTTP request
	req, err := http.NewRequest(r.Method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
