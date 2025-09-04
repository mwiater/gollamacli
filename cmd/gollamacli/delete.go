// cmd/gollamacli/delete.go

package gollamacli

import (
	"github.com/spf13/cobra"
)

// deleteCmd represents the 'delete' command.
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Group commands for deleting resources",
	Long:  `The 'delete' command is used to group subcommands that provide different ways to delete resources or information related to gollamacli.`,
}

// init adds the delete command to the root command.
func init() {
	rootCmd.AddCommand(deleteCmd)
}
