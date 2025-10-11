// cmd/gollamacli/harness.go
package gollamacli

import (
	"github.com/spf13/cobra"
)

// harnessCmd
var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(harnessCmd)
}
