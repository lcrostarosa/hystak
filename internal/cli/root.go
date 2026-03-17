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

// svc is the shared service instance, initialized in PersistentPreRunE.
var svc *service.Service

// newRootCmd builds the full command tree.
func newRootCmd(version, commit, date string) *cobra.Command {
	var cfgDir string

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
			svc, err = service.New(cfgDir)
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
				return cmd.Help()
			}
			app := tui.NewApp(svc)
			p := tea.NewProgram(app, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&cfgDir, "config-dir", "", "config directory (default: ~/.config/hystak)")

	root.AddCommand(newListCmd())
	root.AddCommand(newSyncCmd())
	root.AddCommand(newImportCmd())
	root.AddCommand(newOverrideCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newVersionCmd(version, commit, date))

	return root
}

// Execute runs the CLI with the given version info.
func Execute(version, commit, date string) {
	if err := newRootCmd(version, commit, date).Execute(); err != nil {
		os.Exit(1)
	}
}
