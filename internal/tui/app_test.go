package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/project"
	"github.com/hystak/hystak/internal/registry"
	"github.com/hystak/hystak/internal/service"
)

func setupTestApp(t *testing.T) App {
	t.Helper()
	tmp := t.TempDir()

	reg := registry.New()
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@anthropic/mcp-github"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "postgres",
		Transport: model.TransportStdio,
		Command:   "npx",
	}); err != nil {
		t.Fatal(err)
	}

	projStore := project.NewStore()
	if err := projStore.Add(model.Project{
		Name:          "myproject",
		Path:          filepath.Join(tmp, "myproject"),
		ActiveProfile: "dev",
	}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	if err := mkdirAll(profDir); err != nil {
		t.Fatal(err)
	}
	profMgr := profile.NewManager(profDir)
	if err := profMgr.Save(model.ProjectProfile{
		Name:      "dev",
		MCPs:      []model.MCPAssignment{{Name: "github"}},
		Isolation: model.IsolationNone,
	}); err != nil {
		t.Fatal(err)
	}

	dep := &deploy.ClaudeCodeDeployer{}
	svc := service.New(reg, projStore, profMgr, dep)
	keys := DefaultKeyMap()

	return NewApp(svc, keys, "test", "abc123", "2026-03-22")
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func TestApp_Init_Returns4TabCmds(t *testing.T) {
	app := setupTestApp(t)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, expected batch cmd")
	}
}

func TestApp_HasCorrectTabCount(t *testing.T) {
	app := setupTestApp(t)
	if len(app.tabs) != int(tabCount) {
		t.Errorf("tab count = %d, want %d", len(app.tabs), tabCount)
	}
}

func TestApp_TabTitles(t *testing.T) {
	app := setupTestApp(t)
	want := []string{"Registry", "Projects", "Tools", "Help"}
	for i, tab := range app.tabs {
		if tab.Title() != want[i] {
			t.Errorf("tab %d title = %q, want %q", i, tab.Title(), want[i])
		}
	}
}

func TestApp_TabNavigation_NextTab(t *testing.T) {
	app := setupTestApp(t)
	app.width = 80
	app.height = 24

	if app.activeTab != TabRegistry {
		t.Fatalf("initial tab = %d, want Registry", app.activeTab)
	}

	// Press Tab to go to next
	msg := tea.KeyMsg{Type: tea.KeyTab}
	result, _ := app.Update(msg)
	app = result.(App)

	if app.activeTab != TabProjects {
		t.Errorf("after Tab, activeTab = %d, want Projects", app.activeTab)
	}
}

func TestApp_TabNavigation_PrevTab(t *testing.T) {
	app := setupTestApp(t)
	app.width = 80
	app.height = 24

	// Press Shift+Tab to go to prev (wraps to Help)
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	result, _ := app.Update(msg)
	app = result.(App)

	if app.activeTab != TabHelp {
		t.Errorf("after Shift+Tab from Registry, activeTab = %d, want Help", app.activeTab)
	}
}

func TestApp_TabNavigation_NumberKeys(t *testing.T) {
	app := setupTestApp(t)
	app.width = 80
	app.height = 24

	tests := []struct {
		key  string
		want TabIndex
	}{
		{"2", TabProjects},
		{"3", TabTools},
		{"4", TabHelp},
		{"1", TabRegistry},
	}

	for _, tt := range tests {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
		result, _ := app.Update(msg)
		app = result.(App)
		if app.activeTab != tt.want {
			t.Errorf("after key %q, activeTab = %d, want %d", tt.key, app.activeTab, tt.want)
		}
	}
}

func TestApp_WindowResize_Propagates(t *testing.T) {
	app := setupTestApp(t)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, _ := app.Update(msg)
	app = result.(App)

	if app.width != 120 || app.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", app.width, app.height)
	}
}

func TestApp_View_ContainsTabBar(t *testing.T) {
	app := setupTestApp(t)
	app.width = 80
	app.height = 24

	view := app.View()
	if !strings.Contains(view, "Registry") {
		t.Error("view missing 'Registry' in tab bar")
	}
	if !strings.Contains(view, "Projects") {
		t.Error("view missing 'Projects' in tab bar")
	}
	if !strings.Contains(view, "Tools") {
		t.Error("view missing 'Tools' in tab bar")
	}
	if !strings.Contains(view, "Help") {
		t.Error("view missing 'Help' in tab bar")
	}
}

func TestApp_View_Loading(t *testing.T) {
	app := setupTestApp(t)
	// width/height = 0 triggers loading state
	view := app.View()
	if view != "Loading..." {
		t.Errorf("view with no size = %q, want 'Loading...'", view)
	}
}

func TestApp_CtrlC_Quits(t *testing.T) {
	app := setupTestApp(t)
	app.width = 80
	app.height = 24

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := app.Update(msg)
	if cmd == nil {
		t.Fatal("ctrl+c should return a quit command")
	}
}

func TestApp_ConfirmOverlay_Dismiss(t *testing.T) {
	app := setupTestApp(t)
	app.width = 80
	app.height = 24
	app.overlay = OverlayConfirm
	app.overlayTitle = "Test"
	app.overlayMsg = "Are you sure?"

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := app.Update(msg)
	app = result.(App)

	if app.overlay != OverlayNone {
		t.Errorf("overlay = %d after Esc, want OverlayNone", app.overlay)
	}
}
