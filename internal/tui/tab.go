package tui

import tea "github.com/charmbracelet/bubbletea"

// Tab is the interface for each top-level TUI tab.
// Each tab is a full tea.Model that handles its own keys and rendering.
type Tab interface {
	tea.Model
	Title() string
	// HelpKeys returns the context-specific help entries for the footer bar.
	HelpKeys() []HelpEntry
}

// HelpEntry is a single key-description pair for the help bar.
type HelpEntry struct {
	Key  string
	Desc string
}

// TabIndex identifies which top-level tab is active.
type TabIndex int

const (
	TabRegistry TabIndex = iota
	TabProjects
	TabTools
	TabHelp
	tabCount // sentinel for modular arithmetic
)
