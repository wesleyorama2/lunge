package jsonpath

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// Extract extracts a value from a JSON string using a JSONPath expression
func Extract(json string, path string) (string, error) {
	// Handle empty JSON
	if json == "" {
		return "", fmt.Errorf("empty JSON string")
	}

	// Handle empty path
	if path == "" {
		return "", fmt.Errorf("empty JSONPath expression")
	}

	// Convert JSONPath to gjson path format
	// JSONPath: $.users[0].name
	// gjson:    users.0.name
	gpath := convertToGjsonPath(path)

	// Extract the value
	result := gjson.Get(json, gpath)
	if !result.Exists() {
		return "", fmt.Errorf("path not found: %s", path)
	}

	// Handle null values
	if result.Type == gjson.Null {
		return "null", nil
	}

	// Return the value as a string
	return result.String(), nil
}

// ExtractMultiple extracts multiple values from a JSON string using a map of JSONPath expressions
func ExtractMultiple(json string, paths map[string]string) (map[string]string, error) {
	// Handle empty JSON
	if json == "" {
		return nil, fmt.Errorf("empty JSON string")
	}

	// Handle empty paths
	if len(paths) == 0 {
		return nil, fmt.Errorf("no JSONPath expressions provided")
	}

	// Extract each value
	results := make(map[string]string)
	var errors []string

	for name, path := range paths {
		value, err := Extract(json, path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		results[name] = value
	}

	// Return error if any extractions failed
	if len(errors) > 0 {
		return results, fmt.Errorf("extraction errors: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// convertToGjsonPath converts a JSONPath expression to a gjson path format
func convertToGjsonPath(path string) string {
	// Special case for root path
	if path == "$" {
		return "@this"
	}

	// Remove $ prefix
	path = strings.TrimPrefix(path, "$")

	// Handle root path
	if path == "" {
		return "@this"
	}

	// Remove leading dot if present
	path = strings.TrimPrefix(path, ".")

	// Handle bracket notation with single quotes: $['name']
	if strings.Contains(path, "['") {
		path = strings.Replace(path, "['", "", -1)
		path = strings.Replace(path, "']", "", -1)
	}

	// Handle bracket notation with double quotes: $["name"]
	if strings.Contains(path, "[\"") {
		path = strings.Replace(path, "[\"", "", -1)
		path = strings.Replace(path, "\"]", "", -1)
	}

	// Handle direct array access at the root level: $[0] -> 0
	if strings.HasPrefix(path, "[") && strings.Contains(path, "]") {
		endBracket := strings.Index(path, "]")
		if endBracket > 1 {
			index := path[1:endBracket]
			rest := ""
			if len(path) > endBracket+1 {
				rest = path[endBracket+1:]
			}
			path = index + rest
		}
	}

	// Replace array notation [n] with .n for nested paths
	// This is a simple implementation that doesn't handle all JSONPath features
	for i := 0; i < 10; i++ { // Handle up to 10 levels of nesting
		path = strings.Replace(path, "[", ".", -1)
		path = strings.Replace(path, "]", "", -1)
	}

	return path
}
