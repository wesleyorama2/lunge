package cli

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedBase string
		expectedPath string
	}{
		{
			name:         "Simple URL",
			url:          "https://example.com/path",
			expectedBase: "https://example.com",
			expectedPath: "/path",
		},
		{
			name:         "URL with query parameters",
			url:          "https://example.com/path?param=value",
			expectedBase: "https://example.com",
			expectedPath: "/path?param=value",
		},
		{
			name:         "URL with fragment",
			url:          "https://example.com/path#fragment",
			expectedBase: "https://example.com",
			expectedPath: "/path#fragment",
		},
		{
			name:         "URL without scheme",
			url:          "example.com/path",
			expectedBase: "http://example.com",
			expectedPath: "/path",
		},
		{
			name:         "URL without path",
			url:          "https://example.com",
			expectedBase: "https://example.com",
			expectedPath: "/",
		},
		{
			name:         "Complex URL",
			url:          "https://api.example.com:8080/v1/users/123?filter=active#details",
			expectedBase: "https://api.example.com:8080",
			expectedPath: "/v1/users/123?filter=active#details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, path := parseURL(tt.url)
			if baseURL != tt.expectedBase {
				t.Errorf("parseURL() baseURL = %v, want %v", baseURL, tt.expectedBase)
			}
			if path != tt.expectedPath {
				t.Errorf("parseURL() path = %v, want %v", path, tt.expectedPath)
			}
		})
	}
}

// TestParseURL_EdgeCases tests additional edge cases for the parseURL function
func TestParseURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedBase string
		expectedPath string
	}{
		{
			name:         "URL with port",
			url:          "http://localhost:8080/api",
			expectedBase: "http://localhost:8080",
			expectedPath: "/api",
		},
		{
			name:         "URL with username and password",
			url:          "http://user:pass@example.com/secure",
			expectedBase: "http://user:pass@example.com",
			expectedPath: "/secure",
		},
		{
			name:         "URL with multiple query parameters",
			url:          "https://example.com/search?q=test&page=1&sort=desc",
			expectedBase: "https://example.com",
			expectedPath: "/search?q=test&page=1&sort=desc",
		},
		{
			name:         "URL with empty path",
			url:          "https://example.com",
			expectedBase: "https://example.com",
			expectedPath: "/",
		},
		{
			name:         "URL with trailing slash",
			url:          "https://example.com/api/",
			expectedBase: "https://example.com",
			expectedPath: "/api/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, path := parseURL(tt.url)
			if baseURL != tt.expectedBase {
				t.Errorf("parseURL() baseURL = %v, want %v", baseURL, tt.expectedBase)
			}
			if path != tt.expectedPath {
				t.Errorf("parseURL() path = %v, want %v", path, tt.expectedPath)
			}
		})
	}
}
