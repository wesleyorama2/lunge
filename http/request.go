package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Request represents an HTTP request with a fluent builder pattern.
// Use NewRequest to create a new Request and chain method calls to configure it.
type Request struct {
	Method      string
	Path        string
	QueryParams url.Values
	Headers     map[string]string
	Body        interface{}
}

// NewRequest creates a new HTTP request with the specified method and path.
//
// Example:
//
//	req := http.NewRequest("GET", "/users").
//	    WithQueryParam("limit", "10").
//	    WithHeader("Accept", "application/json")
func NewRequest(method, path string) *Request {
	return &Request{
		Method:      method,
		Path:        path,
		QueryParams: make(url.Values),
		Headers:     make(map[string]string),
	}
}

// WithHeader adds a header to the request.
// Returns the Request to allow method chaining.
func (r *Request) WithHeader(key, value string) *Request {
	r.Headers[key] = value
	return r
}

// WithHeaders adds multiple headers to the request.
// Returns the Request to allow method chaining.
func (r *Request) WithHeaders(headers map[string]string) *Request {
	for key, value := range headers {
		r.Headers[key] = value
	}
	return r
}

// WithQueryParam adds a query parameter to the request.
// Multiple values for the same key can be added by calling this method multiple times.
// Returns the Request to allow method chaining.
func (r *Request) WithQueryParam(key, value string) *Request {
	r.QueryParams.Add(key, value)
	return r
}

// WithQueryParams adds multiple query parameters to the request.
// Returns the Request to allow method chaining.
func (r *Request) WithQueryParams(params map[string]string) *Request {
	for key, value := range params {
		r.QueryParams.Add(key, value)
	}
	return r
}

// WithBody sets the body of the request.
// The body can be:
//   - string: sent as-is
//   - []byte: sent as-is
//   - io.Reader: read and sent
//   - any other type: marshaled as JSON (Content-Type is set to application/json if not already set)
//
// Returns the Request to allow method chaining.
func (r *Request) WithBody(body interface{}) *Request {
	r.Body = body
	return r
}

// WithJSON sets the body of the request as JSON and sets the Content-Type header.
// The value will be marshaled to JSON.
// Returns the Request to allow method chaining.
func (r *Request) WithJSON(v interface{}) *Request {
	r.Body = v
	r.Headers["Content-Type"] = "application/json"
	return r
}

// WithFormData sets the body of the request as URL-encoded form data.
// Sets the Content-Type header to application/x-www-form-urlencoded.
// Returns the Request to allow method chaining.
func (r *Request) WithFormData(data map[string]string) *Request {
	formValues := url.Values{}
	for key, value := range data {
		formValues.Set(key, value)
	}
	r.Body = formValues.Encode()
	r.Headers["Content-Type"] = "application/x-www-form-urlencoded"
	return r
}

// Build constructs an http.Request from the Request configuration.
// This is called internally by Client.Do but is exposed for advanced use cases.
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
