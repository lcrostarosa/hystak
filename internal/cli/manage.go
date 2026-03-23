package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/keyconfig"
	"github.com/hystak/hystak/internal/tui"
	"github.com/spf13/cobra"
)

var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Open the management TUI",
	Long:  "Launch the full management TUI for all resources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return launchTUI()
	},
}

func init() {
	rootCmd.AddCommand(manageCmd)
}

// launchTUI builds a service and starts the Bubble Tea TUI.
func launchTUI() error {
	svc, err := buildServiceReadOnly()
	if err != nil {
		return err
	}

	keyCfg, err := keyconfig.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: loading keybindings: %v\n", err)
		keyCfg = keyconfig.DefaultConfig()
	}

	keys := tui.NewKeyMap(keyCfg.ResolvedBindings())
	app := tui.NewApp(svc, keys, version, commit, date)

	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
