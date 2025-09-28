// cmd/gollamacli/sync_models.go
package gollamacli

import (
	"github.com/spf13/cobra"

	"github.com/mwiater/gollamacli/models"
)

// syncModelsCmd implements 'sync models', which deletes models not in the
// configuration and then pulls any missing models across supported hosts.
var syncModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Sync all models from the config.json file",
	Long:  `The 'models' subcommand syncs all models from the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.SyncModels()
	},
}

func init() {
	syncCmd.AddCommand(syncModelsCmd)
}
