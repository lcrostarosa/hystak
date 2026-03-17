package cli

import (
	"fmt"
	"sort"

	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "sync [project]",
		Short: "Sync server configs to client config files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if all {
				results, err := svc.SyncAll()
				if err != nil {
					return err
				}
				names := make([]string, 0, len(results))
				for name := range results {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					printSyncResults(cmd, name, results[name])
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("project name required (or use --all)")
			}

			results, err := svc.SyncProject(args[0])
			if err != nil {
				return err
			}
			printSyncResults(cmd, args[0], results)
			_ = out
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "sync all projects")

	return cmd
}

func printSyncResults(cmd *cobra.Command, project string, results []service.SyncResult) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Project: %s\n", project)
	for _, r := range results {
		fmt.Fprintf(out, "  %-20s %s\n", r.ServerName, r.Action)
	}
}
