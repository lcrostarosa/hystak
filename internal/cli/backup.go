package cli

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/lcrostarosa/hystak/internal/backup"
	"github.com/spf13/cobra"
)

func (a *cliApp) newBackupCmd() *cobra.Command {
	var (
		all  bool
		list bool
	)

	cmd := &cobra.Command{
		Use:   "backup [project]",
		Short: "Back up client config files",
		Long:  "Create backups of MCP client config files for a project, or list existing backups.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return a.listBackups(cmd, args)
			}

			if all {
				projects := a.svc.ListProjects()
				if len(projects) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No projects found.")
					return nil
				}
				for _, proj := range projects {
					entries, err := a.svc.BackupConfigs(proj.Name)
					if err != nil {
						return err
					}
					out := cmd.OutOrStdout()
					_, _ = fmt.Fprintf(out, "Project: %s\n", proj.Name)
					for _, e := range entries {
						_, _ = fmt.Fprintf(out, "  backed up → %s\n", e.BackupPath)
					}
					if len(entries) == 0 {
						_, _ = fmt.Fprintln(out, "  no configs to back up")
					}
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("project name required (or use --all)")
			}

			entries, err := a.svc.BackupConfigs(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, e := range entries {
				_, _ = fmt.Fprintf(out, "backed up → %s\n", e.BackupPath)
			}
			if len(entries) == 0 {
				_, _ = fmt.Fprintln(out, "no configs to back up")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "back up all projects")
	cmd.Flags().BoolVar(&list, "list", false, "list available backups")

	return cmd
}

func (a *cliApp) listBackups(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) > 0 {
		entries, e := a.svc.ListBackups(args[0])
		if e != nil {
			return e
		}
		if len(entries) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No backups for project %q.\n", args[0])
			return nil
		}
		printBackupTable(cmd, entries)
		return nil
	}

	entries, err := a.svc.ListAllBackups()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No backups found.")
		return nil
	}
	printBackupTable(cmd, entries)
	return nil
}

func printBackupTable(cmd *cobra.Command, entries []backup.BackupEntry) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TIMESTAMP\tCLIENT\tSCOPE\tPATH")

	// Sort newest first.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.ClientType,
			e.Scope,
			e.BackupPath,
		)
	}
	_ = w.Flush()
}
