// cmd/gollamacli/delete_models.go

package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// deleteModelsCmd represents the 'delete models' subcommand.
var deleteModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Delete all models not in the config.json file",
	Long:  `The 'models' subcommand deletes all models not in the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.DeleteModels()
	},
}

// init adds the deleteModelsCmd to the deleteCmd.
func init() {
	deleteCmd.AddCommand(deleteModelsCmd)
}
