# Getting Started with Lunge

This guide will help you get started with Lunge by walking through basic usage and common commands.

## Basic Commands

### Making a Simple GET Request

To make a simple GET request, use the `get` command:

```bash
lunge get https://api.example.com/users
```

This will send a GET request to the specified URL and display the response.

### Making a POST Request

To make a POST request with a JSON body:

```bash
lunge post https://api.example.com/users --body '{"name": "John Doe", "email": "john@example.com"}'
```

### Making a PUT Request

To update a resource, use the `put` command with a JSON body:

```bash
lunge put https://api.example.com/users/123 --json '{"name": "Updated Name", "email": "updated@example.com"}'
```

This will send a PUT request with the specified JSON body to update the resource.

### Making a DELETE Request

To delete a resource, use the `delete` command:

```bash
lunge delete https://api.example.com/users/123
```

This will send a DELETE request to the specified URL and display the response.

### Adding Headers

You can add headers to your requests:

```bash
lunge get https://api.example.com/users --header "Authorization: Bearer token123" --header "Accept: application/json"
```

### Output Formats

Lunge supports multiple output formats to suit different needs:

```bash
# Default text format (human-readable)
lunge get https://api.example.com/users

# JSON format (for programmatic processing)
lunge get https://api.example.com/users --format json

# YAML format (for configuration-like readability)
lunge get https://api.example.com/users --format yaml

# JUnit XML format (for CI/CD integration)
lunge get https://api.example.com/users --format junit
```

The `--format` flag works with all commands (get, post, put, delete, run, test) and supports the following values:
- `text` (default): Human-readable formatted output
- `json`: Structured JSON output
- `yaml`: YAML formatted output
- `junit`: XML output compatible with CI/CD systems

You can also control the verbosity and color of the output:

```bash
# Enable verbose output
lunge get https://api.example.com/users -v

# Disable colored output
lunge get https://api.example.com/users --no-color
```

### Performance Metrics

Lunge provides detailed performance metrics for HTTP requests. When using the verbose flag (`-v`), you'll see a breakdown of timing information:

```bash
# Show detailed timing metrics
lunge get https://api.example.com/users -v
```

The timing metrics include:

- **DNS Lookup**: Time taken to resolve the domain name to an IP address
- **TCP Connection**: Time taken to establish a TCP connection
- **TLS Handshake**: Time taken to complete the TLS handshake (for HTTPS)
- **Time to First Byte (TTFB)**: Time from sending the request to receiving the first byte of the response
- **Content Transfer**: Time taken to download the response body
- **Total**: Overall time from request start to completion

**Note**: The individual timing metrics will not sum up to the total time. This is because:
1. There are gaps between some phases (e.g., between DNS lookup and TCP connection)
2. Some operations are not explicitly measured (e.g., request preparation)
3. Each metric is measured independently using HTTP trace hooks
4. The total time includes all operations from start to finish

These metrics are also included in the structured output formats (JSON, YAML, JUnit XML), making it easy to analyze performance data programmatically.

## Using Configuration Files

Lunge becomes more powerful when using configuration files to define requests, environments, and test suites.

### Creating a Configuration File

Create a file named `lunge.json` (or any name you prefer):

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://api.dev.example.com",
      "variables": {
        "apiKey": "dev-api-key-123"
      }
    },
    "prod": {
      "baseUrl": "https://api.example.com",
      "variables": {
        "apiKey": "prod-api-key-456"
      }
    }
  },
  "requests": {
    "getUsers": {
      "url": "/users",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{apiKey}}",
        "Accept": "application/json"
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
      }
    }
  }
}
```

### Running Requests from Configuration

To run a request defined in your configuration file:

```bash
# Run the getUsers request in the dev environment
lunge run -c lunge.json -e dev -r getUsers

# Run the createUser request in the prod environment
lunge run -c lunge.json -e prod -r createUser
```

## Next Steps

Now that you understand the basics, check out these guides for more advanced usage:

- [Configuration](./Configuration.md) - Learn more about configuration file options
- [Variables](./Variables.md) - Working with environment variables and extraction
- [Testing](./Testing.md) - Running tests and assertions
- [Examples](./Examples.md) - Common usage examples