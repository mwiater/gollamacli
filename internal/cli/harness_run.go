// cmd/gollamacli/harness_run.go
package gollamacli

import (
	"github.com/spf13/cobra"

	"github.com/mwiater/gollamacli/internal/harness"
)

// harnessRunCmd
var harnessRunCmd = &cobra.Command{
	Use:   "run",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		harness.Run()
	},
}

func init() {
	harnessCmd.AddCommand(harnessRunCmd)
}
