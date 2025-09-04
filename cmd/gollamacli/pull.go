// cmd/gollamacli/pull.go

package gollamacli

import (
	"github.com/spf13/cobra"
)

// pullCmd represents the 'pull' command.
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Group commands for pulling resources",
	Long:  `The 'pull' command is used to group subcommands that provide different ways to pull resources or information related to gollamacli.`,
}

// init adds the pull command to the root command.
func init() {
	rootCmd.AddCommand(pullCmd)
}
