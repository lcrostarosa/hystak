package tui

import "github.com/charmbracelet/lipgloss"

var logoStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205")).
	Bold(true)

var logoSubtitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("245")).
	Italic(true)

// RenderLogo returns the ASCII art boot logo for hystak.
func RenderLogo(width int) string {
	art := `
 _               _        _
| |__  _   _ ___| |_ __ _| | __
| '_ \| | | / __| __/ _` + "`" + ` | |/ /
| | | | |_| \__ \ || (_| |   <
|_| |_|\__, |___/\__\__,_|_|\_\
       |___/                    `

	subtitle := "Claude Code session launcher"

	logo := logoStyle.Render(art) + "\n" + logoSubtitleStyle.Render(subtitle)

	if width > 0 {
		return lipgloss.PlaceHorizontal(width, lipgloss.Center, logo)
	}
	return logo
}

// logoHeight returns the number of lines consumed by the logo.
func logoHeight() int {
	return lipgloss.Height(RenderLogo(0))
}
