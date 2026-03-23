package tui

import tea "github.com/charmbracelet/bubbletea"

// confirmModel is a modal confirmation dialog.
type confirmModel struct {
	title   string
	message string
	width   int
	height  int
}

func newConfirmModel(title, message string) confirmModel {
	return confirmModel{title: title, message: message}
}

// confirmYesMsg signals confirmation was accepted.
type confirmYesMsg struct{}

// confirmNoMsg signals confirmation was rejected.
type confirmNoMsg struct{}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (confirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			return m, func() tea.Msg { return confirmYesMsg{} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return confirmNoMsg{} }
		}
	}
	return m, nil
}

func (m confirmModel) View(width, height int) string {
	return confirmOverlay(m.title, m.message, width, height)
}
