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

### Adding Headers

You can add headers to your requests:

```bash
lunge get https://api.example.com/users --header "Authorization: Bearer token123" --header "Accept: application/json"
```

### Formatting Output

By default, Lunge formats the output for readability. You can control this with flags:

```bash
# Pretty-print JSON response
lunge get https://api.example.com/users --pretty

# Raw output (no formatting)
lunge get https://api.example.com/users --raw
```

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