package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/tui"
	"github.com/spf13/cobra"
)

func (a *cliApp) newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Run the setup wizard to import configs and create a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			wizard := tui.NewWizardModel(a.svc)
			p := tea.NewProgram(wizard, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}
