// cmd/gollamacli/list_models.go

package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// listModelsCmd represents the 'list models' subcommand.
var listModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List all models on each node",
	Long:  `The 'models' subcommand lists all models on each node specified in the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.ListModels()
	},
}

// init adds the listModelsCmd to the listCmd.
func init() {
	listCmd.AddCommand(listModelsCmd)
}
