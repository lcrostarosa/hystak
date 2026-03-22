package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables set via ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "hystak %s\ncommit: %s\nbuilt:  %s\n", version, commit, date)
		return err
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
