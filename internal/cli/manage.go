package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/tui"
	"github.com/spf13/cobra"
)

func (a *cliApp) newManageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manage",
		Short: "Open the full management TUI",
		Long:  "Launch the interactive TUI for managing MCPs, profiles, skills, hooks, and permissions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := tui.NewApp(a.svc)
			p := tea.NewProgram(m, tea.WithAltScreen())
			result, err := p.Run()
			if err != nil {
				return err
			}
			if app, ok := result.(tui.AppModel); ok {
				if proj := app.LaunchRequest(); proj != nil {
					return a.syncAndLaunch(cmd, *proj, nil, false, false)
				}
			}
			return nil
		},
	}
}
