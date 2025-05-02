package http

import (
	"testing"
)

func TestRequest_Build(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		baseURL        string
		headers        map[string]string
		queryParams    map[string]string
		body           interface{}
		expectedURL    string
		expectedMethod string
	}{
		{
			name:           "Simple GET request",
			method:         "GET",
			path:           "/users",
			baseURL:        "https://api.example.com",
			headers:        map[string]string{"Accept": "application/json"},
			expectedURL:    "https://api.example.com/users",
			expectedMethod: "GET",
		},
		{
			name:           "Request with query parameters",
			method:         "GET",
			path:           "/users",
			baseURL:        "https://api.example.com",
			queryParams:    map[string]string{"page": "1", "limit": "10"},
			expectedURL:    "https://api.example.com/users?limit=10&page=1",
			expectedMethod: "GET",
		},
		{
			name:           "Request with path and trailing slash in base URL",
			method:         "GET",
			path:           "/users",
			baseURL:        "https://api.example.com/",
			expectedURL:    "https://api.example.com/users",
			expectedMethod: "GET",
		},
		{
			name:           "Request with leading slash in path",
			method:         "GET",
			path:           "/users",
			baseURL:        "https://api.example.com",
			expectedURL:    "https://api.example.com/users",
			expectedMethod: "GET",
		},
		{
			name:           "POST request with body",
			method:         "POST",
			path:           "/users",
			baseURL:        "https://api.example.com",
			body:           map[string]string{"name": "John", "email": "john@example.com"},
			expectedURL:    "https://api.example.com/users",
			expectedMethod: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewRequest(tt.method, tt.path)

			// Add headers
			for key, value := range tt.headers {
				req.WithHeader(key, value)
			}

			// Add query parameters
			for key, value := range tt.queryParams {
				req.WithQueryParam(key, value)
			}

			// Add body
			if tt.body != nil {
				req.WithBody(tt.body)
			}

			// Build request
			httpReq, err := req.Build(tt.baseURL)
			if err != nil {
				t.Fatalf("Error building request: %v", err)
			}

			// Check method
			if httpReq.Method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, httpReq.Method)
			}

			// Check URL
			if httpReq.URL.String() != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, httpReq.URL.String())
			}

			// Check headers
			for key, value := range tt.headers {
				if httpReq.Header.Get(key) != value {
					t.Errorf("Expected header %s: %s, got %s", key, value, httpReq.Header.Get(key))
				}
			}

			// Check body for POST requests
			if tt.body != nil && tt.method == "POST" {
				// Check Content-Type header
				if httpReq.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", httpReq.Header.Get("Content-Type"))
				}

				// Check body is not nil
				if httpReq.Body == nil {
					t.Errorf("Expected body, got nil")
				}
			}
		})
	}
}

func TestRequest_WithMethods(t *testing.T) {
	// Test WithHeader
	req := NewRequest("GET", "/test")
	req.WithHeader("X-Test", "test-value")
	if req.Headers["X-Test"] != "test-value" {
		t.Errorf("Expected header X-Test: test-value, got %s", req.Headers["X-Test"])
	}

	// Test WithQueryParam
	req = NewRequest("GET", "/test")
	req.WithQueryParam("param", "value")
	if req.QueryParams.Get("param") != "value" {
		t.Errorf("Expected query param param=value, got %s", req.QueryParams.Get("param"))
	}

	// Test WithQueryParams
	req = NewRequest("GET", "/test")
	params := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}
	req.WithQueryParams(params)
	if req.QueryParams.Get("param1") != "value1" || req.QueryParams.Get("param2") != "value2" {
		t.Errorf("Expected query params param1=value1&param2=value2, got %s", req.QueryParams.Encode())
	}

	// Test WithBody
	req = NewRequest("POST", "/test")
	body := map[string]string{"name": "John"}
	req.WithBody(body)

	// Check that body was set (can't directly compare maps)
	if req.Body == nil {
		t.Errorf("Expected body to be set, got nil")
	}

	// Check body type
	switch req.Body.(type) {
	case map[string]string:
		bodyMap := req.Body.(map[string]string)
		if bodyMap["name"] != "John" {
			t.Errorf("Expected body[\"name\"] = \"John\", got %v", bodyMap["name"])
		}
	default:
		t.Errorf("Expected body type map[string]string, got %T", req.Body)
	}
}
