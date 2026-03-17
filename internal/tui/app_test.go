package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewApp(t *testing.T) {
	app := NewApp(nil)
	if app.activeTab != ServersTab {
		t.Errorf("expected initial tab to be ServersTab, got %d", app.activeTab)
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

	if app.activeTab != ServersTab {
		t.Fatalf("expected ServersTab initially, got %d", app.activeTab)
	}

	// Press tab to switch to Projects.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	app = m.(AppModel)
	if app.activeTab != ProjectsTab {
		t.Errorf("expected ProjectsTab after tab press, got %d", app.activeTab)
	}

	// Press tab again to wrap back to Servers.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	app = m.(AppModel)
	if app.activeTab != ServersTab {
		t.Errorf("expected ServersTab after second tab press, got %d", app.activeTab)
	}
}

func TestTabSwitchingPrev(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Press shift+tab to go back (wraps to Projects).
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyShiftTab}))
	app = m.(AppModel)
	if app.activeTab != ProjectsTab {
		t.Errorf("expected ProjectsTab after shift+tab, got %d", app.activeTab)
	}
}

func TestTabSwitchingRight(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Press right arrow to go to next tab.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	app = m.(AppModel)
	if app.activeTab != ProjectsTab {
		t.Errorf("expected ProjectsTab after right arrow, got %d", app.activeTab)
	}
}

func TestTabSwitchingLeft(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Press left arrow (wraps to Projects).
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyLeft}))
	app = m.(AppModel)
	if app.activeTab != ProjectsTab {
		t.Errorf("expected ProjectsTab after left arrow, got %d", app.activeTab)
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

func TestViewServersTab(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	view := app.View()
	if !strings.Contains(view, "Servers") {
		t.Errorf("expected view to contain 'Servers' tab label, got:\n%s", view)
	}
	if !strings.Contains(view, "Projects") {
		t.Errorf("expected view to contain 'Projects' tab label, got:\n%s", view)
	}
}

func TestViewProjectsTab(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	// Switch to Projects tab.
	m, _ = app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	app = m.(AppModel)

	view := app.View()
	if !strings.Contains(view, "No project selected") {
		t.Errorf("expected 'No project selected' on projects tab, got:\n%s", view)
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

func TestStatusBarServersTab(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = m.(AppModel)

	view := app.View()
	if !strings.Contains(view, "d: delete") {
		t.Errorf("expected servers status help in status bar, got:\n%s", view)
	}
}

func TestWindowSizePropagation(t *testing.T) {
	app := NewApp(nil)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = m.(AppModel)

	if app.servers.width != 100 {
		t.Errorf("expected servers width 100, got %d", app.servers.width)
	}
	if app.servers.height == 0 {
		t.Errorf("expected servers height > 0, got %d", app.servers.height)
	}
	if app.projects.width != 100 {
		t.Errorf("expected projects width 100, got %d", app.projects.width)
	}
	if app.projects.height == 0 {
		t.Errorf("expected projects height > 0, got %d", app.projects.height)
	}
}
