package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	// Path is the JSON path to the invalid field
	Path string

	// Message describes the validation error
	Message string
}

// Error returns the error message.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidateConfig validates the configuration and returns a slice of validation errors.
// An empty slice indicates the configuration is valid.
//
// Example:
//
//	errors := config.ValidateConfig(cfg)
//	if len(errors) > 0 {
//	    for _, err := range errors {
//	        log.Printf("Validation error: %s", err)
//	    }
//	}
func ValidateConfig(config *Config) []ValidationError {
	var errors []ValidationError

	// Validate environments
	if len(config.Environments) == 0 {
		errors = append(errors, ValidationError{
			Path:    "environments",
			Message: "at least one environment is required",
		})
	}

	for name, env := range config.Environments {
		// Validate environment
		if env.BaseURL == "" {
			errors = append(errors, ValidationError{
				Path:    fmt.Sprintf("environments.%s.baseUrl", name),
				Message: "baseUrl is required",
			})
		}
	}

	// Validate requests
	if len(config.Requests) == 0 {
		errors = append(errors, ValidationError{
			Path:    "requests",
			Message: "at least one request is required",
		})
	}

	for name, req := range config.Requests {
		// Validate request
		if req.URL == "" {
			errors = append(errors, ValidationError{
				Path:    fmt.Sprintf("requests.%s.url", name),
				Message: "url is required",
			})
		}

		if req.Method == "" {
			errors = append(errors, ValidationError{
				Path:    fmt.Sprintf("requests.%s.method", name),
				Message: "method is required",
			})
		} else {
			// Validate method
			method := strings.ToUpper(req.Method)
			if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" &&
				method != "PATCH" && method != "HEAD" && method != "OPTIONS" {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("requests.%s.method", name),
					Message: fmt.Sprintf("invalid method: %s", req.Method),
				})
			}
		}

		// Validate extract paths
		for varName, path := range req.Extract {
			if path == "" {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("requests.%s.extract.%s", name, varName),
					Message: "extract path cannot be empty",
				})
			}
		}
	}

	// Validate suites
	for name, suite := range config.Suites {
		// Validate suite
		if len(suite.Requests) == 0 {
			errors = append(errors, ValidationError{
				Path:    fmt.Sprintf("suites.%s.requests", name),
				Message: "at least one request is required",
			})
		}

		// Validate request references
		for i, reqName := range suite.Requests {
			if _, ok := config.Requests[reqName]; !ok {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("suites.%s.requests[%d]", name, i),
					Message: fmt.Sprintf("request not found: %s", reqName),
				})
			}
		}

		// Validate tests
		for i, test := range suite.Tests {
			if test.Name == "" {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("suites.%s.tests[%d].name", name, i),
					Message: "test name is required",
				})
			}

			if test.Request == "" {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("suites.%s.tests[%d].request", name, i),
					Message: "test request is required",
				})
			} else if _, ok := config.Requests[test.Request]; !ok {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("suites.%s.tests[%d].request", name, i),
					Message: fmt.Sprintf("request not found: %s", test.Request),
				})
			}

			if len(test.Assertions) == 0 {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("suites.%s.tests[%d].assertions", name, i),
					Message: "at least one assertion is required",
				})
			}
		}
	}

	return errors
}

// ValidateEnvironment validates that an environment exists in the configuration.
func ValidateEnvironment(config *Config, envName string) error {
	if _, ok := config.Environments[envName]; !ok {
		return fmt.Errorf("environment not found: %s", envName)
	}
	return nil
}

// ValidateRequest validates that a request exists in the configuration.
func ValidateRequest(config *Config, reqName string) error {
	if _, ok := config.Requests[reqName]; !ok {
		return fmt.Errorf("request not found: %s", reqName)
	}
	return nil
}

// ValidateSuite validates that a suite exists in the configuration.
func ValidateSuite(config *Config, suiteName string) error {
	if _, ok := config.Suites[suiteName]; !ok {
		return fmt.Errorf("suite not found: %s", suiteName)
	}
	return nil
}

// ValidateTest validates that a test exists in a suite.
func ValidateTest(config *Config, suiteName, testName string) error {
	suite, ok := config.Suites[suiteName]
	if !ok {
		return fmt.Errorf("suite not found: %s", suiteName)
	}

	for _, test := range suite.Tests {
		if test.Name == testName {
			return nil
		}
	}

	return fmt.Errorf("test not found: %s in suite %s", testName, suiteName)
}

// GetEnvironmentNames returns a slice of environment names in the configuration.
func GetEnvironmentNames(config *Config) []string {
	names := make([]string, 0, len(config.Environments))
	for name := range config.Environments {
		names = append(names, name)
	}
	return names
}

// GetRequestNames returns a slice of request names in the configuration.
func GetRequestNames(config *Config) []string {
	names := make([]string, 0, len(config.Requests))
	for name := range config.Requests {
		names = append(names, name)
	}
	return names
}

// GetSuiteNames returns a slice of suite names in the configuration.
func GetSuiteNames(config *Config) []string {
	names := make([]string, 0, len(config.Suites))
	for name := range config.Suites {
		names = append(names, name)
	}
	return names
}
