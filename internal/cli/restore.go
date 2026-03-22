package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lcrostarosa/hystak/internal/backup"
	"github.com/spf13/cobra"
)

func (a *cliApp) newRestoreCmd() *cobra.Command {
	var (
		global bool
		index  int
	)

	cmd := &cobra.Command{
		Use:   "restore [project]",
		Short: "Restore a client config from backup",
		Long:  "Restore a previously backed-up MCP client config file.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !global && len(args) == 0 {
				return fmt.Errorf("project name required (or use --global)")
			}

			var projectName string
			if len(args) > 0 {
				projectName = args[0]
			}

			// Get entries.
			var entries []backupEntryWrapper
			if global {
				all, err := a.svc.ListAllBackups()
				if err != nil {
					return err
				}
				for _, e := range all {
					if e.Scope == "global" {
						entries = append(entries, backupEntryWrapper{e, ""})
					}
				}
			} else {
				list, err := a.svc.ListBackups(projectName)
				if err != nil {
					return err
				}
				for _, e := range list {
					entries = append(entries, backupEntryWrapper{e, projectName})
				}
			}

			if len(entries) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No backups found.")
				return nil
			}

			reader := bufio.NewReader(cmd.InOrStdin())

			// Select entry.
			var selected int
			if cmd.Flags().Changed("index") {
				if index < 0 || index >= len(entries) {
					return fmt.Errorf("index %d out of range (0-%d)", index, len(entries)-1)
				}
				selected = index
			} else {
				out := cmd.OutOrStdout()
				_, _ = fmt.Fprintln(out, "Available backups:")
				for i, e := range entries {
					_, _ = fmt.Fprintf(out, "  [%d] %s  %s  %s\n",
						i,
						e.Timestamp.Format("2006-01-02 15:04:05"),
						e.ClientType,
						e.BackupPath,
					)
				}
				_, _ = fmt.Fprint(out, "Select backup to restore: ")

				line, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				line = strings.TrimSpace(line)
				n := 0
				for _, c := range line {
					if c < '0' || c > '9' {
						return fmt.Errorf("invalid selection: %q", line)
					}
					n = n*10 + int(c-'0')
				}
				if n < 0 || n >= len(entries) {
					return fmt.Errorf("index %d out of range (0-%d)", n, len(entries)-1)
				}
				selected = n
			}

			entry := entries[selected]

			// Confirmation.
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Restore %s → %s? [y/N] ", entry.BackupPath, entry.SourcePath)

			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			if strings.TrimSpace(strings.ToLower(line)) != "y" {
				_, _ = fmt.Fprintln(out, "Cancelled.")
				return nil
			}

			if err := a.svc.RestoreBackup(entry.BackupEntry); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(out, "Restored → %s\n", entry.SourcePath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "restore global-scope backup")
	cmd.Flags().IntVar(&index, "index", 0, "select backup by index (0 = most recent)")

	return cmd
}

// backupEntryWrapper pairs a backup entry with its project name for display.
type backupEntryWrapper struct {
	backup.BackupEntry
	projectName string
}
