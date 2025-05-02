# Working with Variables

Variables are a powerful feature in Lunge that allow you to make your requests dynamic and reusable. This guide explains how to define, use, and extract variables.

## Variable Types

Lunge supports several types of variables:

1. **Environment Variables** - Defined in the environment section of your configuration
2. **Suite Variables** - Defined in a specific test suite
3. **Extracted Variables** - Extracted from response data during execution
4. **Command-line Variables** - Provided when running a command

## Defining Variables

### Environment Variables

Environment variables are defined in the `environments` section of your configuration file:

```json
"environments": {
  "dev": {
    "baseUrl": "https://api.dev.example.com",
    "variables": {
      "apiKey": "dev-key-123",
      "userId": "test-user-1"
    }
  }
}
```

### Suite Variables

Suite variables are defined in the `variables` section of a suite:

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

### Command-line Variables

You can provide variables when running a command:

```bash
lunge run -c config.json -e dev -r getUser --var userId=123 --var format=json
```

## Using Variables

Variables are referenced using the `{{variableName}}` syntax. They can be used in:

- URLs
- Headers
- Query parameters
- Request bodies
- JSONPath expressions

Examples:

```json
"getUser": {
  "url": "/users/{{userId}}",
  "headers": {
    "Authorization": "Bearer {{apiKey}}"
  },
  "queryParams": {
    "format": "{{format}}"
  }
}
```

## Variable Extraction

One of the most powerful features of Lunge is the ability to extract values from response data and use them in subsequent requests. This is especially useful for workflows that require data from one API call to be used in another.

### Defining Extractions

Extractions are defined in the `extract` section of a request:

```json
"createUser": {
  "url": "/users",
  "method": "POST",
  "body": {
    "name": "John Doe",
    "email": "john@example.com"
  },
  "extract": {
    "userId": "$.id",
    "authToken": "$.token"
  }
}
```

In this example, after the `createUser` request is executed:
- The value at `$.id` in the response will be stored in the `userId` variable
- The value at `$.token` in the response will be stored in the `authToken` variable

### JSONPath Syntax

Lunge uses JSONPath to extract values from JSON responses. Some common JSONPath expressions:

- `$.property` - Access a property at the root level
- `$.nested.property` - Access a nested property
- `$.array[0]` - Access the first element of an array
- `$.array[*].property` - Access a property from all elements in an array

### Using Extracted Variables

Once extracted, variables can be used in subsequent requests in the same suite:

```json
"suites": {
  "userFlow": {
    "requests": ["createUser", "getUser"],
    "tests": [
      {
        "name": "Create User Test",
        "request": "createUser"
      },
      {
        "name": "Get User Test",
        "request": "getUser",
        "assertions": [
          { "path": "$.id", "equals": "{{userId}}" }
        ]
      }
    ]
  }
}
```

In this example, the `userId` extracted from the `createUser` response is used in:
1. The URL of the `getUser` request (if it's defined as `/users/{{userId}}`)
2. An assertion to verify the returned user has the expected ID

## Variable Precedence

When multiple variables with the same name are defined, Lunge uses the following precedence (highest to lowest):

1. Extracted variables from previous responses
2. Command-line variables
3. Suite variables
4. Environment variables

This means that if a variable is extracted during execution, it will override any pre-defined variable with the same name.

## Next Steps

- [Testing](./Testing.md) - Learn about testing and assertions
- [Examples](./Examples.md) - See examples of variable usage in action