package gollamacli

import (
	"github.com/mwiater/gollamacli/models"
	"github.com/spf13/cobra"
)

// pullModelsCmd represents the 'pull models' subcommand.
var pullModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Pull all models from the config.json file",
	Long:  `The 'models' subcommand pulls all models from the config.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		models.PullModels()
	},
}

// init adds the pullModelsCmd to the pullCmd.
func init() {
	pullCmd.AddCommand(pullModelsCmd)
}
