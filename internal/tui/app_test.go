package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewApp(t *testing.T) {
	app := NewApp(nil)
	if app.activeTab != ProfilesTab {
		t.Errorf("expected initial tab to be ProfilesTab, got %d", app.activeTab)
	}
	if app.mode != ModeBrowse {
		t.Errorf("expected initial mode to be ModeBrowse, got %d", app.mode)
	}
}

func TestTabSwitchingNext(t *testing.T) {
	app := NewApp(nil)
	// Simulate window size so View works.
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	if app.activeTab != ProfilesTab {
		t.Fatalf("expected ProfilesTab initially, got %d", app.activeTab)
	}

	// Press tab to switch to MCPs.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	app = m.(AppModel)
	if app.activeTab != MCPsTab {
		t.Errorf("expected MCPsTab after tab press, got %d", app.activeTab)
	}

	// Press tab again to switch to Skills.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	app = m.(AppModel)
	if app.activeTab != SkillsTab {
		t.Errorf("expected SkillsTab after second tab press, got %d", app.activeTab)
	}

	// Advance through all remaining tabs and verify wrap-around.
	for i := 0; i < int(tabCount)-2; i++ {
		m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
		app = m.(AppModel)
	}
	if app.activeTab != ProfilesTab {
		t.Errorf("expected ProfilesTab after cycling all tabs, got %d", app.activeTab)
	}
}

func TestTabSwitchingPrev(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Press shift+tab to go back (wraps to last tab: PromptsTab).
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyShiftTab}))
	app = m.(AppModel)
	if app.activeTab != PromptsTab {
		t.Errorf("expected PromptsTab after shift+tab, got %d", app.activeTab)
	}
}

func TestQuitKey(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Press q to quit.
	_, cmd := app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}}))
	if cmd == nil {
		t.Fatal("expected a command from quit key, got nil")
	}
	// Execute the cmd and check it produces a QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestCtrlCQuit(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	_, cmd := app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))
	if cmd == nil {
		t.Fatal("expected a command from ctrl+c, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestViewProfilesTab(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	view := app.View()
	if !strings.Contains(view, "Profiles") {
		t.Errorf("expected view to contain 'Profiles' tab label, got:\n%s", view)
	}
	if !strings.Contains(view, "MCPs") {
		t.Errorf("expected view to contain 'MCPs' tab label, got:\n%s", view)
	}
}

func TestViewMCPsTab(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Switch to MCPs tab.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	app = m.(AppModel)

	view := app.View()
	if !strings.Contains(view, "No MCP selected") {
		t.Errorf("expected 'No MCP selected' on MCPs tab, got:\n%s", view)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(AppModel)

	if app.width != 120 {
		t.Errorf("expected width 120, got %d", app.width)
	}
	if app.height != 40 {
		t.Errorf("expected height 40, got %d", app.height)
	}
}

func TestInitReturnsNil(t *testing.T) {
	app := NewApp(nil)
	cmd := app.Init()
	if cmd != nil {
		t.Errorf("expected Init() to return nil, got %v", cmd)
	}
}

func TestViewBeforeWindowSize(t *testing.T) {
	app := NewApp(nil)
	view := app.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected 'Loading...' before window size, got:\n%s", view)
	}
}

func TestStatusBarProfilesTab(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	view := app.View()
	if !strings.Contains(view, "d: delete") {
		t.Errorf("expected profiles status help in status bar, got:\n%s", view)
	}
}

func TestWindowSizePropagation(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = m.(AppModel)

	if app.mcps.width != 100 {
		t.Errorf("expected mcps width 100, got %d", app.mcps.width)
	}
	if app.mcps.height == 0 {
		t.Errorf("expected mcps height > 0, got %d", app.mcps.height)
	}
	if app.profiles.width != 100 {
		t.Errorf("expected profiles width 100, got %d", app.profiles.width)
	}
	if app.profiles.height == 0 {
		t.Errorf("expected profiles height > 0, got %d", app.profiles.height)
	}
}
