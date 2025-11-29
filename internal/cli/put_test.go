package cli

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestPutCommand(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request method is PUT
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}

		// Check headers if they were set
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Errorf("Expected X-Test-Header to be 'test-value', got '%s'", r.Header.Get("X-Test-Header"))
		}

		// Check Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type to be 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Read and verify the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Error reading request body: %v", err)
		}

		// Parse the JSON body
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			t.Errorf("Error parsing JSON body: %v", err)
		}

		// Check if the body contains the expected data
		if name, ok := data["name"]; !ok || name != "Updated Resource" {
			t.Errorf("Expected body to contain name='Updated Resource', got %v", name)
		}

		// Return a success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "name": "Updated Resource", "updated": true}`))
	}))
	defer server.Close()

	// Create a new root command for testing to avoid global state issues
	rootCmd := &cobra.Command{Use: "lunge"}

	// Reset and re-add the put command
	putCmd.ResetFlags()
	putCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP headers to include (can be used multiple times)")
	putCmd.Flags().StringP("json", "j", "", "JSON data to send in the request body")
	putCmd.Flags().StringP("file", "f", "", "File containing data to send in the request body")
	putCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	putCmd.Flags().Bool("no-color", false, "Disable colored output")
	putCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	putCmd.Flags().String("format", "", "Output format (text, json, yaml, junit)")

	rootCmd.AddCommand(putCmd)

	// Set up command arguments
	rootCmd.SetArgs([]string{"put", server.URL, "--header", "X-Test-Header:test-value", "--json", `{"name": "Updated Resource", "description": "This resource has been updated"}`, "--verbose=false", "--no-color"})

	// Execute the command
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Error executing put command: %v", err)
	}
}
