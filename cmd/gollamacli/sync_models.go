package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// syncModelsCmd represents the 'sync models' subcommand.
var syncModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Sync all models from the config.json file",
	Long:  `The 'models' subcommand syncs all models from the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.SyncModels()
	},
}

// init adds the syncModelsCmd to the syncCmd.
func init() {
	syncCmd.AddCommand(syncModelsCmd)
}
