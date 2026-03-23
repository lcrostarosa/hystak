package cli

import "github.com/spf13/cobra"

// S-006: Re-run setup wizard on demand.
var setupCmd = &cobra.Command{
	Use:         "setup",
	Short:       "Re-run first-time setup",
	Long:        "Re-run the first-run flow: keybinding selection, config scanning, and server import.",
	Annotations: map[string]string{"mutates": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFirstRunFlow(cmd)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
