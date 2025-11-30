// Package config provides configuration loading and validation utilities
// for Lunge JSON configuration files.
//
// This package supports loading configuration files that define:
//   - Environments: Base URLs, headers, and variables for different target environments
//   - Requests: HTTP request templates with URL, method, headers, and body
//   - Suites: Collections of requests with variable substitution
//   - Tests: Assertions for validating request responses
//
// Basic Usage:
//
//	cfg, err := config.LoadConfig("lunge-config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access environments
//	env := cfg.Environments["production"]
//	fmt.Printf("Base URL: %s\n", env.BaseURL)
//
//	// Access requests
//	req := cfg.Requests["getUsers"]
//	fmt.Printf("Method: %s, URL: %s\n", req.Method, req.URL)
//
// Variable Substitution:
//
// Variables can be defined at the environment or suite level and used in
// request URLs, headers, and bodies using the {{variableName}} syntax.
//
//	// Process variable substitution
//	url := config.ProcessEnvironment(req.URL, env.Vars)
//
// Configuration Validation:
//
// The ValidateConfig function validates the configuration and returns
// a slice of validation errors:
//
//	errors := config.ValidateConfig(cfg)
//	if len(errors) > 0 {
//	    for _, err := range errors {
//	        log.Printf("Validation error: %s", err)
//	    }
//	}
package config
