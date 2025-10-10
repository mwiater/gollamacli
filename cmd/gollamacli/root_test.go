// cmd/gollamacli/root_test.go
package gollamacli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd(t *testing.T) {
	// Redirect stdout to a buffer
	b := new(bytes.Buffer)
	rootCmd.SetOut(b)
	rootCmd.SetErr(b)

	// Execute the command with a non-existent subcommand
	rootCmd.SetArgs([]string{"nonexistent"})
	_, err := rootCmd.ExecuteC()

	// Check if an error is returned
	if err == nil {
		t.Error("Expected an error for a nonexistent command, but got none")
	}

	// Check the output for the expected error message
	expected := "unknown command \"nonexistent\" for \"gollamacli\""
	if !strings.Contains(b.String(), expected) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expected, b.String())
	}
}