// cmd/gollamacli/sync.go
package gollamacli

import (
	"github.com/spf13/cobra"
)

// syncCmd represents the 'sync' command group for synchronizing resources.
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Group commands for syncing resources",
	Long:  `The 'sync' command groups subcommands that synchronize resources or information related to gollamacli.`,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
