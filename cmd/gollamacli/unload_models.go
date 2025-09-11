package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// unloadModelsCmd represents the 'unload models' subcommand.
var unloadModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Unload all loaded models on each host",
	Long:  `The 'models' subcommand unloads all loaded models on each host.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.UnloadModels()
	},
}

// init adds the unloadModelsCmd to the root command.
func init() {
	unloadCmd.AddCommand(unloadModelsCmd)
}
