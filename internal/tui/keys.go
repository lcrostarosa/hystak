package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines global key bindings for the app.
type KeyMap struct {
	Quit    key.Binding
	TabNext key.Binding
	TabPrev key.Binding
}

func newKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		TabNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		TabPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
	}
}

// ShortHelp returns bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.TabNext, k.Quit}
}

// FullHelp returns bindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.TabNext, k.TabPrev},
		{k.Quit},
	}
}
