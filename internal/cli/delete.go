// cmd/gollamacli/delete.go
package gollamacli

import (
	"github.com/spf13/cobra"
)

// deleteCmd represents the 'delete' command group for deleting resources.
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Group commands for deleting resources",
	Long:  `The 'delete' command groups subcommands that delete resources or information related to gollamacli.`,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
