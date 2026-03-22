package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/keyconfig"
)

func TestNewKeyMap_FromArrowsProfile(t *testing.T) {
	cfg := keyconfig.DefaultConfig()
	km := NewKeyMap(cfg.ResolvedBindings())

	if len(km.NextTab.Keys()) == 0 {
		t.Error("NextTab has no bindings")
	}
	if km.NextTab.Keys()[0] != "tab" {
		t.Errorf("NextTab.Keys()[0] = %q, want tab", km.NextTab.Keys()[0])
	}
	if len(km.ListUp.Keys()) == 0 || km.ListUp.Keys()[0] != "up" {
		t.Errorf("ListUp = %v, want [up]", km.ListUp.Keys())
	}
}

func TestNewKeyMap_FromVimProfile(t *testing.T) {
	cfg := keyconfig.Config{Profile: keyconfig.ProfileVim}
	km := NewKeyMap(cfg.ResolvedBindings())

	if len(km.ListUp.Keys()) == 0 || km.ListUp.Keys()[0] != "k" {
		t.Errorf("vim ListUp = %v, want [k]", km.ListUp.Keys())
	}
	if len(km.ListDown.Keys()) == 0 || km.ListDown.Keys()[0] != "j" {
		t.Errorf("vim ListDown = %v, want [j]", km.ListDown.Keys())
	}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	if len(km.Cancel.Keys()) == 0 {
		t.Error("DefaultKeyMap Cancel has no bindings")
	}
}

func TestKeyMatches_Tab(t *testing.T) {
	km := DefaultKeyMap()
	msg := tea.KeyMsg{Type: tea.KeyTab}
	if !key.Matches(msg, km.NextTab) {
		t.Error("expected Tab to match NextTab binding")
	}
}

func TestKeyMatches_Rune(t *testing.T) {
	km := DefaultKeyMap()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	if !key.Matches(msg, km.Add) {
		t.Error("expected 'a' to match Add binding")
	}
}

func TestKeyMatches_NoMatch(t *testing.T) {
	km := DefaultKeyMap()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")}
	if key.Matches(msg, km.Add) {
		t.Error("expected 'z' not to match Add binding")
	}
}

func TestNewBinding_LowercasesKeys(t *testing.T) {
	b := newBinding([]string{"Tab", "Shift+Tab"}, "tab", "test")
	keys := b.Keys()
	if keys[0] != "tab" {
		t.Errorf("expected lowercase 'tab', got %q", keys[0])
	}
	if keys[1] != "shift+tab" {
		t.Errorf("expected lowercase 'shift+tab', got %q", keys[1])
	}
}
