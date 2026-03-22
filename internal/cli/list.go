package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/model"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered MCP servers",
	Long:  "Print a tab-separated table of all MCP servers in the registry.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	svc, err := buildServiceReadOnly()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tTRANSPORT\tCOMMAND/URL"); err != nil {
		return err
	}

	for _, s := range svc.Registry.Servers.List() {
		endpoint := commandOrURL(s)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Transport, endpoint); err != nil {
			return err
		}
	}

	return w.Flush()
}

func commandOrURL(s model.ServerDef) string {
	switch s.Transport {
	case model.TransportSSE, model.TransportHTTP:
		return s.URL
	default:
		return s.Command
	}
}
