# Lunge: Terminal-based HTTP Client Tool

## Overview

Lunge is a powerful yet simple terminal-based HTTP client written in Go that combines curl's simplicity with Postman/Insomnia's power, with a special emphasis on testing capabilities and response validation.

## Architecture

```mermaid
graph TD
    A[CLI Interface] --> B[Command Parser]
    B --> C[Request Builder]
    B --> D[Config Manager]
    D --> E[Suite Runner]
    C --> F[HTTP Client]
    E --> C
    F --> G[Response Processor]
    G --> H[Output Formatter]
    G --> I[Response Validator]
    I --> I1[Schema Validator]
    I --> I2[Status Validator]
    I --> I3[Header Validator]
    I --> I4[Body Validator]
    I --> I5[Performance Validator]
    G --> J[Variable Extractor]
    J --> K[Environment Store]
    K --> C
    E --> L[Test Runner]
    L --> M[Test Reporter]
    L --> N[Assertion Engine]
    N --> I
```

## Core Components

### 1. CLI Interface
- Provides a simple, intuitive command-line interface
- Supports direct commands and configuration file loading
- Handles flags and arguments for request customization

### 2. Config Manager
- Loads and parses JSON configuration files
- Validates configuration structure
- Manages environment variables and request suites

### 3. Request Builder
- Constructs HTTP requests with headers, query params, and body
- Supports different content types (JSON, form, multipart, etc.)
- Handles variable substitution from environment

### 4. HTTP Client
- Performs the actual HTTP requests
- Manages timeouts, retries, and connection pooling
- Handles different authentication methods

### 5. Response Processor
- Parses HTTP responses
- Extracts data for variable storage
- Prepares data for validation and formatting

### 6. Output Formatter
- Pretty-prints requests and responses
- Supports different output formats (colored text, JSON, etc.)
- Provides different verbosity levels

## Performance Testing Architecture

### Rate Limiting System

Lunge uses an optimized token bucket algorithm for accurate, low-overhead rate limiting during performance tests.

#### Token Bucket Algorithm

The token bucket algorithm provides superior performance compared to traditional ticker-based approaches:

**How It Works:**
1. A "bucket" holds tokens that represent permission to send requests
2. Tokens are added to the bucket at a constant rate (target RPS)
3. Each request consumes one or more tokens
4. If tokens are available, requests proceed immediately
5. If no tokens are available, requests wait until tokens are refilled

**Key Advantages:**
- **No ticker overhead**: Only calculates time when tokens are needed (not every millisecond)
- **Burst handling**: Allows brief bursts up to 2x target rate while maintaining average rate
- **Adaptive refill**: Refills based on actual elapsed time, not fixed intervals
- **Low CPU usage**: Reduces context switches by 100x compared to ticker-based approaches
- **High efficiency**: Achieves 95%+ efficiency at all target RPS levels

**Performance Characteristics:**
- CPU overhead: <5% at 1000 RPS, <10% at 10000 RPS
- Efficiency: 95%+ at 200-1000 RPS, 90%+ at 10000 RPS
- Accuracy: ±2% of target rate
- Burst capacity: 2x target rate (configurable)

#### Batch Request Generation

To further reduce overhead, Lunge generates requests in batches:

**Adaptive Batch Sizing:**
- Low RPS (<100): Batch size = 1 (no batching needed)
- Medium RPS (100-1000): Batch size = 1-10 (linear scaling)
- High RPS (1000-10000): Batch size = 10-100 (linear scaling)
- Very high RPS (>10000): Batch size = 100 (maximum)

This reduces per-request overhead by 10-100x at high RPS levels.

#### Lock-Free Metrics Collection

Metrics are collected using atomic operations to minimize contention:

- **Atomic counters**: No locks for request counting
- **Ring buffer**: Lock-free response time recording
- **Batch flush**: Periodically flush ring buffer to main storage
- **Separate hot/cold paths**: Counters (hot) vs analysis (cold)

This ensures metrics collection doesn't become a bottleneck at high RPS.

#### Efficiency Monitoring

Lunge tracks rate limiter efficiency in real-time:

- **Efficiency**: Ratio of actual RPS to target RPS (0.0 to 1.0)
- **Generation rate**: Rate at which requests are sent to workers
- **Completion rate**: Rate at which workers complete requests
- **Channel utilization**: Percentage of time channel is full
- **Wait time**: Average time spent waiting for tokens

Efficiency warnings are logged when efficiency drops below 95%.

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Performance Engine                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │ Token Bucket │───▶│    Batch     │───▶│   Channel    │  │
│  │ Rate Limiter │    │  Generator   │    │   Manager    │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                    │                    │          │
│         │                    │                    ▼          │
│         │                    │            ┌──────────────┐  │
│         │                    │            │ Worker Pool  │  │
│         │                    │            │ (Goroutines) │  │
│         │                    │            └──────────────┘  │
│         │                    │                    │          │
│         ▼                    ▼                    ▼          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           Atomic Metrics Collector                    │  │
│  │  (Lock-free counters + Ring buffer)                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Ramp-Up and Ramp-Down

Lunge supports smooth rate transitions during ramp-up and ramp-down phases:

**Implementation:**
- Separate goroutine updates token bucket rate every 100ms
- Linear interpolation from start RPS to end RPS
- No per-request calculations during ramps
- Maintains 95%+ efficiency throughout transitions

**Example:**
```
Ramp-up: 0 → 1000 RPS over 30 seconds
- Rate updates every 100ms (300 updates total)
- Each update: currentRPS = startRPS + (endRPS - startRPS) * progress
- Token bucket handles smoothing automatically
```

## Enhanced Testing & Validation Components

### 1. Response Validator (Expanded)

#### Status Validator
- Validates HTTP status codes against expected values
- Supports ranges (e.g., 2xx, 4xx) and specific codes
- Provides detailed error messages for failed validations

#### Header Validator
- Validates response headers against expected values
- Supports exact match, contains, regex patterns
- Case-insensitive and multi-value header support

#### Body Validator
- JSON Schema validation for structured responses
- Content validation using JSONPath expressions
- Support for different content types (JSON, XML, HTML)
- Deep equality and partial matching

#### Schema Validator
- Validates response against OpenAPI/JSON Schema definitions
- Supports draft-04, draft-07, and 2019-09 JSON Schema
- Detailed validation error reporting

#### Performance Validator
- Response time validation against thresholds
- Tracking of slow requests and performance degradation
- Statistical analysis for multiple runs

### 2. Test Runner (New)

#### Test Suite Management
- Organize tests into suites with setup/teardown
- Dependency management between tests
- Parallel test execution option

#### Assertion Engine
- Rich set of assertions (equals, contains, matches, etc.)
- Chained assertions for complex validations
- Custom assertion extensions

#### Test Reporter
- Detailed test reports with pass/fail status
- Summary statistics and timing information
- Export to various formats (JSON, JUnit XML)
- Terminal-friendly colored output

## Configuration Format

The JSON configuration format will support:

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://api-dev.example.com",
      "apiKey": "dev-key"
    },
    "prod": {
      "baseUrl": "https://api.example.com",
      "apiKey": "prod-key"
    }
  },
  "requests": {
    "getUsers": {
      "url": "{{baseUrl}}/users",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{token}}",
        "Content-Type": "application/json"
      },
      "validate": {
        "status": 200,
        "headers": {
          "Content-Type": "application/json"
        },
        "responseTime": "<500ms",
        "body": {
          "type": "array",
          "minItems": 1,
          "path": "$.users",
          "schema": {
            "type": "object",
            "required": ["id", "name", "email"],
            "properties": {
              "id": { "type": "integer" },
              "name": { "type": "string" },
              "email": { "type": "string", "format": "email" }
            }
          }
        }
      }
    },
    "login": {
      "url": "{{baseUrl}}/auth",
      "method": "POST",
      "body": {
        "username": "{{username}}",
        "password": "{{password}}"
      },
      "extract": {
        "token": "$.data.token"
      },
      "validate": {
        "status": 200,
        "body": {
          "path": "$.data.token",
          "exists": true,
          "matches": "^[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]*$"
        }
      }
    }
  },
  "suites": {
    "authFlow": {
      "requests": ["login", "getUsers"],
      "variables": {
        "username": "testuser",
        "password": "testpass"
      },
      "tests": [
        {
          "name": "Login returns valid JWT token",
          "request": "login",
          "assertions": [
            { "path": "$.data.token", "exists": true },
            { "path": "$.data.token", "matches": "^[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]*$" }
          ]
        },
        {
          "name": "Users endpoint returns at least one user",
          "request": "getUsers",
          "assertions": [
            { "status": 200 },
            { "path": "$.users", "isArray": true },
            { "path": "$.users", "minLength": 1 },
            { "path": "$.users[0].id", "exists": true }
          ]
        },
        {
          "name": "Response time is acceptable",
          "request": "getUsers",
          "assertions": [
            { "responseTime": "<500ms" }
          ]
        }
      ]
    }
  }
}
```

## Project Structure

```
lunge/
├── cmd/
│   ├── lunge/                    # Main CLI entry point
│   └── generate-sample-report/   # Report generation utility
├── config/                       # Public config package (library)
├── http/                         # Public HTTP client package (library)
├── perf/                         # Public performance testing package (library)
│   ├── config/                   # Performance test configuration
│   ├── executor/                 # Executor types and interfaces
│   ├── metrics/                  # Metrics collection types
│   └── rate/                     # Rate limiting
├── internal/
│   ├── cli/                      # CLI command implementations
│   ├── config/                   # Internal config handling
│   ├── http/                     # Internal HTTP client implementation
│   ├── output/                   # Output formatting
│   └── performance/
│       └── v2/                   # Performance engine v2
│           ├── config/           # Parser and schema
│           ├── engine/           # Test engine
│           ├── executor/         # Executor implementations
│           ├── metrics/          # Metrics collection
│           ├── output/           # Console output
│           ├── rate/             # Rate limiters
│           └── report/           # Report generation
├── pkg/
│   ├── jsonpath/                 # JSONPath utilities
│   └── jsonschema/               # JSON Schema validation
├── doc/                          # Documentation
├── examples/                     # Example configurations
└── scripts/                      # Development scripts
```

## External Dependencies (Minimal)

1. **github.com/spf13/cobra** - For CLI command structure
2. **github.com/fatih/color** - For terminal color output
3. **github.com/tidwall/gjson** - For JSON parsing and extraction
4. **github.com/olekukonko/tablewriter** - For formatted table output
5. **github.com/xeipuuv/gojsonschema** - For JSON Schema validation
6. **github.com/stretchr/testify** - For assertion capabilities

## User Experience Examples

### Simple Request

```bash
# Simple GET request
lunge get https://api.example.com/users

# POST with JSON body
lunge post https://api.example.com/users -H "Content-Type: application/json" -d '{"name": "John"}'

# Using a configuration file
lunge run -c config.json -e dev -s authFlow
```

### Testing Commands

```bash
# Run a test suite
lunge test -c config.json -e dev -s authFlow

# Run a specific test
lunge test -c config.json -e dev -t "Login returns valid JWT token"

# Run with detailed output
lunge test -c config.json -e dev -s authFlow --verbose
```

### Output Example

```
▶ REQUEST: GET https://api.example.com/users
  Headers:
    User-Agent: lunge/1.0
    Accept: application/json

◀ RESPONSE: 200 OK (154ms)
  Headers:
    Content-Type: application/json
    Content-Length: 237
  
  Body:
  {
    "users": [
      {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com"
      },
      {
        "id": 2,
        "name": "Jane Smith",
        "email": "jane@example.com"
      }
    ]
  }

✓ Validation passed: Status code is 200
```

### Test Output Example

```
▶ RUNNING TEST SUITE: authFlow (3 tests)

TEST 1: Login returns valid JWT token
  ▶ REQUEST: POST https://api-dev.example.com/auth
    Body: {"username":"testuser","password":"testpass"}
  
  ◀ RESPONSE: 200 OK (87ms)
    Body: {"status":"success","data":{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}}
  
  ✓ ASSERTION PASSED: $.data.token exists
  ✓ ASSERTION PASSED: $.data.token matches JWT pattern
  
  ✅ TEST PASSED (87ms)

TEST 2: Users endpoint returns at least one user
  ▶ REQUEST: GET https://api-dev.example.com/users
    Headers: Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
  
  ◀ RESPONSE: 200 OK (124ms)
    Body: {"users":[{"id":1,"name":"John Doe","email":"john@example.com"},{"id":2,"name":"Jane Smith","email":"jane@example.com"}]}
  
  ✓ ASSERTION PASSED: Status code is 200
  ✓ ASSERTION PASSED: $.users is an array
  ✓ ASSERTION PASSED: $.users has at least 1 item
  ✓ ASSERTION PASSED: $.users[0].id exists
  
  ✅ TEST PASSED (124ms)

TEST 3: Response time is acceptable
  ▶ REQUEST: GET https://api-dev.example.com/users
    Headers: Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
  
  ◀ RESPONSE: 200 OK (124ms)
  
  ✓ ASSERTION PASSED: Response time 124ms is less than 500ms
  
  ✅ TEST PASSED (124ms)

▶ TEST SUITE SUMMARY: authFlow
  ✅ Tests: 3 passed, 0 failed
  ✅ Assertions: 8 passed, 0 failed
  ✅ Total time: 335ms
```

## Advanced Testing Features

### 1. Data-Driven Testing
```json
{
  "tests": {
    "userCreation": {
      "request": "createUser",
      "dataSource": "users.json",
      "iterations": [
        {
          "variables": {
            "name": "{{item.name}}",
            "email": "{{item.email}}"
          },
          "assertions": [
            { "status": 201 },
            { "path": "$.id", "exists": true }
          ]
        }
      ]
    }
  }
}
```

### 2. Conditional Testing
```json
{
  "tests": {
    "conditionalTest": {
      "request": "getFeature",
      "when": "{{environment}} == 'prod'",
      "assertions": [
        { "status": 200 }
      ],
      "otherwise": {
        "assertions": [
          { "status": 404 }
        ]
      }
    }
  }
}
```

### 3. Performance Testing
```json
{
  "tests": {
    "loadTest": {
      "request": "getUsers",
      "iterations": 100,
      "concurrency": 10,
      "assertions": [
        { "responseTime": { "mean": "<200ms", "p95": "<500ms" } },
        { "status": 200 }
      ]
    }
  }
}
```

## Next Steps and Considerations

1. **Schema Registry**: Support for storing and referencing schemas
2. **Test History**: Track test results over time to identify regressions
3. **Mock Server Integration**: Generate mock servers from test definitions
4. **CI/CD Integration**: Easy integration with CI/CD pipelines
5. **Extensibility**: Plugin system for custom validators or formatters
6. **Interoperability**: Import/export with Postman/Insomnia formats