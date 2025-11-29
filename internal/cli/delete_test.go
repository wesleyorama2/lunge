package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestDeleteCommand(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request method is DELETE
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		// Check headers if they were set
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Errorf("Expected X-Test-Header to be 'test-value', got '%s'", r.Header.Get("X-Test-Header"))
		}

		// Return a success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","message":"Resource deleted"}`))
	}))
	defer server.Close()

	// Create a new root command for testing to avoid global state issues
	rootCmd := &cobra.Command{Use: "lunge"}

	// Reset and re-add the delete command
	deleteCmd.ResetFlags()
	deleteCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP headers to include (can be used multiple times)")
	deleteCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	deleteCmd.Flags().Bool("no-color", false, "Disable colored output")
	deleteCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	deleteCmd.Flags().String("format", "", "Output format (text, json, yaml, junit)")

	rootCmd.AddCommand(deleteCmd)

	// Set up command arguments
	rootCmd.SetArgs([]string{"delete", server.URL, "--header", "X-Test-Header:test-value", "--verbose=false", "--no-color"})

	// Execute the command
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Error executing delete command: %v", err)
	}
}
