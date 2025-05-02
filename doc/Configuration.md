# Configuration File Reference

Lunge uses JSON configuration files to define environments, requests, and test suites. This document provides a comprehensive reference for the configuration file format.

## File Structure

A Lunge configuration file has the following top-level structure:

```json
{
  "environments": { ... },
  "requests": { ... },
  "suites": { ... },
  "schemas": { ... }
}
```

## Environments

The `environments` section defines different environments (e.g., development, staging, production) with their base URLs and variables.

```json
"environments": {
  "dev": {
    "baseUrl": "https://api.dev.example.com",
    "variables": {
      "apiKey": "dev-key-123",
      "userId": "test-user-1"
    }
  },
  "prod": {
    "baseUrl": "https://api.example.com",
    "variables": {
      "apiKey": "prod-key-456",
      "userId": "live-user-1"
    }
  }
}
```

### Environment Properties

| Property | Type | Description |
|----------|------|-------------|
| `baseUrl` | String | The base URL for all requests in this environment |
| `variables` | Object | Key-value pairs of variables available in this environment |

## Requests

The `requests` section defines reusable HTTP requests that can be executed individually or as part of a suite.

```json
"requests": {
  "getUser": {
    "url": "/users/{{userId}}",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{apiKey}}",
      "Accept": "application/json"
    },
    "queryParams": {
      "include": "profile,settings"
    }
  },
  "createUser": {
    "url": "/users",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer {{apiKey}}"
    },
    "body": {
      "name": "John Doe",
      "email": "john@example.com"
    },
    "extract": {
      "userId": "$.id"
    }
  }
}
```

### Request Properties

| Property | Type | Description |
|----------|------|-------------|
| `url` | String | The URL path (appended to the environment's baseUrl) |
| `method` | String | HTTP method (GET, POST, PUT, DELETE, etc.) |
| `headers` | Object | Key-value pairs of HTTP headers |
| `queryParams` | Object | Key-value pairs of query parameters |
| `body` | Object/String | Request body (object will be serialized as JSON) |
| `extract` | Object | Variables to extract from the response (key: variable name, value: JSONPath) |

## Suites

The `suites` section defines test suites that run multiple requests in sequence.

```json
"suites": {
  "userFlow": {
    "requests": ["createUser", "getUser"],
    "variables": {
      "testData": "custom-value"
    }
  }
}
```

### Suite Properties

| Property | Type | Description |
|----------|------|-------------|
| `requests` | Array | List of request names to execute in order |
| `variables` | Object | Additional variables specific to this suite |
| `tests` | Array | Test configurations for validating responses |

## Tests

Within a suite, you can define tests with assertions to validate responses.

```json
"suites": {
  "apiTests": {
    "requests": ["getUser", "getPosts"],
    "tests": [
      {
        "name": "User API Test",
        "request": "getUser",
        "assertions": [
          { "status": 200 },
          { "responseTime": "<500" },
          { "header": "Content-Type", "contains": "application/json" },
          { "path": "$.name", "exists": true },
          { "path": "$.email", "matches": ".*@.*\\..*" }
        ]
      }
    ]
  }
}
```

### Test Properties

| Property | Type | Description |
|----------|------|-------------|
| `name` | String | Name of the test |
| `request` | String | Name of the request to test |
| `assertions` | Array | List of assertions to validate the response |

### Assertion Types

| Type | Example | Description |
|------|---------|-------------|
| Status | `{ "status": 200 }` | Checks response status code |
| Response Time | `{ "responseTime": "<500" }` | Checks response time (ms) |
| Header | `{ "header": "Content-Type", "contains": "json" }` | Checks response header |
| JSONPath | `{ "path": "$.name", "exists": true }` | Checks value at JSONPath |
| JSONPath Equals | `{ "path": "$.status", "equals": "active" }` | Checks equality at JSONPath |
| JSONPath Contains | `{ "path": "$.message", "contains": "success" }` | Checks if value contains substring |
| JSONPath Matches | `{ "path": "$.email", "matches": ".*@.*\\..*" }` | Checks if value matches regex |
| JSONPath Array | `{ "path": "$", "isArray": true }` | Checks if value is an array |
| JSONPath Length | `{ "path": "$", "minLength": 5 }` | Checks array minimum length |

## Schemas

The `schemas` section defines JSON schemas for validating responses.

```json
"schemas": {
  "userSchema": {
    "type": "object",
    "required": ["id", "name", "email"],
    "properties": {
      "id": { "type": "integer" },
      "name": { "type": "string" },
      "email": { "type": "string", "format": "email" }
    }
  }
}
```

## Variable Substitution

Variables can be referenced in the configuration using the `{{variableName}}` syntax. Variables can come from:

1. Environment variables defined in the configuration
2. Suite variables defined in the configuration
3. Extracted variables from previous responses in a suite

## Next Steps

- [Variables](./Variables.md) - Learn more about working with variables and extraction
- [Testing](./Testing.md) - Detailed guide on testing and assertions
- [JSON Schema Validation](./JSON-Schema-Validation.md) - Learn about schema validation