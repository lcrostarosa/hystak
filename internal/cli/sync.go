package cli

import (
	"fmt"
	"sort"

	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/spf13/cobra"
)

func (a *cliApp) newSyncCmd() *cobra.Command {
	var (
		all         bool
		profileName string
	)

	cmd := &cobra.Command{
		Use:   "sync [project]",
		Short: "Sync server configs to client config files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				results, err := a.svc.SyncAll()
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

			projectName := args[0]

			// If --profile specified, sync that specific profile.
			if profileName != "" {
				results, err := a.svc.SyncProfile(projectName, profileName)
				if err != nil {
					return err
				}
				printSyncResults(cmd, projectName, results)
				return nil
			}

			// Default: sync using active profile (or legacy direct assignments).
			results, err := a.svc.SyncProject(projectName)
			if err != nil {
				return err
			}
			printSyncResults(cmd, projectName, results)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "sync all projects")
	cmd.Flags().StringVar(&profileName, "profile", "", "sync using a specific profile")

	return cmd
}

func printSyncResults(cmd *cobra.Command, project string, results []service.SyncResult) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Project: %s\n", project)
	for _, r := range results {
		_, _ = fmt.Fprintf(out, "  %-20s %s\n", r.ServerName, r.Action)
	}
}
