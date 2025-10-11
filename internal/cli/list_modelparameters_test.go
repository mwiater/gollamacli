// cmd/gollamacli/list_modelparameters_test.go
package gollamacli

import (
	"bytes"
	"testing"
)

func TestListModelParametersCmd(t *testing.T) {
	// Redirect stdout to a buffer
	b := new(bytes.Buffer)
	rootCmd.SetOut(b)

	// Execute the command
	listModelParametersCmd.Run(listModelParametersCmd, []string{})

	// For now, just check that the command runs without error.
	// A more robust test would involve mocking the models.ListModelParameters function
	// and checking the output.
}
