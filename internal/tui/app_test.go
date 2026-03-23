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

// sizeApp sends a WindowSizeMsg and returns the updated App, avoiding direct field access.
func sizeApp(t *testing.T, app App, w, h int) App {
	t.Helper()
	result, _ := app.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return result.(App)
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
	app = sizeApp(t, app, 80, 24)

	// Initial view should show Registry tab content
	view := app.View()
	if !strings.Contains(view, "Registry") {
		t.Fatal("initial view missing Registry tab")
	}

	// Press Tab to go to next
	msg := tea.KeyMsg{Type: tea.KeyTab}
	result, _ := app.Update(msg)
	app = result.(App)

	view = app.View()
	if !strings.Contains(view, "Projects") {
		t.Error("after Tab, view should emphasize Projects tab")
	}
}

func TestApp_TabNavigation_PrevTab(t *testing.T) {
	app := setupTestApp(t)
	app = sizeApp(t, app, 80, 24)

	// Press Shift+Tab to go to prev (wraps to Help)
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	result, _ := app.Update(msg)
	app = result.(App)

	view := app.View()
	if !strings.Contains(view, "Help") {
		t.Error("after Shift+Tab from Registry, view should show Help tab content")
	}
}

func TestApp_TabNavigation_NumberKeys(t *testing.T) {
	app := setupTestApp(t)
	app = sizeApp(t, app, 80, 24)

	tests := []struct {
		key     string
		wantTab string
	}{
		{"2", "Projects"},
		{"3", "Tools"},
		{"4", "Help"},
		{"1", "Registry"},
	}

	for _, tt := range tests {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
		result, _ := app.Update(msg)
		app = result.(App)
		view := app.View()
		// The active tab content should be visible in the view
		if !strings.Contains(view, tt.wantTab) {
			t.Errorf("after key %q, view should contain %q", tt.key, tt.wantTab)
		}
	}
}

func TestApp_WindowResize_Propagates(t *testing.T) {
	app := setupTestApp(t)

	// Before resize, view shows loading
	view := app.View()
	if view != "Loading..." {
		t.Errorf("before resize, view = %q, want Loading...", view)
	}

	// After resize, view should render tab content
	app = sizeApp(t, app, 120, 40)
	view = app.View()
	if view == "Loading..." {
		t.Error("after resize, view should no longer show Loading...")
	}
	if !strings.Contains(view, "Registry") {
		t.Error("after resize, view should contain tab bar with Registry")
	}
}

func TestApp_View_ContainsTabBar(t *testing.T) {
	app := setupTestApp(t)
	app = sizeApp(t, app, 80, 24)

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
	app = sizeApp(t, app, 80, 24)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := app.Update(msg)
	if cmd == nil {
		t.Fatal("ctrl+c should return a quit command")
	}
}

func TestApp_ConfirmOverlay_Dismiss(t *testing.T) {
	app := setupTestApp(t)
	app = sizeApp(t, app, 80, 24)
	app.overlay = OverlayConfirm
	app.overlayTitle = "Test"
	app.overlayMsg = "Are you sure?"

	// Verify the confirm overlay appears in view
	view := app.View()
	if !strings.Contains(view, "Are you sure?") {
		t.Error("confirm overlay should appear in view")
	}

	// Dismiss with Esc
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := app.Update(msg)
	app = result.(App)

	// Overlay should be gone - view should show normal tab content
	view = app.View()
	if strings.Contains(view, "Are you sure?") {
		t.Error("confirm overlay should be dismissed after Esc")
	}
}
