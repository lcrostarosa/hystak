package tui

import "github.com/charmbracelet/lipgloss"

// Color scheme — monochrome-compatible with color enhancement.
var (
	colorPrimary   = lipgloss.Color("12")  // blue
	colorGreen     = lipgloss.Color("10")  // green
	colorYellow    = lipgloss.Color("11")  // yellow
	colorRed       = lipgloss.Color("9")   // red
	colorDim       = lipgloss.Color("245") // gray
	colorCyan      = lipgloss.Color("14")  // cyan
	colorHighlight = lipgloss.Color("15")  // bright white
)

// Tab bar styles.
var (
	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			Foreground(colorHighlight)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorDim)

	styleTabBar = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingBottom(1)
)

// List styles.
var (
	styleListHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorDim)

	styleListSelected = lipgloss.NewStyle().
				Reverse(true)

	styleListNormal = lipgloss.NewStyle()
)

// Status indicator styles.
var (
	styleSynced    = lipgloss.NewStyle().Foreground(colorGreen)
	styleDrifted   = lipgloss.NewStyle().Foreground(colorYellow)
	styleMissing   = lipgloss.NewStyle().Foreground(colorPrimary)
	styleUnmanaged = lipgloss.NewStyle().Foreground(colorDim)
	styleError     = lipgloss.NewStyle().Foreground(colorRed)
)

// Overlay styles.
var (
	styleOverlayBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan).
		Padding(1, 2)
)

// Footer/help bar styles.
var (
	styleHelpKey  = lipgloss.NewStyle().Bold(true)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colorDim)
	styleFooter   = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingTop(1)
)

// Title style.
var styleTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorHighlight)
