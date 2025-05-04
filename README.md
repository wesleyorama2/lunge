# Lunge

Lunge is a powerful yet simple terminal-based HTTP client written in Go that combines curl's simplicity with Postman/Insomnia's power, with a special emphasis on testing capabilities and response validation.

## Features

- Command-line simplicity with powerful request customization
- Multiple output formats (Text, JSON, YAML, JUnit XML)
- Pretty request/response formatting in the terminal
- Detailed performance metrics (DNS, TCP, TLS, TTFB, etc.)
- JSON configuration for request suites
- Environment variables and variable substitution
- Response validation (status, headers, body, schema)
- Testing capabilities with assertions
- Request chaining (using output from one request in another)

## Installation

```bash
# Install from source
go install github.com/wesleyorama2/lunge/cmd/lunge

# Or clone the repository and build
git clone https://github.com/wesleyorama2/lunge.git
cd lunge
go build -o lunge ./cmd/lunge
```

## Usage

### Simple Requests

```bash
# Simple GET request
lunge get https://api.example.com/users

# GET request with headers
lunge get https://api.example.com/users -H "Authorization: Bearer token" -H "Accept: application/json"

# POST with JSON body
lunge post https://api.example.com/users -H "Content-Type: application/json" -d '{"name": "John"}'

# POST with JSON body from a separate flag
lunge post https://api.example.com/users -j '{"name": "John"}'

# PUT to update a resource
lunge put https://api.example.com/users/123 -j '{"name": "Updated Name", "email": "updated@example.com"}'

# DELETE a resource
lunge delete https://api.example.com/users/123

# Enable verbose output
lunge get https://api.example.com/users -v

# Set request timeout
lunge get https://api.example.com/users -t 10s

# Output in different formats
lunge get https://api.example.com/users --format json
lunge get https://api.example.com/users --format yaml
lunge get https://api.example.com/users --format junit
```

### Using Configuration Files

Lunge supports JSON configuration files that define environments, requests, and test suites. Here's how to use them:

```bash
# Run a request from a configuration file
lunge run -c examples/simple.json -e dev -r getUser

# Run a suite of requests
lunge run -c examples/simple.json -e dev -s userFlow
```

### Testing

Lunge includes a powerful testing framework that allows you to validate responses and create test suites:

```bash
# Run a test suite
lunge test -c examples/simple.json -e dev -s userFlow

# Run a specific test
lunge test -c examples/simple.json -e dev -s userFlow -t "User exists and has valid email"
```

## Configuration Format

Lunge uses a JSON configuration format that defines environments, requests, and test suites. Here's a simple example:

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

See the `examples/` directory for more configuration examples.

## Documentation

Comprehensive documentation is available in the `doc/` directory:

- [Installation](./doc/Installation.md) - How to install Lunge
- [Getting Started](./doc/Getting-Started.md) - Quick start guide
- [Configuration](./doc/Configuration.md) - Configuration file format and options
- [Variables](./doc/Variables.md) - Working with environment variables and extraction
- [Testing](./doc/Testing.md) - Running tests and assertions
- [JSON Schema Validation](./doc/JSON-Schema-Validation.md) - Validating responses against schemas
- [Examples](./doc/Examples.md) - Common usage examples

## License

[BSD 3-Clause License](./LICENSE)

Copyright (c) 2025, Wesley Brown. All rights reserved.