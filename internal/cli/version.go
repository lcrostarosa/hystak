package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		// Skip service initialization for version command.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "hystak %s\n", version)
			fmt.Fprintf(out, "  commit: %s\n", commit)
			fmt.Fprintf(out, "  built:  %s\n", date)
		},
	}
}
