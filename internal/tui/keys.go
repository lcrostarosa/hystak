package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/lcrostarosa/hystak/internal/keyconfig"
)

// KeyMap defines all key bindings for the app, populated from keyconfig.
type KeyMap struct {
	// Global
	Quit    key.Binding
	TabNext key.Binding
	TabPrev key.Binding

	// Profiles tab
	ProfileLaunch    key.Binding
	ProfileConfigure key.Binding
	ProfileDelete    key.Binding

	// Tools tab
	ToolsExecute key.Binding

	// MCPs tab
	MCPAdd    key.Binding
	MCPEdit   key.Binding
	MCPDelete key.Binding
	MCPImport key.Binding

	// Resource tabs (skills, hooks, permissions, templates, prompts)
	ResourceAdd    key.Binding
	ResourceEdit   key.Binding
	ResourceDelete key.Binding
}

func newKeyMap(r keyconfig.ResolvedKeys) KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys(r.Global.Quit...),
			key.WithHelp(r.Global.Quit[0], "quit"),
		),
		TabNext: key.NewBinding(
			key.WithKeys(r.Global.TabNext...),
			key.WithHelp(r.Global.TabNext[0], "next tab"),
		),
		TabPrev: key.NewBinding(
			key.WithKeys(r.Global.TabPrev...),
			key.WithHelp(r.Global.TabPrev[0], "prev tab"),
		),

		ProfileLaunch: key.NewBinding(
			key.WithKeys(r.Profiles.Launch...),
			key.WithHelp(r.Profiles.Launch[0], "launch"),
		),
		ProfileConfigure: key.NewBinding(
			key.WithKeys(r.Profiles.Configure...),
			key.WithHelp(r.Profiles.Configure[0], "configure"),
		),
		ProfileDelete: key.NewBinding(
			key.WithKeys(r.Profiles.Delete...),
			key.WithHelp(r.Profiles.Delete[0], "delete"),
		),

		ToolsExecute: key.NewBinding(
			key.WithKeys(r.Tools.Execute...),
			key.WithHelp(r.Tools.Execute[0], "run"),
		),

		MCPAdd: key.NewBinding(
			key.WithKeys(r.MCPs.Add...),
			key.WithHelp(r.MCPs.Add[0], "add"),
		),
		MCPEdit: key.NewBinding(
			key.WithKeys(r.MCPs.Edit...),
			key.WithHelp(r.MCPs.Edit[0], "edit"),
		),
		MCPDelete: key.NewBinding(
			key.WithKeys(r.MCPs.Delete...),
			key.WithHelp(r.MCPs.Delete[0], "delete"),
		),
		MCPImport: key.NewBinding(
			key.WithKeys(r.MCPs.Import...),
			key.WithHelp(r.MCPs.Import[0], "import"),
		),

		ResourceAdd: key.NewBinding(
			key.WithKeys(r.Skills.Add...),
			key.WithHelp(r.Skills.Add[0], "add"),
		),
		ResourceEdit: key.NewBinding(
			key.WithKeys(r.Skills.Edit...),
			key.WithHelp(r.Skills.Edit[0], "edit"),
		),
		ResourceDelete: key.NewBinding(
			key.WithKeys(r.Skills.Delete...),
			key.WithHelp(r.Skills.Delete[0], "delete"),
		),
	}
}

// newDefaultKeyMap creates a KeyMap from the default (arrows) preset.
func newDefaultKeyMap() KeyMap {
	r, _ := keyconfig.Preset(keyconfig.ProfileArrows)
	return newKeyMap(r)
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

// tabNavHelp returns a formatted string showing the tab navigation keys.
func (k KeyMap) tabNavHelp() string {
	return k.TabNext.Help().Key + "/" + k.TabPrev.Help().Key + ": switch tabs"
}
