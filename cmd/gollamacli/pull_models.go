// cmd/gollamacli/pull_models.go
package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// pullModelsCmd implements 'pull models', which pulls all configured models
// to each supported host defined in the configuration file.
var pullModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Pull all models from the config.json file",
	Long:  `The 'models' subcommand pulls all models from the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.PullModels()
	},
}

func init() {
	pullCmd.AddCommand(pullModelsCmd)
}
