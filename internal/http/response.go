package http

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// Response represents an HTTP response
type Response struct {
	StatusCode   int
	Status       string
	Headers      http.Header
	Body         io.ReadCloser
	ResponseTime time.Duration
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
