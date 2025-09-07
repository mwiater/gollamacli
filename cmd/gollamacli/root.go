package gollamacli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for the gollamacli application.
// It provides the core command line interface for building and running instrumented versions of Go applications.
// The command holds the primary configuration, usage details, and a detailed description of the application's purpose.
var rootCmd = &cobra.Command{
	Use:   "gollamacli",
	Short: "gollamacli",
	Long:  `gollamacli`,
}

// Execute executes the root command along with any registered subcommands.
// If the command execution results in an error, the error is printed and the program exits
// with a non-zero status code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// init initializes the root command's configuration.
// This function is reserved for setting up additional top-level flags or configurations.
// Since subcommands are self-registered in their respective files, no further initialization
// is required here.
func init() {
}
