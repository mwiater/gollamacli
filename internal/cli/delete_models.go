// cmd/gollamacli/delete_models.go
package gollamacli

import (
	"github.com/mwiater/gollamacli/internal/models"
	"github.com/spf13/cobra"
)

// deleteModelsCmd implements 'delete models', which removes models not listed
// in the configuration from each supported host.
var deleteModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Delete all models not in the config.json file",
	Long:  `The 'models' subcommand deletes all models not in the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.DeleteModels()
	},
}

func init() {
	deleteCmd.AddCommand(deleteModelsCmd)
}
