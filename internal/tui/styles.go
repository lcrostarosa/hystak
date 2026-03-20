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

	formTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	formLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245"))

	formBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	formSelectedTransportStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("205")).
					Background(lipgloss.Color("236")).
					Padding(0, 1)

	formTransportStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)

	formHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	diffDelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("45"))

	diffHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("245"))

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginTop(1)

	sectionActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	sectionDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)
