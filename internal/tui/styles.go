package tui

import "github.com/charmbracelet/lipgloss"

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 2)

	tabGapStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("238"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	detailPaneStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("238"))

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("245"))

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("202")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	syncMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))
)
