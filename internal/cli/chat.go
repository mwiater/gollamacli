// cmd/gollamacli/chat.go
package gollamacli

import (
	"github.com/mwiater/gollamacli/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startGUI = cli.StartGUI

// Declare a variable to store the config file path.
// This is not strictly necessary if you only access via viper,
// but it's common practice with StringVar.
var cfgFile string

// chatCmd represents the 'chat' command.
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a chat session",
	Long:  `The 'chat' command starts an interactive chat session with a large language model.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath := viper.GetString("config")
		startGUI(configPath)
	},
}

func init() {
	// 1. Add the command to the root
	rootCmd.AddCommand(chatCmd)

	// 2. Define the string flag 'config'
	// StringVarP: Target variable, Flag name, Shorthand (e.g., "c"), Default value, Description
	chatCmd.Flags().StringVarP(&cfgFile, "config", "c", "config.json", "config file (e.g., config.Authors.json)")

	// 3. Bind the Cobra flag to Viper
	// The key in viper will be "config"
	viper.BindPFlag("config", chatCmd.Flags().Lookup("config"))
}
