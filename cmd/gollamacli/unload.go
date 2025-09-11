package gollamacli

import (
	"github.com/spf13/cobra"
)

// unloadCmd represents the 'unload' command
var unloadCmd = &cobra.Command{
	Use:   "unload",
	Short: "Unload a resource",
	Long:  `The 'unload' command unloads a specified resource.`,
}

// init adds the unloadCmd to the rootCmd.
func init() {
	rootCmd.AddCommand(unloadCmd)
}
