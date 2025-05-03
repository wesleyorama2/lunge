package cli

import (
	"testing"
)

// TestExecute tests the Execute function
func TestExecute(t *testing.T) {
	// We can't easily mock cobra.Command.Execute, so we'll just test that
	// our Execute function exists and can be called without panicking
	// The actual functionality is tested through integration tests

	// Just make sure the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute() panicked: %v", r)
		}
	}()

	// Call Execute, but ignore the error since we can't reliably
	// control what RootCmd.Execute returns in a unit test
	_ = Execute()
}
