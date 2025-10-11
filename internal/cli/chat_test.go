// cmd/gollamacli/chat_test.go
package gollamacli

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
)

func TestChatCmd(t *testing.T) {
	// Redirect stdout to a buffer
	b := new(bytes.Buffer)
	rootCmd.SetOut(b)
	rootCmd.SetErr(b)

	originalStartGUI := startGUI
	defer func() { startGUI = originalStartGUI }()

	var receivedPath string
	startCalled := false
	startGUI = func(path string) {
		startCalled = true
		receivedPath = path
	}

	viper.Set("config", "test-config.json")
	defer viper.Set("config", nil)

	// Execute the command
	chatCmd.Run(chatCmd, []string{})

	if !startCalled {
		t.Fatal("expected startGUI to be invoked")
	}

	if receivedPath != "test-config.json" {
		t.Fatalf("expected config path 'test-config.json', got %q", receivedPath)
	}
}
