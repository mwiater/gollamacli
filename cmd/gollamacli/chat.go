// cmd/gollamacli/chat.go

package gollamacli

import (
	"github.com/mwiater/gollamacli/cli"
	"github.com/spf13/cobra"
)

// chatCmd represents the 'chat' command.
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a chat session",
	Long:  `The 'chat' command starts an interactive chat session with a large language model.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli.StartGUI()
	},
}

// init adds the chat command to the root command.
func init() {
	rootCmd.AddCommand(chatCmd)
}
