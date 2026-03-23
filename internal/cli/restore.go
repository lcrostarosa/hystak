package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/hystak/hystak/internal/backup"
	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/project"
	"github.com/spf13/cobra"
)

var restoreIndex int

var restoreCmd = &cobra.Command{
	Use:         "restore <project>",
	Short:       "Restore project config from backup",
	Long:        "Restore a client config file from a timestamped backup. Interactive by default, or use --index for non-interactive.",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"mutates": "true"},
	RunE:        runRestore,
}

var undoCmd = &cobra.Command{
	Use:         "undo [project]",
	Short:       "Undo last sync",
	Long:        "Restore from the most recent automatic backup. Shortcut for 'hystak restore <project> --index 0'.",
	Args:        cobra.MaximumNArgs(1),
	Annotations: map[string]string{"mutates": "true"},
	RunE:        runUndo,
}

func init() {
	restoreCmd.Flags().IntVar(&restoreIndex, "index", -1, "restore by index (0 = most recent)")
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(undoCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	mgr := backup.NewDefaultManager()

	entries, err := mgr.List(projectName)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no backups found for project %q", projectName)
	}

	var selected backup.Entry
	if restoreIndex >= 0 {
		if restoreIndex >= len(entries) {
			return fmt.Errorf("index %d out of range (0-%d)", restoreIndex, len(entries)-1)
		}
		selected = entries[restoreIndex]
	} else {
		// Interactive selection (S-068)
		fmt.Fprintln(cmd.OutOrStdout(), "Available backups:")
		for i, e := range entries {
			ts := e.Timestamp.Format("2006-01-02 15:04:05")
			fmt.Fprintf(cmd.OutOrStdout(), "  [%d] %s  %s  %s\n", i, ts, e.Scope, e.Path)
		}
		fmt.Fprint(cmd.OutOrStdout(), "\nSelect backup index: ")

		reader := bufio.NewReader(cmd.InOrStdin())
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		var idx int
		if _, err := fmt.Sscanf(strings.TrimSpace(input), "%d", &idx); err != nil {
			return fmt.Errorf("invalid index: %w", err)
		}
		if idx < 0 || idx >= len(entries) {
			return fmt.Errorf("index %d out of range (0-%d)", idx, len(entries)-1)
		}
		selected = entries[idx]
	}

	// Resolve target path
	targetPath, err := resolveRestoreTarget(projectName, selected.Scope)
	if err != nil {
		return err
	}

	// Confirmation
	ts := selected.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Fprintf(cmd.OutOrStdout(), "Restore %s (%s) to %s? [y/N]: ", ts, selected.Scope, targetPath)

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil // EOF = abort
	}
	choice := strings.TrimSpace(strings.ToLower(input))
	if choice != "y" && choice != "yes" {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return nil
	}

	if err := mgr.Restore(selected.Path, targetPath); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Restored %s\n", targetPath)
	return nil
}

func runUndo(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project name is required")
	}

	projectName := args[0]
	mgr := backup.NewDefaultManager()

	entry, found, err := mgr.LatestForProject(projectName)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no backups found for project %q", projectName)
	}

	targetPath, err := resolveRestoreTarget(projectName, entry.Scope)
	if err != nil {
		return err
	}

	if err := mgr.Restore(entry.Path, targetPath); err != nil {
		return err
	}

	ts := entry.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Fprintf(cmd.OutOrStdout(), "Restored %s from %s backup\n", targetPath, ts)
	return nil
}

func resolveRestoreTarget(projectName, scope string) (string, error) {
	projStore, err := project.LoadDefault()
	if err != nil {
		return "", fmt.Errorf("loading projects: %w", err)
	}

	proj, ok := projStore.Get(projectName)
	if !ok {
		return "", fmt.Errorf("project %q not found", projectName)
	}

	dep, ok := deploy.NewDeployer(model.ClientClaudeCode)
	if !ok {
		return "", fmt.Errorf("no deployer for %s", model.ClientClaudeCode)
	}

	switch scope {
	case "mcp":
		return dep.ConfigPath(proj.Path), nil
	case "settings":
		return proj.Path + "/.claude/settings.local.json", nil
	default:
		return dep.ConfigPath(proj.Path), nil
	}
}
