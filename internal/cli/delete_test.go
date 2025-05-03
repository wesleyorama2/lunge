package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

	// Set up command arguments and flags
	deleteCmd.SetArgs([]string{server.URL})
	deleteCmd.Flags().Set("header", "X-Test-Header:test-value")
	deleteCmd.Flags().Set("verbose", "false")
	deleteCmd.Flags().Set("no-color", "true")

	// Execute the command
	err := deleteCmd.Execute()
	if err != nil {
		t.Errorf("Error executing delete command: %v", err)
	}
}
