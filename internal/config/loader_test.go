package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"environments": {
			"dev": {
				"baseUrl": "https://api-dev.example.com",
				"variables": {
					"userId": "1"
				}
			},
			"prod": {
				"baseUrl": "https://api.example.com",
				"variables": {
					"userId": "2"
				}
			}
		},
		"requests": {
			"getUser": {
				"url": "/users/{{userId}}",
				"method": "GET",
				"headers": {
					"Accept": "application/json"
				}
			},
			"getPosts": {
				"url": "/posts",
				"method": "GET",
				"queryParams": {
					"userId": "{{userId}}"
				}
			}
		},
		"suites": {
			"userFlow": {
				"requests": ["getUser", "getPosts"],
				"variables": {
					"userId": "1"
				},
				"tests": [
					{
						"name": "User exists",
						"request": "getUser",
						"assertions": [
							{ "status": 200 },
							{ "path": "$.email", "exists": true }
						]
					}
				]
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Error creating test config file: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Error loading config: %v", err)
	}

	// Check environments
	if len(config.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(config.Environments))
	}

	devEnv, ok := config.Environments["dev"]
	if !ok {
		t.Errorf("Expected dev environment to exist")
	} else {
		if devEnv.BaseURL != "https://api-dev.example.com" {
			t.Errorf("Expected dev baseUrl to be https://api-dev.example.com, got %s", devEnv.BaseURL)
		}
		if devEnv.Vars["userId"] != "1" {
			t.Errorf("Expected dev userId to be 1, got %s", devEnv.Vars["userId"])
		}
	}

	// Check requests
	if len(config.Requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(config.Requests))
	}

	getUserReq, ok := config.Requests["getUser"]
	if !ok {
		t.Errorf("Expected getUser request to exist")
	} else {
		if getUserReq.URL != "/users/{{userId}}" {
			t.Errorf("Expected getUser URL to be /users/{{userId}}, got %s", getUserReq.URL)
		}
		if getUserReq.Method != "GET" {
			t.Errorf("Expected getUser method to be GET, got %s", getUserReq.Method)
		}
	}

	// Check suites
	if len(config.Suites) != 1 {
		t.Errorf("Expected 1 suite, got %d", len(config.Suites))
	}

	userFlowSuite, ok := config.Suites["userFlow"]
	if !ok {
		t.Errorf("Expected userFlow suite to exist")
	} else {
		if len(userFlowSuite.Requests) != 2 {
			t.Errorf("Expected userFlow to have 2 requests, got %d", len(userFlowSuite.Requests))
		}
		if userFlowSuite.Vars["userId"] != "1" {
			t.Errorf("Expected userFlow userId to be 1, got %s", userFlowSuite.Vars["userId"])
		}
		if len(userFlowSuite.Tests) != 1 {
			t.Errorf("Expected userFlow to have 1 test, got %d", len(userFlowSuite.Tests))
		}
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("non-existent-file.json")
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestProcessEnvironment(t *testing.T) {
	env := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "123",
		"token":   "abc123",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No variables",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "Single variable",
			input:    "{{baseUrl}}/users",
			expected: "https://api.example.com/users",
		},
		{
			name:     "Multiple variables",
			input:    "{{baseUrl}}/users/{{userId}}?token={{token}}",
			expected: "https://api.example.com/users/123?token=abc123",
		},
		{
			name:     "Unknown variable",
			input:    "{{baseUrl}}/users/{{unknown}}",
			expected: "https://api.example.com/users/{{unknown}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessEnvironment(tt.input, env)
			if result != tt.expected {
				t.Errorf("ProcessEnvironment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestProcessEnvironmentInMap(t *testing.T) {
	env := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "123",
	}

	input := map[string]string{
		"url":   "{{baseUrl}}/users/{{userId}}",
		"token": "Bearer {{userId}}-token",
		"plain": "No variables here",
	}

	expected := map[string]string{
		"url":   "https://api.example.com/users/123",
		"token": "Bearer 123-token",
		"plain": "No variables here",
	}

	result := ProcessEnvironmentInMap(input, env)

	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("ProcessEnvironmentInMap()[%s] = %v, want %v", key, result[key], expectedValue)
		}
	}
}

func TestMergeEnvironments(t *testing.T) {
	base := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "123",
		"token":   "abc",
	}

	override := map[string]string{
		"userId": "456",
		"newVar": "xyz",
	}

	expected := map[string]string{
		"baseUrl": "https://api.example.com",
		"userId":  "456",
		"token":   "abc",
		"newVar":  "xyz",
	}

	result := MergeEnvironments(base, override)

	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("MergeEnvironments()[%s] = %v, want %v", key, result[key], expectedValue)
		}
	}
}

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		configPath string
		expected   string
	}{
		{
			configPath: "/path/to/config.json",
			expected:   "/path/to",
		},
		{
			configPath: "config.json",
			expected:   ".",
		},
		{
			configPath: "./config.json",
			expected:   ".",
		},
		{
			configPath: "../config.json",
			expected:   "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.configPath, func(t *testing.T) {
			result := GetConfigDir(tt.configPath)
			if result != tt.expected {
				t.Errorf("GetConfigDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}
