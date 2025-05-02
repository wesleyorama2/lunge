# Lunge Examples

This document provides practical examples of using Lunge for common API testing and automation scenarios.

## Basic Examples

### Simple GET Request

```bash
lunge get https://jsonplaceholder.typicode.com/users/1
```

### POST Request with JSON Body

```bash
lunge post https://jsonplaceholder.typicode.com/posts \
  --header "Content-Type: application/json" \
  --body '{"title": "Test Post", "body": "This is a test post", "userId": 1}'
```

### PUT Request with JSON Body

```bash
lunge put https://jsonplaceholder.typicode.com/posts/1 \
  --header "Content-Type: application/json" \
  --body '{"id": 1, "title": "Updated Post", "body": "This post has been updated", "userId": 1}'
```

### DELETE Request

```bash
lunge delete https://jsonplaceholder.typicode.com/posts/1
```

## Configuration File Examples

### Basic Configuration

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://jsonplaceholder.typicode.com",
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
    },
    "getPosts": {
      "url": "/posts",
      "method": "GET",
      "queryParams": {
        "userId": "{{userId}}"
      },
      "headers": {
        "Accept": "application/json"
      }
    },
    "createPost": {
      "url": "/posts",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      "body": {
        "title": "Test Post",
        "body": "This is a test post",
        "userId": 1
      }
    }
  }
}
```

Run a request from this configuration:

```bash
lunge run -c config.json -e dev -r getUser
```

## Variable Extraction Example

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://jsonplaceholder.typicode.com"
    }
  },
  "requests": {
    "createPost": {
      "url": "/posts",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "title": "Test Post",
        "body": "This is a test post",
        "userId": 1
      },
      "extract": {
        "postId": "$.id"
      }
    },
    "getPost": {
      "url": "/posts/{{postId}}",
      "method": "GET",
      "headers": {
        "Accept": "application/json"
      }
    }
  },
  "suites": {
    "postFlow": {
      "requests": ["createPost", "getPost"]
    }
  }
}
```

Run this suite:

```bash
lunge run -c config.json -e dev -s postFlow
```

## Testing Example

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://jsonplaceholder.typicode.com",
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
    },
    "getPosts": {
      "url": "/posts",
      "method": "GET",
      "queryParams": {
        "userId": "{{userId}}"
      },
      "headers": {
        "Accept": "application/json"
      }
    }
  },
  "suites": {
    "apiTests": {
      "requests": ["getUser", "getPosts"],
      "tests": [
        {
          "name": "User API Test",
          "request": "getUser",
          "assertions": [
            { "status": 200 },
            { "responseTime": "<1000" },
            { "header": "Content-Type", "contains": "application/json" },
            { "path": "$.name", "exists": true },
            { "path": "$.email", "matches": ".*@.*\\..*" },
            { "path": "$.address.city", "exists": true }
          ]
        },
        {
          "name": "Posts API Test",
          "request": "getPosts",
          "assertions": [
            { "status": 200 },
            { "path": "$", "isArray": true },
            { "path": "$", "minLength": 1 },
            { "path": "$[0].userId", "equals": "{{userId}}" }
          ]
        }
      ]
    }
  }
}
```

Run these tests:

```bash
lunge test -c config.json -e dev -s apiTests
```

## JSON Schema Validation Example

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://jsonplaceholder.typicode.com"
    }
  },
  "requests": {
    "getUser": {
      "url": "/users/1",
      "method": "GET",
      "headers": {
        "Accept": "application/json"
      }
    }
  },
  "schemas": {
    "userSchema": {
      "type": "object",
      "required": ["id", "name", "email"],
      "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" },
        "email": { "type": "string", "format": "email" },
        "address": {
          "type": "object",
          "properties": {
            "street": { "type": "string" },
            "suite": { "type": "string" },
            "city": { "type": "string" },
            "zipcode": { "type": "string" },
            "geo": {
              "type": "object",
              "properties": {
                "lat": { "type": "string" },
                "lng": { "type": "string" }
              }
            }
          }
        }
      }
    }
  },
  "suites": {
    "schemaTests": {
      "requests": ["getUser"],
      "tests": [
        {
          "name": "User Schema Test",
          "request": "getUser",
          "assertions": [
            { "status": 200 },
            { "schema": "userSchema" }
          ]
        }
      ]
    }
  }
}
```

Run schema validation:

```bash
lunge test -c config.json -e dev -s schemaTests
```

## Authentication Examples

### Basic Auth

```json
"requests": {
  "getProtectedResource": {
    "url": "/protected",
    "method": "GET",
    "auth": {
      "type": "basic",
      "username": "{{username}}",
      "password": "{{password}}"
    }
  }
}
```

### Bearer Token

```json
"requests": {
  "login": {
    "url": "/auth/login",
    "method": "POST",
    "body": {
      "username": "{{username}}",
      "password": "{{password}}"
    },
    "extract": {
      "token": "$.token"
    }
  },
  "getProtectedResource": {
    "url": "/protected",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{token}}"
    }
  }
}
```

## Complex Workflow Example

This example demonstrates a complete workflow with authentication, data creation, and validation:

```json
{
  "environments": {
    "dev": {
      "baseUrl": "https://api.example.com",
      "variables": {
        "username": "testuser",
        "password": "password123"
      }
    }
  },
  "requests": {
    "login": {
      "url": "/auth/login",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "username": "{{username}}",
        "password": "{{password}}"
      },
      "extract": {
        "token": "$.token",
        "userId": "$.user.id"
      }
    },
    "createProject": {
      "url": "/projects",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "Authorization": "Bearer {{token}}"
      },
      "body": {
        "name": "Test Project",
        "description": "This is a test project"
      },
      "extract": {
        "projectId": "$.id"
      }
    },
    "getProject": {
      "url": "/projects/{{projectId}}",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{token}}"
      }
    },
    "addTask": {
      "url": "/projects/{{projectId}}/tasks",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "Authorization": "Bearer {{token}}"
      },
      "body": {
        "title": "Test Task",
        "description": "This is a test task",
        "dueDate": "2023-12-31"
      },
      "extract": {
        "taskId": "$.id"
      }
    },
    "getTask": {
      "url": "/tasks/{{taskId}}",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{token}}"
      }
    }
  },
  "suites": {
    "projectWorkflow": {
      "requests": ["login", "createProject", "getProject", "addTask", "getTask"],
      "tests": [
        {
          "name": "Login Test",
          "request": "login",
          "assertions": [
            { "status": 200 },
            { "path": "$.token", "exists": true }
          ]
        },
        {
          "name": "Create Project Test",
          "request": "createProject",
          "assertions": [
            { "status": 201 },
            { "path": "$.id", "exists": true },
            { "path": "$.name", "equals": "Test Project" }
          ]
        },
        {
          "name": "Get Project Test",
          "request": "getProject",
          "assertions": [
            { "status": 200 },
            { "path": "$.id", "equals": "{{projectId}}" }
          ]
        },
        {
          "name": "Add Task Test",
          "request": "addTask",
          "assertions": [
            { "status": 201 },
            { "path": "$.id", "exists": true }
          ]
        },
        {
          "name": "Get Task Test",
          "request": "getTask",
          "assertions": [
            { "status": 200 },
            { "path": "$.id", "equals": "{{taskId}}" },
            { "path": "$.title", "equals": "Test Task" }
          ]
        }
      ]
    }
  }
}
```

Run this workflow:

```bash
lunge test -c config.json -e dev -s projectWorkflow
```

These examples demonstrate the flexibility and power of Lunge for API testing and automation. You can adapt them to your specific needs and build upon them to create more complex workflows.