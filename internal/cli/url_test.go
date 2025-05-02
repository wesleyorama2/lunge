package cli

import (
	"strings"
	"testing"
)

func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "Path without leading slash",
			baseURL:  "https://example.com",
			path:     "api/users",
			expected: "https://example.com/api/users",
		},
		{
			name:     "Path with leading slash",
			baseURL:  "https://example.com",
			path:     "/api/users",
			expected: "https://example.com/api/users",
		},
		{
			name:     "BaseURL with trailing slash, path without leading slash",
			baseURL:  "https://example.com/",
			path:     "api/users",
			expected: "https://example.com/api/users",
		},
		{
			name:     "BaseURL with trailing slash, path with leading slash",
			baseURL:  "https://example.com/",
			path:     "/api/users",
			expected: "https://example.com/api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string

			// This is the same logic used in the executeRequest function
			if strings.HasPrefix(tt.path, "/") {
				result = tt.baseURL + tt.path
			} else {
				result = tt.baseURL + "/" + tt.path
			}

			// Handle trailing slash in baseURL to avoid double slashes
			result = strings.Replace(result, "//", "/", -1)

			// Fix protocol after replacing slashes
			result = strings.Replace(result, ":/", "://", 1)

			if result != tt.expected {
				t.Errorf("URL construction = %v, want %v", result, tt.expected)
			}
		})
	}
}
