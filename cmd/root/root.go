package root

import (
	"github.com/celestiaorg/knuu/cmd/api"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "knuu",
	Short: "Knuu CLI",
	Long:  "Knuu CLI provides commands to manage the Knuu API server and its operations.",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(api.NewAPICmd())
}
