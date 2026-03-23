package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
)

func TestRegistryTab_LoadData(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)

	// Simulate loaded message
	msg := registryLoadedMsg{
		servers: []model.ServerDef{
			{Name: "github", Transport: model.TransportStdio, Command: "npx"},
			{Name: "postgres", Transport: model.TransportStdio, Command: "npx"},
		},
	}
	updated, _ := reg.Update(msg)
	reg = updated.(*registryTab)

	if len(reg.servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(reg.servers))
	}
	if len(reg.filtered) != 2 {
		t.Fatalf("filtered = %d, want 2", len(reg.filtered))
	}
}

func TestRegistryTab_CursorNavigation(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)

	// Load data
	loaded, _ := reg.Update(registryLoadedMsg{
		servers: []model.ServerDef{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		},
	})
	reg = loaded.(*registryTab)

	if reg.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", reg.cursor)
	}

	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := reg.Update(downMsg)
	reg = updated.(*registryTab)
	if reg.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", reg.cursor)
	}

	// Move up
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ = reg.Update(upMsg)
	reg = updated.(*registryTab)
	if reg.cursor != 0 {
		t.Errorf("cursor after up = %d, want 0", reg.cursor)
	}

	// Can't go above 0
	updated, _ = reg.Update(upMsg)
	reg = updated.(*registryTab)
	if reg.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", reg.cursor)
	}
}

func TestRegistryTab_MultiSelect(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)

	loaded, _ := reg.Update(registryLoadedMsg{
		servers: []model.ServerDef{
			{Name: "a"}, {Name: "b"},
		},
	})
	reg = loaded.(*registryTab)

	// Select first item
	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	updated, _ := reg.Update(spaceMsg)
	reg = updated.(*registryTab)
	if !reg.selected["a"] {
		t.Error("'a' should be selected after space")
	}

	// Toggle off
	updated, _ = reg.Update(spaceMsg)
	reg = updated.(*registryTab)
	if reg.selected["a"] {
		t.Error("'a' should be deselected after second space")
	}
}

func TestRegistryTab_FilterMode(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)

	loaded, _ := reg.Update(registryLoadedMsg{
		servers: []model.ServerDef{
			{Name: "github"}, {Name: "postgres"}, {Name: "gitlab"},
		},
	})
	reg = loaded.(*registryTab)

	// Enter filter mode
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	updated, _ := reg.Update(filterMsg)
	reg = updated.(*registryTab)

	if reg.mode != modeFilter {
		t.Fatalf("mode = %d, want modeFilter", reg.mode)
	}

	// Type "git" — live filter
	for _, r := range "git" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		updated, _ = reg.Update(charMsg)
		reg = updated.(*registryTab)
	}

	if len(reg.filtered) != 2 {
		t.Errorf("filtered after 'git' = %d, want 2 (github, gitlab)", len(reg.filtered))
	}

	// Press Enter to apply
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ = reg.Update(enterMsg)
	reg = updated.(*registryTab)
	if reg.mode != modeList {
		t.Errorf("mode after enter = %d, want modeList", reg.mode)
	}
	if reg.filterText != "git" {
		t.Errorf("filterText = %q, want git", reg.filterText)
	}
}

func TestRegistryTab_AddForm(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg.width = 80
	reg.height = 24

	loaded, _ := reg.Update(registryLoadedMsg{servers: nil})
	reg = loaded.(*registryTab)

	// Press 'a' to add
	addMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	updated, _ := reg.Update(addMsg)
	reg = updated.(*registryTab)

	if reg.mode != modeForm {
		t.Errorf("mode after 'a' = %d, want modeForm", reg.mode)
	}
	if reg.editName != "" {
		t.Errorf("editName = %q, want empty for add", reg.editName)
	}
}

func TestRegistryTab_EditForm(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg.width = 80
	reg.height = 24

	loaded, _ := reg.Update(registryLoadedMsg{
		servers: []model.ServerDef{{Name: "github", Transport: model.TransportStdio}},
	})
	reg = loaded.(*registryTab)

	// Press 'e' to edit
	editMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	updated, _ := reg.Update(editMsg)
	reg = updated.(*registryTab)

	if reg.mode != modeForm {
		t.Errorf("mode after 'e' = %d, want modeForm", reg.mode)
	}
	if reg.editName != "github" {
		t.Errorf("editName = %q, want github", reg.editName)
	}
}

func TestRegistryTab_DeleteConfirm(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)

	loaded, _ := reg.Update(registryLoadedMsg{
		servers: []model.ServerDef{{Name: "github"}},
	})
	reg = loaded.(*registryTab)

	// Press 'd' to delete
	delMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	updated, _ := reg.Update(delMsg)
	reg = updated.(*registryTab)

	if reg.mode != modeConfirm {
		t.Errorf("mode after 'd' = %d, want modeConfirm", reg.mode)
	}
	if reg.deleteName != "github" {
		t.Errorf("deleteName = %q, want github", reg.deleteName)
	}
}

func TestRegistryTab_View_ContainsSubNav(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg.width = 80
	reg.height = 24

	loaded, _ := reg.Update(registryLoadedMsg{servers: nil})
	reg = loaded.(*registryTab)

	view := reg.View()
	if !strings.Contains(view, "MCPs") {
		t.Error("view should contain MCPs sub-nav")
	}
	if !strings.Contains(view, "Skills") {
		t.Error("view should contain Skills sub-nav")
	}
}

func TestRegistryTab_View_ShowsServers(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg.width = 80
	reg.height = 24

	loaded, _ := reg.Update(registryLoadedMsg{
		servers: []model.ServerDef{
			{Name: "github", Transport: model.TransportStdio, Command: "npx"},
		},
	})
	reg = loaded.(*registryTab)

	view := reg.View()
	if !strings.Contains(view, "github") {
		t.Error("view should contain server name 'github'")
	}
	if !strings.Contains(view, "stdio") {
		t.Error("view should contain transport 'stdio'")
	}
}

// --- Parsing helper tests ---

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"  ", 0},
		{"a, b, c", 3},
		{"-y, @anthropic/mcp-github", 2},
		{"single", 1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCSV(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseCSV(%q) = %d items, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestParseKV(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"KEY=val", 1},
		{"A=1, B=2, C=3", 3},
		{"invalid", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseKV(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseKV(%q) = %d items, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestFormatEnv(t *testing.T) {
	got := formatEnv(nil)
	if got != "" {
		t.Errorf("formatEnv(nil) = %q, want empty", got)
	}
	got = formatEnv(map[string]string{"A": "1"})
	if got != "A=1" {
		t.Errorf("formatEnv = %q, want A=1", got)
	}
	// Verify deterministic sort order
	got = formatEnv(map[string]string{"Z": "3", "A": "1", "M": "2"})
	if got != "A=1, M=2, Z=3" {
		t.Errorf("formatEnv multi = %q, want sorted 'A=1, M=2, Z=3'", got)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"ab", 2, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
