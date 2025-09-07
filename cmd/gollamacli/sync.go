package gollamacli

import (
	"github.com/spf13/cobra"
)

// syncCmd represents the 'sync' command.
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Group commands for syncing resources",
	Long:  `The 'sync' command is used to group subcommands that provide different ways to sync resources or information related to gollamacli.`,
}

// init adds the sync command to the root command.
func init() {
	rootCmd.AddCommand(syncCmd)
}
