package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *cliApp) newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <project>",
		Short: "Show drift diff between deployed and expected configs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			diff, err := a.svc.Diff(args[0])
			if err != nil {
				return err
			}
			if diff == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No drift detected.")
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), diff)
			return nil
		},
	}
}
