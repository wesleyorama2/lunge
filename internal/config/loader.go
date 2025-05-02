package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the top-level configuration
type Config struct {
	Environments map[string]Environment `json:"environments"`
	Requests     map[string]Request     `json:"requests"`
	Suites       map[string]Suite       `json:"suites"`
}

// Environment represents an environment configuration
type Environment struct {
	BaseURL string            `json:"baseUrl"`
	Headers map[string]string `json:"headers,omitempty"`
	Vars    map[string]string `json:"variables,omitempty"`
}

// Request represents a request configuration
type Request struct {
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string]string      `json:"queryParams,omitempty"`
	Body        interface{}            `json:"body,omitempty"`
	Extract     map[string]string      `json:"extract,omitempty"`
	Validate    map[string]interface{} `json:"validate,omitempty"`
}

// Suite represents a suite of requests
type Suite struct {
	Requests []string          `json:"requests"`
	Vars     map[string]string `json:"variables,omitempty"`
	Tests    []Test            `json:"tests,omitempty"`
}

// Test represents a test configuration
type Test struct {
	Name       string                   `json:"name"`
	Request    string                   `json:"request"`
	Assertions []map[string]interface{} `json:"assertions"`
}

// LoadConfig loads a configuration file
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

// ProcessEnvironment processes environment variables in a string
func ProcessEnvironment(input string, env map[string]string) string {
	result := input

	// Replace environment variables
	for key, value := range env {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}

	return result
}

// ProcessEnvironmentInMap processes environment variables in a map
func ProcessEnvironmentInMap(input map[string]string, env map[string]string) map[string]string {
	result := make(map[string]string)

	for key, value := range input {
		result[key] = ProcessEnvironment(value, env)
	}

	return result
}

// MergeEnvironments merges two environments, with the second taking precedence
func MergeEnvironments(base, override map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy base environment
	for key, value := range base {
		result[key] = value
	}

	// Override with second environment
	for key, value := range override {
		result[key] = value
	}

	return result
}

// GetConfigDir returns the directory containing the config file
func GetConfigDir(configPath string) string {
	return filepath.Dir(configPath)
}
