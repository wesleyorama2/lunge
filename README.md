# Lunge

[![Go Report Card](https://goreportcard.com/badge/github.com/wesleyorama2/lunge)](https://goreportcard.com/report/github.com/wesleyorama2/lunge)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](LICENSE)

Lunge is a powerful yet simple terminal-based HTTP client and performance testing tool written in Go. It combines curl's simplicity with Postman/Insomnia's power, featuring a comprehensive performance testing engine inspired by k6 and Gatling.

## Features

### HTTP Client
- **Command-line simplicity** with powerful request customization
- **Multiple output formats** - Text, JSON, YAML, JUnit XML
- **Pretty formatting** - Syntax-highlighted request/response output
- **Detailed timing metrics** - DNS, TCP, TLS, TTFB breakdown
- **Request chaining** - Use output from one request in another

### Testing Framework
- **JSON/YAML configuration** for request suites
- **Environment variables** and variable substitution
- **Response validation** - Status, headers, body, JSON schema
- **Assertion testing** with detailed pass/fail reporting

### Performance Testing (v2 Engine)
- **Four executor types** - K6/Gatling-style load testing:
  - `constant-vus` - Fixed virtual users
  - `ramping-vus` - Variable VU stages
  - `constant-arrival-rate` - Fixed RPS throughput
  - `ramping-arrival-rate` - Variable RPS stages
- **HTML reports** with Chart.js visualization
- **Threshold support** for pass/fail criteria
- **TTY-aware console** with real-time progress bars
- **Lock-free metrics** for high-performance collection

## Installation

### From Source

```bash
go install github.com/wesleyorama2/lunge/cmd/lunge@latest
```

### Build from Repository

```bash
git clone https://github.com/wesleyorama2/lunge.git
cd lunge
go build -o lunge ./cmd/lunge
```

### Docker

```bash
docker build -t lunge .
docker run --rm lunge get https://api.example.com/health
```

## Quick Start

### Simple HTTP Requests

```bash
# GET request
lunge get https://api.example.com/users

# GET with headers
lunge get https://api.example.com/users \
  -H "Authorization: Bearer token" \
  -H "Accept: application/json"

# POST with JSON body
lunge post https://api.example.com/users \
  -j '{"name": "John", "email": "john@example.com"}'

# PUT request
lunge put https://api.example.com/users/123 \
  -j '{"name": "Updated Name"}'

# DELETE request
lunge delete https://api.example.com/users/123

# Verbose output with timing details
lunge get https://api.example.com/users -v
```

### Configuration File Requests

```bash
# Run a single request
lunge run -c examples/simple.json -e dev -r getUser

# Run a test suite
lunge test -c examples/simple.json -e dev -s userFlow
```

### Performance Testing

```bash
# Quick load test
lunge perf --url https://api.example.com/health \
  --vus 10 --duration 30s

# Ramping VUs stress test
lunge perf --url https://api.example.com/health \
  --executor ramping-vus \
  --stages "30s:10,2m:50,30s:0"

# Constant arrival rate (guaranteed RPS)
lunge perf --url https://api.example.com/health \
  --executor constant-arrival-rate \
  --rate 100 --duration 5m --max-vus 200

# Using a configuration file with HTML report
lunge perf -c examples/v2-stress-test.yaml --html --output report.html
```

## Configuration Format

### HTTP Request Configuration (JSON)

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://api-dev.example.com",
      "variables": {
        "userId": "1"
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
    }
  },
  "suites": {
    "userFlow": {
      "requests": ["getUser"],
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
}
```

### Performance Test Configuration (YAML)

```yaml
name: "API Load Test"
settings:
  baseUrl: "https://api.example.com"
  timeout: 30s

scenarios:
  load_test:
    executor: ramping-vus
    stages:
      - duration: 1m
        target: 20
      - duration: 3m
        target: 50
      - duration: 1m
        target: 0
    requests:
      - name: "Health Check"
        method: GET
        url: "{{baseUrl}}/health"

thresholds:
  http_req_duration:
    - "p95 < 500ms"
  http_req_failed:
    - "rate < 0.01"
```

## Documentation

Comprehensive documentation is available in the [`doc/`](./doc/) directory:

- **Getting Started**
  - [Installation](./doc/Installation.md) - Installation methods
  - [Getting Started](./doc/Getting-Started.md) - Quick start guide
  
- **Usage Guides**
  - [Configuration](./doc/Configuration.md) - Configuration file format
  - [Variables](./doc/Variables.md) - Environment variables and extraction
  - [Testing](./doc/Testing.md) - Test assertions and validation
  - [JSON Schema Validation](./doc/JSON-Schema-Validation.md) - Response schema validation
  
- **Performance Testing**
  - [Performance Testing](./doc/Performance-Testing.md) - Load testing guide
  - [Benchmarks](./doc/Benchmarks.md) - Performance benchmarks
  - [Rate Limiting](./doc/Rate-Limiting.md) - Rate limiting details
  
- **Reference**
  - [Examples](./doc/Examples.md) - Common usage examples

## Examples

See the [`examples/`](./examples/) directory for ready-to-use configurations:

- **HTTP Testing**: `simple.json`, `httpbin-tests.json`
- **Performance Testing**: `v2-*.yaml` files for various load patterns
- **Schema Validation**: `schema-validation.json`

## Architecture

Lunge is designed with a clean, modular architecture:

```
cmd/lunge/          # CLI entry point
internal/
├── cli/            # Command implementations
├── config/         # Configuration parsing
├── http/           # HTTP client
├── output/         # Output formatting
└── performance/v2/ # Performance testing engine
    ├── config/     # YAML/JSON config parsing
    ├── engine/     # Test orchestration
    ├── executor/   # Load generation executors
    ├── metrics/    # Metrics collection
    ├── output/     # Console output
    ├── rate/       # Rate limiting
    └── report/     # HTML reporting
pkg/
├── jsonpath/       # JSONPath extraction
└── jsonschema/     # JSON Schema validation
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development

```bash
# Run tests
go test ./...

# Run tests with race detection
go test -race ./...

# Build
go build ./cmd/lunge

# Format code
go fmt ./...

# Vet code
go vet ./cmd/... ./internal/...
```

## License

[BSD 3-Clause License](./LICENSE)

Copyright (c) 2025, Wesley Brown. All rights reserved.