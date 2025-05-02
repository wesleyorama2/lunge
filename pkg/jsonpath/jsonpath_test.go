package jsonpath

import (
	"testing"
)

func TestExtract(t *testing.T) {
	// Test JSON
	json := `{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com",
		"address": {
			"street": "123 Main St",
			"city": "Anytown",
			"zipcode": "12345"
		},
		"phones": [
			{
				"type": "home",
				"number": "555-1234"
			},
			{
				"type": "work",
				"number": "555-5678"
			}
		],
		"active": true,
		"scores": [10, 20, 30, 40],
		"metadata": null
	}`

	tests := []struct {
		name          string
		path          string
		expected      string
		expectedError bool
	}{
		{
			name:          "Root path",
			path:          "$",
			expected:      json,
			expectedError: false,
		},
		{
			name:          "Simple property",
			path:          "$.name",
			expected:      "John Doe",
			expectedError: false,
		},
		{
			name:          "Numeric property",
			path:          "$.age",
			expected:      "30",
			expectedError: false,
		},
		{
			name:          "Boolean property",
			path:          "$.active",
			expected:      "true",
			expectedError: false,
		},
		{
			name:          "Nested property",
			path:          "$.address.city",
			expected:      "Anytown",
			expectedError: false,
		},
		{
			name:          "Array element",
			path:          "$.scores[1]",
			expected:      "20",
			expectedError: false,
		},
		{
			name:          "Object in array",
			path:          "$.phones[0].number",
			expected:      "555-1234",
			expectedError: false,
		},
		{
			name:          "Last array element",
			path:          "$.scores[3]",
			expected:      "40",
			expectedError: false,
		},
		{
			name:          "Null value",
			path:          "$.metadata",
			expected:      "null",
			expectedError: false,
		},
		{
			name:          "Non-existent property",
			path:          "$.nonexistent",
			expected:      "",
			expectedError: true,
		},
		{
			name:          "Non-existent nested property",
			path:          "$.address.country",
			expected:      "",
			expectedError: true,
		},
		{
			name:          "Array index out of bounds",
			path:          "$.scores[10]",
			expected:      "",
			expectedError: true,
		},
		{
			name:          "Empty path",
			path:          "",
			expected:      "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Extract(json, tt.path)

			// Check error
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			// Check result if no error expected
			if !tt.expectedError && result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}

	// Test empty JSON
	_, err := Extract("", "$.name")
	if err == nil {
		t.Errorf("Expected error for empty JSON, got nil")
	}
}

func TestExtractMultiple(t *testing.T) {
	// Test JSON
	json := `{
		"user": {
			"name": "John Doe",
			"email": "john@example.com",
			"address": {
				"city": "Anytown"
			}
		},
		"status": "active",
		"items": [
			{"id": 1, "name": "Item 1"},
			{"id": 2, "name": "Item 2"}
		]
	}`

	// Test cases
	tests := []struct {
		name          string
		paths         map[string]string
		expected      map[string]string
		expectedError bool
	}{
		{
			name: "Multiple valid paths",
			paths: map[string]string{
				"name":   "$.user.name",
				"email":  "$.user.email",
				"status": "$.status",
				"item":   "$.items[0].name",
			},
			expected: map[string]string{
				"name":   "John Doe",
				"email":  "john@example.com",
				"status": "active",
				"item":   "Item 1",
			},
			expectedError: false,
		},
		{
			name: "Some invalid paths",
			paths: map[string]string{
				"name":    "$.user.name",
				"country": "$.user.address.country", // This path doesn't exist
			},
			expected: map[string]string{
				"name": "John Doe",
				// country will be missing
			},
			expectedError: true,
		},
		{
			name:          "Empty paths",
			paths:         map[string]string{},
			expected:      nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExtractMultiple(json, tt.paths)

			// Check error
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			// Check results
			for name, expected := range tt.expected {
				if result, ok := results[name]; ok {
					if result != expected {
						t.Errorf("Expected %s=%q, got %q", name, expected, result)
					}
				} else {
					t.Errorf("Missing expected result for %s", name)
				}
			}
		})
	}

	// Test empty JSON
	_, err := ExtractMultiple("", map[string]string{"name": "$.name"})
	if err == nil {
		t.Errorf("Expected error for empty JSON, got nil")
	}
}

func TestConvertToGjsonPath(t *testing.T) {
	tests := []struct {
		jsonPath  string
		gjsonPath string
	}{
		{"$.name", "name"},
		{"$['name']", "name"},
		{"$.user.name", "user.name"},
		{"$.items[0]", "items.0"},
		{"$.items[0].name", "items.0.name"},
		{"$.deeply.nested[0].array[1].value", "deeply.nested.0.array.1.value"},
		{"$", "@this"},
		{"$[0]", "0"},
		{"$[0].name", "0.name"},
	}

	for _, tt := range tests {
		t.Run(tt.jsonPath, func(t *testing.T) {
			result := convertToGjsonPath(tt.jsonPath)
			if result != tt.gjsonPath {
				t.Errorf("convertToGjsonPath(%q) = %q, want %q", tt.jsonPath, result, tt.gjsonPath)
			}
		})
	}
}
