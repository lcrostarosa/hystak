package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/lcrostarosa/hystak/internal/config"
	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/lcrostarosa/hystak/internal/tui"
	"github.com/spf13/cobra"
)

// cliApp holds the shared service instance for all subcommands.
type cliApp struct {
	svc *service.Service
}

// newRootCmd builds the full command tree.
func newRootCmd(version, commit, date string) *cobra.Command {
	var cfgDir string
	app := &cliApp{}

	root := &cobra.Command{
		Use:   "hystak [-- claude-args...]",
		Short: "Claude Code launcher with profile management",
		Long: `hystak manages MCP server configurations, skills, hooks, and permissions
from a central registry, syncs them to client config files, and launches
Claude Code with the selected profile.

Run with no arguments to pick a profile interactively, or use subcommands
for non-interactive workflows.

Arguments after -- are forwarded to the claude process.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgDir == "" {
				cfgDir = config.ConfigDir()
			}

			// Migrate from legacy ~/.config/hystak/ if needed.
			if warning, err := config.Migrate(); err != nil {
				return fmt.Errorf("config migration: %w", err)
			} else if warning != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), warning)
			}

			if err := config.EnsureConfigDir(); err != nil {
				return fmt.Errorf("creating config directory: %w", err)
			}
			var err error
			app.svc, err = service.New(cfgDir)
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Split args at -- separator.
			var extraArgs []string
			if dashAt := cmd.ArgsLenAtDash(); dashAt >= 0 {
				extraArgs = args[dashAt:]
				args = args[:dashAt]
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
				return cmd.Help()
			}

			// First-time setup wizard.
			if app.svc.IsEmpty() {
				wizard := tui.NewWizardModel(app.svc)
				wp := tea.NewProgram(wizard, tea.WithAltScreen())
				if _, err := wp.Run(); err != nil {
					return err
				}
			}

			// Show picker.
			picker := tui.NewPickerModel(app.svc)
			p := tea.NewProgram(picker, tea.WithAltScreen())
			result, err := p.Run()
			if err != nil {
				return err
			}

			pickerResult := result.(tui.PickerModel).Result()
			if pickerResult == nil {
				// User cancelled.
				return nil
			}

			if pickerResult.Manage {
				// Launch full management TUI.
				m := tui.NewApp(app.svc)
				mp := tea.NewProgram(m, tea.WithAltScreen())
				_, err := mp.Run()
				return err
			}

			if pickerResult.Project == nil {
				// Launch without profile.
				return launchBare(extraArgs)
			}

			// Sync and launch selected project.
			return app.syncAndLaunch(cmd, *pickerResult.Project, extraArgs, false, false)
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&cfgDir, "config-dir", "", "config directory (default: ~/.hystak)")

	root.AddCommand(app.newListCmd())
	root.AddCommand(app.newSyncCmd())
	root.AddCommand(app.newImportCmd())
	root.AddCommand(app.newOverrideCmd())
	root.AddCommand(app.newDiffCmd())
	root.AddCommand(app.newRunCmd())
	root.AddCommand(app.newBackupCmd())
	root.AddCommand(app.newRestoreCmd())
	root.AddCommand(app.newManageCmd())
	root.AddCommand(app.newSetupCmd())
	root.AddCommand(newVersionCmd(version, commit, date))

	return root
}

// Execute runs the CLI with the given version info.
func Execute(version, commit, date string) {
	if err := newRootCmd(version, commit, date).Execute(); err != nil {
		os.Exit(1)
	}
}
