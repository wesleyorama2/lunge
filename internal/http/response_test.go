package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestResponse_GetBody(t *testing.T) {
	// Create a response with a body
	body := `{"message":"success"}`
	resp := &Response{
		StatusCode:   200,
		Status:       "200 OK",
		Headers:      make(http.Header),
		Body:         ioutil.NopCloser(strings.NewReader(body)),
		ResponseTime: 100 * time.Millisecond,
	}

	// Get body
	bodyBytes, err := resp.GetBody()
	if err != nil {
		t.Fatalf("Error getting body: %v", err)
	}

	// Check body
	if string(bodyBytes) != body {
		t.Errorf("Expected body %s, got %s", body, string(bodyBytes))
	}

	// Check that body is cached
	if !resp.parsed || string(resp.rawBody) != body {
		t.Errorf("Body not cached correctly")
	}

	// Get body again (should use cached value)
	bodyBytes2, err := resp.GetBody()
	if err != nil {
		t.Fatalf("Error getting body second time: %v", err)
	}

	// Check body
	if string(bodyBytes2) != body {
		t.Errorf("Expected body %s, got %s", body, string(bodyBytes2))
	}
}

func TestResponse_GetBodyAsString(t *testing.T) {
	// Create a response with a body
	body := `{"message":"success"}`
	resp := &Response{
		StatusCode:   200,
		Status:       "200 OK",
		Headers:      make(http.Header),
		Body:         ioutil.NopCloser(strings.NewReader(body)),
		ResponseTime: 100 * time.Millisecond,
	}

	// Get body as string
	bodyStr, err := resp.GetBodyAsString()
	if err != nil {
		t.Fatalf("Error getting body as string: %v", err)
	}

	// Check body
	if bodyStr != body {
		t.Errorf("Expected body %s, got %s", body, bodyStr)
	}
}

func TestResponse_GetBodyAsJSON(t *testing.T) {
	// Create a response with a JSON body
	body := `{"message":"success","code":200}`
	resp := &Response{
		StatusCode:   200,
		Status:       "200 OK",
		Headers:      make(http.Header),
		Body:         ioutil.NopCloser(strings.NewReader(body)),
		ResponseTime: 100 * time.Millisecond,
	}

	// Define a struct to unmarshal into
	var result struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	// Get body as JSON
	err := resp.GetBodyAsJSON(&result)
	if err != nil {
		t.Fatalf("Error getting body as JSON: %v", err)
	}

	// Check unmarshaled values
	if result.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", result.Message)
	}
	if result.Code != 200 {
		t.Errorf("Expected code 200, got %d", result.Code)
	}
}

func TestResponse_GetHeader(t *testing.T) {
	// Create headers
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Test", "test-value")

	// Create response
	resp := &Response{
		StatusCode:   200,
		Status:       "200 OK",
		Headers:      headers,
		Body:         ioutil.NopCloser(bytes.NewReader([]byte{})),
		ResponseTime: 100 * time.Millisecond,
	}

	// Check headers
	if resp.GetHeader("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", resp.GetHeader("Content-Type"))
	}
	if resp.GetHeader("X-Test") != "test-value" {
		t.Errorf("Expected X-Test: test-value, got %s", resp.GetHeader("X-Test"))
	}
	if resp.GetHeader("Non-Existent") != "" {
		t.Errorf("Expected empty string for non-existent header, got %s", resp.GetHeader("Non-Existent"))
	}
}

func TestResponse_StatusMethods(t *testing.T) {
	tests := []struct {
		statusCode    int
		isSuccess     bool
		isRedirect    bool
		isClientError bool
		isServerError bool
	}{
		{200, true, false, false, false},
		{201, true, false, false, false},
		{301, false, true, false, false},
		{302, false, true, false, false},
		{400, false, false, true, false},
		{404, false, false, true, false},
		{500, false, false, false, true},
		{503, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.statusCode), func(t *testing.T) {
			resp := &Response{StatusCode: tt.statusCode}

			if resp.IsSuccess() != tt.isSuccess {
				t.Errorf("IsSuccess() = %v, want %v", resp.IsSuccess(), tt.isSuccess)
			}
			if resp.IsRedirect() != tt.isRedirect {
				t.Errorf("IsRedirect() = %v, want %v", resp.IsRedirect(), tt.isRedirect)
			}
			if resp.IsClientError() != tt.isClientError {
				t.Errorf("IsClientError() = %v, want %v", resp.IsClientError(), tt.isClientError)
			}
			if resp.IsServerError() != tt.isServerError {
				t.Errorf("IsServerError() = %v, want %v", resp.IsServerError(), tt.isServerError)
			}
		})
	}
}

func TestResponse_GetResponseTimeMillis(t *testing.T) {
	// Create response with response time
	resp := &Response{
		ResponseTime: 123 * time.Millisecond,
	}

	// Check response time
	if resp.GetResponseTimeMillis() != 123 {
		t.Errorf("Expected response time 123ms, got %dms", resp.GetResponseTimeMillis())
	}
}
