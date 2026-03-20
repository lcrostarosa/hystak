package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func (a *cliApp) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registry servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			servers := a.svc.ListServers()
			if len(servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No servers in registry.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTRANSPORT\tCOMMAND/URL")
			for _, s := range servers {
				fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Transport, s.Target())
			}
			return w.Flush()
		},
	}
}
