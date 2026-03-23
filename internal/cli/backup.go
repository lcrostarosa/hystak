package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/backup"
	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/project"
	"github.com/spf13/cobra"
)

var (
	backupAll  bool
	backupList bool
)

var backupCmd = &cobra.Command{
	Use:   "backup [project]",
	Short: "Backup project configs",
	Long:  "Snapshot client config files to ~/.hystak/backups/ with timestamps.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBackup,
}

func init() {
	backupCmd.Flags().BoolVar(&backupAll, "all", false, "backup all projects")
	backupCmd.Flags().BoolVar(&backupList, "list", false, "list backups for a project (S-067)")
	rootCmd.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, args []string) error {
	mgr := backup.NewDefaultManager()

	if backupList {
		return runBackupList(cmd, mgr, args)
	}

	if !backupAll && len(args) == 0 {
		return fmt.Errorf("project name is required (or use --all)")
	}

	projStore, err := project.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading projects: %w", err)
	}

	dep, ok := deploy.NewDeployer(model.ClientClaudeCode)
	if !ok {
		return fmt.Errorf("no deployer for %s", model.ClientClaudeCode)
	}

	if backupAll {
		for _, p := range projStore.List() {
			configPath := dep.ConfigPath(p.Path)
			path, err := mgr.Backup(p.Name, configPath)
			if err != nil {
				if _, wErr := fmt.Fprintf(cmd.ErrOrStderr(), "  %s: error: %v\n", p.Name, err); wErr != nil {
					return wErr
				}
				continue
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", p.Name, path); err != nil {
				return err
			}
		}
		return nil
	}

	projectName := args[0]
	proj, ok := projStore.Get(projectName)
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	configPath := dep.ConfigPath(proj.Path)
	path, err := mgr.Backup(projectName, configPath)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Backed up to %s\n", path); err != nil {
		return err
	}

	// Prune old backups (S-070)
	userCfg, err := config.LoadUserConfig()
	if err != nil {
		return nil // non-blocking
	}
	pruned, err := mgr.Prune(userCfg.MaxBackups)
	if err != nil {
		if _, wErr := fmt.Fprintf(cmd.ErrOrStderr(), "warning: pruning backups: %v\n", err); wErr != nil {
			return wErr
		}
	}
	if pruned > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Pruned %d old backup(s)\n", pruned); err != nil {
			return err
		}
	}
	return nil
}

func runBackupList(cmd *cobra.Command, mgr *backup.Manager, args []string) error {
	projectName := ""
	if len(args) > 0 {
		projectName = args[0]
	}

	entries, err := mgr.List(projectName)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No backups found."); err != nil {
			return err
		}
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "TIMESTAMP\tPROJECT\tSCOPE\tPATH"); err != nil {
		return err
	}
	for _, e := range entries {
		ts := e.Timestamp.Format("2006-01-02 15:04:05")
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ts, e.Project, e.Scope, e.Path); err != nil {
			return err
		}
	}
	return w.Flush()
}
