# Testing with Lunge

Lunge provides powerful testing capabilities that allow you to validate API responses with assertions. This guide explains how to define and run tests.

## Test Structure

Tests in Lunge are defined within suites in your configuration file. A test consists of:

1. A name for identification
2. A reference to a request to execute
3. A list of assertions to validate the response

## Defining Tests

Tests are defined in the `tests` array within a suite:

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
          { "path": "$.name", "exists": true }
        ]
      },
      {
        "name": "Posts API Test",
        "request": "getPosts",
        "assertions": [
          { "status": 200 },
          { "path": "$", "isArray": true },
          { "path": "$", "minLength": 5 }
        ]
      }
    ]
  }
}
```

## Running Tests

To run tests, use the `test` command:

```bash
# Run all tests in a suite
lunge test -c config.json -e dev -s apiTests

# Run a specific test
lunge test -c config.json -e dev -s apiTests -t "User API Test"
```

## Assertion Types

Lunge supports various types of assertions to validate different aspects of the response.

### Status Code Assertions

Verify the HTTP status code:

```json
{ "status": 200 }
```

### Response Time Assertions

Verify the response time in milliseconds:

```json
{ "responseTime": "<500" }  // Less than 500ms
{ "responseTime": ">10" }   // Greater than 10ms
{ "responseTime": "=100" }  // Exactly 100ms
{ "responseTime": "<=1000" } // Less than or equal to 1000ms
{ "responseTime": ">=50" }  // Greater than or equal to 50ms
```

### Header Assertions

Verify response headers:

```json
// Check if header exists
{ "header": "Content-Type", "exists": true }

// Check exact header value
{ "header": "Content-Type", "equals": "application/json; charset=utf-8" }

// Check if header contains a substring
{ "header": "Content-Type", "contains": "application/json" }

// Check if header matches a regex pattern
{ "header": "Content-Type", "matches": "application/.*" }
```

### JSONPath Assertions

Verify values in the response body using JSONPath:

```json
// Check if a path exists
{ "path": "$.name", "exists": true }

// Check exact value
{ "path": "$.status", "equals": "active" }

// Check if value contains a substring
{ "path": "$.message", "contains": "success" }

// Check if value matches a regex pattern
{ "path": "$.email", "matches": ".*@.*\\..*" }

// Check if value is an array
{ "path": "$", "isArray": true }

// Check array minimum length
{ "path": "$", "minLength": 5 }
```

## Advanced Assertions

### Using Variables in Assertions

You can use variables in assertions to compare against dynamic values:

```json
{ "path": "$.id", "equals": "{{userId}}" }
```

### Chaining Assertions

You can apply multiple assertions to the same path or header:

```json
[
  { "path": "$.items", "exists": true },
  { "path": "$.items", "isArray": true },
  { "path": "$.items", "minLength": 1 }
]
```

## Test Results

When you run tests, Lunge provides detailed output showing:

- Request details (URL, method, headers, body)
- Response details (status, headers, body)
- Each assertion result (pass/fail)
- Overall test result (pass/fail)
- Summary of all tests in the suite

Example output:

```
▶ RUNNING TEST SUITE: apiTests (2 tests)

TEST 1: User API Test
  ▶ REQUEST: GET https://api.example.com/users/1
    Headers:
      Accept: application/json
    ◀ RESPONSE: 200 OK (45ms)
    Body:
  {
      "id": 1,
      "name": "John Doe",
      "email": "john@example.com"
  }
    ✓ ASSERTION PASSED: Status code is 200
  ✓ ASSERTION PASSED: Response time 45ms is less than 500ms
  ✓ ASSERTION PASSED: Header Content-Type contains application/json
  ✓ ASSERTION PASSED: Path $.name exists: true

  ✓ TEST PASSED (45ms)

TEST 2: Posts API Test
  ...

▶ TEST SUITE SUMMARY: apiTests
  ✅ Tests: 2 passed, 0 failed
  ✅ Assertions: 7 passed, 0 failed
  ✅ Total time: 120ms
```

## Best Practices

1. **Name tests clearly** - Use descriptive names that indicate what's being tested
2. **Test one thing per test** - Each test should focus on validating one aspect of the API
3. **Use variables** - Extract values from responses and use them in subsequent tests
4. **Check response times** - Include response time assertions to catch performance issues
5. **Validate schema** - Use JSON Schema validation for comprehensive response validation

## Next Steps

- [JSON Schema Validation](./JSON-Schema-Validation.md) - Learn about schema validation
- [Examples](./Examples.md) - See examples of tests in action