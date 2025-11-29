# Requirements Document

## Introduction

Lunge is a terminal-based HTTP client written in Go that combines curl's simplicity with Postman/Insomnia's power, with a special emphasis on testing capabilities and response validation. The system provides command-line HTTP request execution, configuration-based request management, comprehensive response validation, and detailed testing capabilities.

## Glossary

- **Lunge_System**: The complete HTTP client application including CLI, configuration management, and testing framework
- **HTTP_Client**: The core component responsible for executing HTTP requests
- **Config_Manager**: Component that loads and validates JSON configuration files
- **Test_Runner**: Component that executes test suites with assertions
- **Response_Validator**: Component that validates HTTP responses against defined criteria
- **Output_Formatter**: Component that formats and displays request/response data
- **Variable_Extractor**: Component that extracts values from responses for use in subsequent requests
- **Environment_Store**: Component that manages environment variables and configurations
- **Suite_Runner**: Component that executes multiple requests in sequence
- **Assertion_Engine**: Component that evaluates test assertions against response data

## Requirements

### Requirement 1

**User Story:** As a developer, I want to execute simple HTTP requests from the command line, so that I can quickly test API endpoints without complex setup.

#### Acceptance Criteria

1. WHEN a user executes a GET command with a URL, THE Lunge_System SHALL send an HTTP GET request to the specified URL
2. WHEN a user executes a POST command with a URL and body data, THE Lunge_System SHALL send an HTTP POST request with the provided body
3. WHEN a user executes a PUT command with a URL and body data, THE Lunge_System SHALL send an HTTP PUT request with the provided body
4. WHEN a user executes a DELETE command with a URL, THE Lunge_System SHALL send an HTTP DELETE request to the specified URL
5. THE Lunge_System SHALL display the response status, headers, and body in a human-readable format

### Requirement 2

**User Story:** As a developer, I want to customize HTTP requests with headers and query parameters, so that I can test APIs that require authentication or specific parameters.

#### Acceptance Criteria

1. WHEN a user provides header flags with key-value pairs, THE Lunge_System SHALL include those headers in the HTTP request
2. WHEN a user provides query parameter flags, THE Lunge_System SHALL append those parameters to the request URL
3. WHEN a user provides a timeout flag, THE Lunge_System SHALL apply the specified timeout to the HTTP request
4. WHEN a user provides a verbose flag, THE Lunge_System SHALL display detailed request and response information
5. THE Lunge_System SHALL support multiple header and query parameter specifications in a single command

### Requirement 3

**User Story:** As a developer, I want to use JSON configuration files to define reusable requests and environments, so that I can manage complex API testing scenarios efficiently.

#### Acceptance Criteria

1. WHEN a configuration file is provided, THE Config_Manager SHALL load and parse the JSON configuration
2. WHEN an environment is specified, THE Config_Manager SHALL apply the environment's base URL and variables
3. WHEN a request name is specified, THE Config_Manager SHALL execute the named request from the configuration
4. THE Config_Manager SHALL validate configuration file structure and report errors for invalid configurations
5. THE Variable_Extractor SHALL substitute variables in URLs, headers, and body content using the {{variable}} syntax

### Requirement 4

**User Story:** As a developer, I want to extract values from API responses and use them in subsequent requests, so that I can create request chains and workflows.

#### Acceptance Criteria

1. WHEN a request defines extraction rules, THE Variable_Extractor SHALL extract values from the response using JSONPath expressions
2. WHEN extracted variables are available, THE Environment_Store SHALL store them for use in subsequent requests
3. WHEN a request references an extracted variable, THE Lunge_System SHALL substitute the variable value in the request
4. THE Variable_Extractor SHALL support extraction from JSON response bodies
5. THE Environment_Store SHALL maintain variable scope within suite execution

### Requirement 5

**User Story:** As a developer, I want to run test suites with assertions to validate API responses, so that I can automate API testing and ensure correctness.

#### Acceptance Criteria

1. WHEN a test suite is executed, THE Test_Runner SHALL execute all requests in the specified suite
2. WHEN test assertions are defined, THE Assertion_Engine SHALL evaluate each assertion against the response
3. WHEN assertions pass or fail, THE Test_Runner SHALL report the results with detailed information
4. THE Assertion_Engine SHALL support status code, response time, header, and JSONPath assertions
5. THE Test_Runner SHALL provide summary statistics for test suite execution

### Requirement 6

**User Story:** As a developer, I want to validate API responses against JSON schemas, so that I can ensure response structure compliance.

#### Acceptance Criteria

1. WHEN a schema is defined in the configuration, THE Response_Validator SHALL store the schema definition
2. WHEN a schema assertion is specified, THE Response_Validator SHALL validate the response against the named schema
3. WHEN schema validation fails, THE Response_Validator SHALL provide detailed error information
4. THE Response_Validator SHALL support JSON Schema draft-04, draft-07, and 2019-09 specifications
5. THE Response_Validator SHALL report validation results as part of test assertion outcomes

### Requirement 7

**User Story:** As a developer, I want multiple output formats for request and test results, so that I can integrate with different tools and workflows.

#### Acceptance Criteria

1. WHEN a format flag is specified, THE Output_Formatter SHALL format output according to the specified format
2. THE Output_Formatter SHALL support text, JSON, YAML, and JUnit XML output formats
3. WHEN verbose mode is enabled, THE Output_Formatter SHALL include detailed timing metrics
4. WHEN color output is disabled, THE Output_Formatter SHALL produce plain text without color codes
5. THE Output_Formatter SHALL include performance metrics in all output formats

### Requirement 8

**User Story:** As a developer, I want detailed performance metrics for HTTP requests, so that I can analyze and optimize API performance.

#### Acceptance Criteria

1. WHEN a request is executed, THE HTTP_Client SHALL measure DNS lookup, TCP connection, TLS handshake, and response timing
2. WHEN verbose mode is enabled, THE Output_Formatter SHALL display all timing metrics
3. WHEN response time assertions are specified, THE Assertion_Engine SHALL validate response times against thresholds
4. THE HTTP_Client SHALL provide timing data in milliseconds with appropriate precision
5. THE Output_Formatter SHALL include timing metrics in structured output formats

### Requirement 9

**User Story:** As a developer, I want to organize requests into suites for sequential execution, so that I can create complex testing workflows.

#### Acceptance Criteria

1. WHEN a suite is defined, THE Suite_Runner SHALL execute requests in the specified order
2. WHEN suite variables are defined, THE Environment_Store SHALL make them available to all requests in the suite
3. WHEN a suite is executed, THE Suite_Runner SHALL maintain variable state across request executions
4. THE Suite_Runner SHALL support both request execution and test validation within suites
5. THE Suite_Runner SHALL provide consolidated reporting for suite execution results

### Requirement 10

**User Story:** As a developer, I want comprehensive assertion types for response validation, so that I can thoroughly test API behavior.

#### Acceptance Criteria

1. THE Assertion_Engine SHALL support status code equality assertions
2. THE Assertion_Engine SHALL support response time comparison assertions with operators
3. THE Assertion_Engine SHALL support header existence, equality, contains, and regex match assertions
4. THE Assertion_Engine SHALL support JSONPath existence, equality, contains, regex, array, and length assertions
5. THE Assertion_Engine SHALL provide detailed failure messages for failed assertions