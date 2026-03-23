package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// OverlayKind identifies the type of modal overlay.
type OverlayKind int

const (
	OverlayNone OverlayKind = iota
	OverlayConfirm
	OverlayForm
	OverlayDiff
)

// confirmOverlay renders a centered confirmation dialog.
func confirmOverlay(title, message string, width, height int) string {
	content := title + "\n\n" + message + "\n\n" +
		styleHelpKey.Render("Y") + styleHelpDesc.Render(":Confirm  ") +
		styleHelpKey.Render("N/Esc") + styleHelpDesc.Render(":Cancel")

	box := styleOverlayBorder.
		Width(min(width-4, 60)).
		Render(content)

	return centerOverlay(box, width, height)
}

// centerOverlay places a rendered box in the center of the terminal.
func centerOverlay(box string, width, height int) string {
	boxLines := strings.Split(box, "\n")
	boxHeight := len(boxLines)
	boxWidth := lipgloss.Width(box)

	// Vertical centering
	topPad := (height - boxHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Horizontal centering
	leftPad := (width - boxWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	var b strings.Builder
	for range topPad {
		b.WriteString("\n")
	}
	padStr := strings.Repeat(" ", leftPad)
	for _, line := range boxLines {
		b.WriteString(padStr)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}
