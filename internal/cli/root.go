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
		Use:   "hystak",
		Short: "MCP server configuration manager",
		Long:  "hystak manages MCP server configurations from a central registry and deploys them to MCP client config files.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgDir == "" {
				cfgDir = config.ConfigDir()
			}
			if err := os.MkdirAll(cfgDir, 0o755); err != nil {
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
			if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
				return cmd.Help()
			}
			m := tui.NewApp(app.svc)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&cfgDir, "config-dir", "", "config directory (default: ~/.config/hystak)")

	root.AddCommand(app.newListCmd())
	root.AddCommand(app.newSyncCmd())
	root.AddCommand(app.newImportCmd())
	root.AddCommand(app.newOverrideCmd())
	root.AddCommand(app.newDiffCmd())
	root.AddCommand(app.newRunCmd())
	root.AddCommand(newVersionCmd(version, commit, date))

	return root
}

// Execute runs the CLI with the given version info.
func Execute(version, commit, date string) {
	if err := newRootCmd(version, commit, date).Execute(); err != nil {
		os.Exit(1)
	}
}
