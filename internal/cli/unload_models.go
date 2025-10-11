// cmd/gollamacli/unload_models.go
package gollamacli

import (
	"github.com/mwiater/gollamacli/internal/models"
	"github.com/spf13/cobra"
)

// unloadModelsCmd implements 'unload models', which unloads all currently
// loaded models on each supported host.
var unloadModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Unload all loaded models on each host",
	Long:  `The 'models' subcommand unloads all loaded models on each host.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.UnloadModels()
	},
}

func init() {
	unloadCmd.AddCommand(unloadModelsCmd)
}
