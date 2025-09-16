// cmd/gollamacli/list_modelparameters.go
package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// listModelParametersCmd implements 'list modelParameters', which enumerates
// all models on each configured host and prints their current parameters.
var listModelParametersCmd = &cobra.Command{
	Use:   "modelParameters",
	Short: "List parameters for each model on each node",
	Long:  `The 'modelParameters' subcommand iterates models on each configured node and prints their current parameters from /api/show.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.ListModelParameters()
	},
}

func init() {
	listCmd.AddCommand(listModelParametersCmd)
}
