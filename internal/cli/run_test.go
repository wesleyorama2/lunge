package cli

import (
	"testing"
)

func TestIsAbsoluteURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "HTTP URL",
			url:      "http://example.com/path",
			expected: true,
		},
		{
			name:     "HTTPS URL",
			url:      "https://example.com/path",
			expected: true,
		},
		{
			name:     "Path starting with slash",
			url:      "/path",
			expected: false, // Changed from true to false after fixing the function
		},
		{
			name:     "Relative path",
			url:      "path",
			expected: false,
		},
		{
			name:     "Empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "Short string with h",
			url:      "h",
			expected: false,
		},
		{
			name:     "Short string with http",
			url:      "http",
			expected: false,
		},
		{
			name:     "Short string with https",
			url:      "https",
			expected: false,
		},
		{
			name:     "Incomplete HTTP URL",
			url:      "http:/",
			expected: false,
		},
		{
			name:     "Incomplete HTTPS URL",
			url:      "https:/",
			expected: false,
		},
		{
			name:     "URL with different protocol",
			url:      "ftp://example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAbsoluteURL(tt.url)
			if result != tt.expected {
				t.Errorf("isAbsoluteURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}
