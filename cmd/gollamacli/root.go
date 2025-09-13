// cmd/gollamacli/root.go
package gollamacli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base Cobra command for the gollamacli application.
// All subcommands are attached to this root to form the complete CLI.
var rootCmd = &cobra.Command{
	Use:   "gollamacli",
	Short: "gollamacli",
	Long:  `gollamacli`,
}

// Execute runs the root Cobra command and all registered subcommands.
// It prints any returned error and exits the process with a non-zero
// status code on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
}
