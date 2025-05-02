package config

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		expectedError bool
		errorCount    int
	}{
		{
			name: "Valid config",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{"getUser"},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Missing environment",
			config: &Config{
				Environments: map[string]Environment{},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Missing baseURL in environment",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Missing requests",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Missing URL in request",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						Method: "GET",
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Missing method in request",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL: "/users/{{userId}}",
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Invalid method in request",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "INVALID",
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Empty extract path",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
						Extract: map[string]string{
							"username": "",
						},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Empty suite requests",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Non-existent request in suite",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{"nonExistentRequest"},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Missing test name",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{"getUser"},
						Tests: []Test{
							{
								Request:    "getUser",
								Assertions: []map[string]interface{}{{"status": 200}},
							},
						},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Missing test request",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{"getUser"},
						Tests: []Test{
							{
								Name:       "Test user",
								Assertions: []map[string]interface{}{{"status": 200}},
							},
						},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Non-existent test request",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{"getUser"},
						Tests: []Test{
							{
								Name:       "Test user",
								Request:    "nonExistentRequest",
								Assertions: []map[string]interface{}{{"status": 200}},
							},
						},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
		{
			name: "Empty test assertions",
			config: &Config{
				Environments: map[string]Environment{
					"dev": {
						BaseURL: "https://api-dev.example.com",
					},
				},
				Requests: map[string]Request{
					"getUser": {
						URL:    "/users/{{userId}}",
						Method: "GET",
					},
				},
				Suites: map[string]Suite{
					"userFlow": {
						Requests: []string{"getUser"},
						Tests: []Test{
							{
								Name:       "Test user",
								Request:    "getUser",
								Assertions: []map[string]interface{}{},
							},
						},
					},
				},
			},
			expectedError: true,
			errorCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateConfig(tt.config)

			if tt.expectedError && len(errors) == 0 {
				t.Errorf("ValidateConfig() expected errors, got none")
			}

			if !tt.expectedError && len(errors) > 0 {
				t.Errorf("ValidateConfig() expected no errors, got %v", errors)
			}

			if tt.errorCount > 0 && len(errors) != tt.errorCount {
				t.Errorf("ValidateConfig() expected %d errors, got %d", tt.errorCount, len(errors))
			}
		})
	}
}

func TestValidateEnvironment(t *testing.T) {
	config := &Config{
		Environments: map[string]Environment{
			"dev": {
				BaseURL: "https://api-dev.example.com",
			},
		},
	}

	// Test valid environment
	err := ValidateEnvironment(config, "dev")
	if err != nil {
		t.Errorf("ValidateEnvironment() expected no error, got %v", err)
	}

	// Test invalid environment
	err = ValidateEnvironment(config, "prod")
	if err == nil {
		t.Errorf("ValidateEnvironment() expected error, got nil")
	}
}

func TestValidateRequest(t *testing.T) {
	config := &Config{
		Requests: map[string]Request{
			"getUser": {
				URL:    "/users/{{userId}}",
				Method: "GET",
			},
		},
	}

	// Test valid request
	err := ValidateRequest(config, "getUser")
	if err != nil {
		t.Errorf("ValidateRequest() expected no error, got %v", err)
	}

	// Test invalid request
	err = ValidateRequest(config, "nonExistentRequest")
	if err == nil {
		t.Errorf("ValidateRequest() expected error, got nil")
	}
}

func TestValidateSuite(t *testing.T) {
	config := &Config{
		Suites: map[string]Suite{
			"userFlow": {
				Requests: []string{"getUser"},
			},
		},
	}

	// Test valid suite
	err := ValidateSuite(config, "userFlow")
	if err != nil {
		t.Errorf("ValidateSuite() expected no error, got %v", err)
	}

	// Test invalid suite
	err = ValidateSuite(config, "nonExistentSuite")
	if err == nil {
		t.Errorf("ValidateSuite() expected error, got nil")
	}
}

func TestValidateTest(t *testing.T) {
	config := &Config{
		Suites: map[string]Suite{
			"userFlow": {
				Requests: []string{"getUser"},
				Tests: []Test{
					{
						Name:    "Test user",
						Request: "getUser",
					},
				},
			},
		},
	}

	// Test valid test
	err := ValidateTest(config, "userFlow", "Test user")
	if err != nil {
		t.Errorf("ValidateTest() expected no error, got %v", err)
	}

	// Test invalid test
	err = ValidateTest(config, "userFlow", "nonExistentTest")
	if err == nil {
		t.Errorf("ValidateTest() expected error, got nil")
	}

	// Test invalid suite
	err = ValidateTest(config, "nonExistentSuite", "Test user")
	if err == nil {
		t.Errorf("ValidateTest() expected error, got nil")
	}
}
