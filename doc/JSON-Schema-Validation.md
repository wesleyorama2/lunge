# JSON Schema Validation

Lunge supports validating API responses against JSON Schema definitions. This provides a powerful way to ensure that responses match expected structures and data types.

## What is JSON Schema?

JSON Schema is a vocabulary that allows you to annotate and validate JSON documents. It helps you:

- Validate required fields
- Ensure correct data types
- Validate string formats (email, date, etc.)
- Validate numeric ranges
- Define complex object structures
- And much more

## Defining Schemas

Schemas are defined in the `schemas` section of your configuration file:

```json
"schemas": {
  "userSchema": {
    "type": "object",
    "required": ["id", "name", "email"],
    "properties": {
      "id": { "type": "integer" },
      "name": { "type": "string" },
      "email": { "type": "string", "format": "email" },
      "age": { "type": "integer", "minimum": 18 },
      "address": {
        "type": "object",
        "properties": {
          "street": { "type": "string" },
          "city": { "type": "string" },
          "zipcode": { "type": "string", "pattern": "^\\d{5}(-\\d{4})?$" }
        }
      },
      "tags": {
        "type": "array",
        "items": { "type": "string" }
      }
    }
  }
}
```

## Using Schemas in Tests

To validate a response against a schema, use the `schema` assertion in your test:

```json
"tests": [
  {
    "name": "User API Test",
    "request": "getUser",
    "assertions": [
      { "status": 200 },
      { "schema": "userSchema" }
    ]
  }
]
```

This will validate the entire response body against the specified schema.

## Schema Validation for Nested Data

You can also validate specific parts of the response using JSONPath with schema validation:

```json
"assertions": [
  { "path": "$.user", "schema": "userSchema" },
  { "path": "$.metadata", "schema": "metadataSchema" }
]
```

## Common Schema Validations

### Required Fields

Ensure that specific fields are present:

```json
{
  "type": "object",
  "required": ["id", "name", "email"]
}
```

### Data Types

Validate the data type of fields:

```json
{
  "properties": {
    "id": { "type": "integer" },
    "name": { "type": "string" },
    "active": { "type": "boolean" },
    "score": { "type": "number" }
  }
}
```

### String Formats

Validate string formats:

```json
{
  "properties": {
    "email": { "type": "string", "format": "email" },
    "website": { "type": "string", "format": "uri" },
    "date": { "type": "string", "format": "date" }
  }
}
```

### Numeric Ranges

Validate numeric ranges:

```json
{
  "properties": {
    "age": { "type": "integer", "minimum": 18, "maximum": 100 },
    "score": { "type": "number", "exclusiveMinimum": 0, "exclusiveMaximum": 10 }
  }
}
```

### String Patterns

Validate strings against regular expressions:

```json
{
  "properties": {
    "zipcode": { "type": "string", "pattern": "^\\d{5}(-\\d{4})?$" },
    "phone": { "type": "string", "pattern": "^\\+?[1-9]\\d{1,14}$" }
  }
}
```

### Arrays

Validate arrays and their items:

```json
{
  "properties": {
    "tags": {
      "type": "array",
      "items": { "type": "string" },
      "minItems": 1,
      "maxItems": 10,
      "uniqueItems": true
    }
  }
}
```

### Enums

Restrict values to a set of possibilities:

```json
{
  "properties": {
    "status": {
      "type": "string",
      "enum": ["pending", "active", "suspended", "deleted"]
    }
  }
}
```

## Schema Validation Results

When schema validation fails, Lunge provides detailed error messages indicating which part of the schema was violated:

```
✗ ASSERTION FAILED: Schema validation failed for userSchema:
  - $.age: Expected integer, got string
  - $.email: String does not match email format
  - $.address.zipcode: String does not match pattern ^\\d{5}(-\\d{4})?$
```

## External Schema References

You can also reference external schema files:

```json
"schemas": {
  "userSchema": {
    "$ref": "file:///path/to/user-schema.json"
  }
}
```

## Next Steps

- [Examples](./Examples.md) - See examples of schema validation in action
- [Testing](./Testing.md) - Learn more about testing and assertions