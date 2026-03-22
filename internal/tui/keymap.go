package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/hystak/hystak/internal/keyconfig"
)

// KeyMap holds resolved key bindings for the TUI using bubbles key.Binding.
// This integrates with bubbles' key.Matches for routing and help display.
type KeyMap struct {
	NextTab      key.Binding
	PrevTab      key.Binding
	ListUp       key.Binding
	ListDown     key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Top          key.Binding
	Bottom       key.Binding
	Select       key.Binding
	Confirm      key.Binding
	Cancel       key.Binding
	Add          key.Binding
	Edit         key.Binding
	Delete       key.Binding
	Filter       key.Binding
	Launch       key.Binding
	Import       key.Binding
	Preview      key.Binding
	SyncFromDiff key.Binding
}

// newBinding creates a key.Binding from a keyconfig action's key strings and help text.
// Normalizes key names to lowercase for bubbletea compatibility.
func newBinding(keys []string, helpKey, helpDesc string) key.Binding {
	lower := make([]string, len(keys))
	for i, k := range keys {
		lower[i] = strings.ToLower(k)
	}
	return key.NewBinding(
		key.WithKeys(lower...),
		key.WithHelp(helpKey, helpDesc),
	)
}

// NewKeyMap creates a KeyMap from resolved keybinding config.
func NewKeyMap(bindings keyconfig.Bindings) KeyMap {
	return KeyMap{
		NextTab:      newBinding(bindings["next_tab"], "tab", "next tab"),
		PrevTab:      newBinding(bindings["prev_tab"], "shift+tab", "prev tab"),
		ListUp:       newBinding(bindings["list_up"], "up", "up"),
		ListDown:     newBinding(bindings["list_down"], "down", "down"),
		PageUp:       newBinding(bindings["page_up"], "pgup", "page up"),
		PageDown:     newBinding(bindings["page_down"], "pgdn", "page down"),
		Top:          newBinding(bindings["top"], "home", "top"),
		Bottom:       newBinding(bindings["bottom"], "end", "bottom"),
		Select:       newBinding(bindings["select"], "space", "toggle"),
		Confirm:      newBinding(bindings["confirm"], "enter", "confirm"),
		Cancel:       newBinding(bindings["cancel"], "esc", "cancel"),
		Add:          newBinding(bindings["add"], "a", "add"),
		Edit:         newBinding(bindings["edit"], "e", "edit"),
		Delete:       newBinding(bindings["delete"], "d", "delete"),
		Filter:       newBinding(bindings["filter"], "/", "filter"),
		Launch:       newBinding(bindings["launch"], "l", "launch"),
		Import:       newBinding(bindings["import"], "i", "import"),
		Preview:      newBinding(bindings["preview"], "p", "preview"),
		SyncFromDiff: newBinding(bindings["sync_from_diff"], "s", "sync"),
	}
}

// DefaultKeyMap returns a KeyMap based on the arrows profile defaults.
func DefaultKeyMap() KeyMap {
	cfg := keyconfig.DefaultConfig()
	return NewKeyMap(cfg.ResolvedBindings())
}
