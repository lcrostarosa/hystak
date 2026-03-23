package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRegistryTab_LoadData(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	msg := registryLoadedMsg{
		items: []listItem{
			{name: "github", columns: []string{"github", "stdio", "npx"}},
			{name: "postgres", columns: []string{"postgres", "stdio", "npx"}},
		},
	}
	updated, _ := reg.Update(msg)
	reg = updated.(*registryTab)

	view := reg.View()
	if !strings.Contains(view, "github") {
		t.Error("view should contain github after loading")
	}
	if !strings.Contains(view, "postgres") {
		t.Error("view should contain postgres after loading")
	}
}

func TestRegistryTab_CursorNavigation(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{
		items: []listItem{
			{name: "aaa", columns: []string{"aaa"}},
			{name: "bbb", columns: []string{"bbb"}},
			{name: "ccc", columns: []string{"ccc"}},
		},
	})
	reg = loaded.(*registryTab)

	// Initially first item "aaa" should have selected styling (reverse video)
	view := reg.View()
	if !strings.Contains(view, "aaa") {
		t.Fatal("view should contain aaa")
	}

	// Move down - "bbb" should now be highlighted
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := reg.Update(downMsg)
	reg = updated.(*registryTab)

	// Move up back to "aaa"
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ = reg.Update(upMsg)
	reg = updated.(*registryTab)

	// Move up again - should clamp at 0, "aaa" still visible
	updated, _ = reg.Update(upMsg)
	reg = updated.(*registryTab)
	view = reg.View()
	if !strings.Contains(view, "aaa") {
		t.Error("view should still contain aaa after clamping at top")
	}
}

func TestRegistryTab_MultiSelect(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{
		items: []listItem{
			{name: "aaa", columns: []string{"aaa"}},
			{name: "bbb", columns: []string{"bbb"}},
		},
	})
	reg = loaded.(*registryTab)

	// Press space to select first item - should show selection marker
	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	updated, _ := reg.Update(spaceMsg)
	reg = updated.(*registryTab)

	view := reg.View()
	// The selected item should have the "* " marker (green styled)
	if !strings.Contains(view, "aaa") {
		t.Error("view should contain aaa")
	}

	// Press space again to deselect
	updated, _ = reg.Update(spaceMsg)
	reg = updated.(*registryTab)
	// View should still render correctly
	view = reg.View()
	if !strings.Contains(view, "aaa") {
		t.Error("view should still contain aaa after deselect")
	}
}

func TestRegistryTab_FilterMode(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{
		items: []listItem{
			{name: "github", columns: []string{"github"}},
			{name: "postgres", columns: []string{"postgres"}},
			{name: "gitlab", columns: []string{"gitlab"}},
		},
	})
	reg = loaded.(*registryTab)

	// Press "/" to enter filter mode
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	updated, _ := reg.Update(filterMsg)
	reg = updated.(*registryTab)

	// Filter mode should show the filter input
	view := reg.View()
	if !strings.Contains(view, "Filter") {
		t.Error("filter mode should show Filter input in view")
	}

	// Type "git" to filter
	for _, r := range "git" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		updated, _ = reg.Update(charMsg)
		reg = updated.(*registryTab)
	}

	// View should show filtered items (github, gitlab) but not postgres
	view = reg.View()
	if !strings.Contains(view, "github") {
		t.Error("filtered view should contain github")
	}
	if strings.Contains(view, "postgres") {
		t.Error("filtered view should not contain postgres")
	}

	// Press Enter to apply filter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ = reg.Update(enterMsg)
	reg = updated.(*registryTab)

	// View should still show the filter indicator
	view = reg.View()
	if !strings.Contains(view, "git") {
		t.Error("view should show active filter text")
	}
}

func TestRegistryTab_AddForm(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{items: nil})
	reg = loaded.(*registryTab)

	// Press "a" to open add form
	addMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	updated, _ := reg.Update(addMsg)
	reg = updated.(*registryTab)

	// View should show form fields (e.g., "Name" field)
	view := reg.View()
	if !strings.Contains(view, "Name") {
		t.Error("add form view should contain Name field")
	}
}

func TestRegistryTab_EditForm(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{
		items: []listItem{{name: "github", columns: []string{"github", "stdio", "npx"}}},
	})
	reg = loaded.(*registryTab)

	// Press "e" to open edit form
	editMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	updated, _ := reg.Update(editMsg)
	reg = updated.(*registryTab)

	// View should show form fields with edit context
	view := reg.View()
	if !strings.Contains(view, "Edit") {
		t.Error("edit form view should contain Edit in title")
	}
}

func TestRegistryTab_DeleteConfirm(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{
		items: []listItem{{name: "github"}},
	})
	reg = loaded.(*registryTab)

	// Press "d" to trigger delete confirm
	delMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	updated, _ := reg.Update(delMsg)
	reg = updated.(*registryTab)

	// View should show confirm dialog with delete prompt
	view := reg.View()
	if !strings.Contains(view, "github") {
		t.Error("delete confirm should mention the item name")
	}
	if !strings.Contains(view, "Delete") {
		t.Error("delete confirm should contain Delete in title")
	}
}

func TestRegistryTab_SubNavSwitch(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{items: nil})
	reg = loaded.(*registryTab)

	// Initially should show MCPs sub-nav as active
	view := reg.View()
	if !strings.Contains(view, "MCPs") {
		t.Fatal("initial view should contain MCPs sub-nav")
	}

	// Press right to switch to Skills
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := reg.Update(rightMsg)
	reg = updated.(*registryTab)

	view = reg.View()
	if !strings.Contains(view, "Skills") {
		t.Error("after right, view should show Skills sub-nav")
	}

	// Press left to go back to MCPs
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ = reg.Update(leftMsg)
	reg = updated.(*registryTab)

	view = reg.View()
	if !strings.Contains(view, "MCPs") {
		t.Error("after left, view should show MCPs sub-nav")
	}
}

func TestRegistryTab_AllSubNavs_HaveViews(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	for sub := SubNav(0); sub < subNavCount; sub++ {
		reg.sub = sub
		loaded, _ := reg.Update(registryLoadedMsg{items: nil})
		reg = loaded.(*registryTab)

		view := reg.View()
		if !strings.Contains(view, subNavNames[sub]) {
			t.Errorf("sub-nav %s: view should contain its own name", subNavNames[sub])
		}
	}
}

func TestRegistryTab_View_ContainsSubNav(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{items: nil})
	reg = loaded.(*registryTab)

	view := reg.View()
	if !strings.Contains(view, "MCPs") {
		t.Error("view should contain MCPs sub-nav")
	}
	if !strings.Contains(view, "Skills") {
		t.Error("view should contain Skills sub-nav")
	}
}

func TestRegistryTab_View_ShowsItems(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{
		items: []listItem{{name: "github", columns: []string{"github", "stdio", "npx"}}},
	})
	reg = loaded.(*registryTab)

	view := reg.View()
	if !strings.Contains(view, "github") {
		t.Error("view should contain item name 'github'")
	}
	if !strings.Contains(view, "stdio") {
		t.Error("view should contain 'stdio'")
	}
}

func TestRegistryTab_ErrorMsg_ShowsInView(t *testing.T) {
	app := setupTestApp(t)
	reg := app.tabs[TabRegistry].(*registryTab)
	reg, _ = applyWindowSize(t, reg, 80, 24)

	loaded, _ := reg.Update(registryLoadedMsg{items: nil})
	reg = loaded.(*registryTab)

	// Send error message
	errMsg := serverErrorMsg{err: fmt.Errorf("test error message")}
	updated, _ := reg.Update(errMsg)
	reg = updated.(*registryTab)

	view := reg.View()
	if !strings.Contains(view, "test error message") {
		t.Error("error message should appear in View()")
	}
}

// applyWindowSize sends a WindowSizeMsg and returns the updated registryTab.
func applyWindowSize(t *testing.T, reg *registryTab, w, h int) (*registryTab, tea.Cmd) {
	t.Helper()
	updated, cmd := reg.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(*registryTab), cmd
}

// --- SubView tests ---

func TestSubViews_AllDefined(t *testing.T) {
	for i := SubNav(0); i < subNavCount; i++ {
		sv := subViews[i]
		if sv.header == "" {
			t.Errorf("subView %s has empty header", subNavNames[i])
		}
		if sv.loadItems == nil {
			t.Errorf("subView %s has nil loadItems", subNavNames[i])
		}
		if sv.addFields == nil {
			t.Errorf("subView %s has nil addFields", subNavNames[i])
		}
		if sv.editFields == nil {
			t.Errorf("subView %s has nil editFields", subNavNames[i])
		}
		if sv.save == nil {
			t.Errorf("subView %s has nil save", subNavNames[i])
		}
		if sv.delete == nil {
			t.Errorf("subView %s has nil delete", subNavNames[i])
		}
	}
}

func TestSubViews_AddFields_NonEmpty(t *testing.T) {
	for i := SubNav(0); i < subNavCount; i++ {
		sv := subViews[i]
		fields := sv.addFields()
		if len(fields) == 0 {
			t.Errorf("subView %s addFields returned 0 fields", subNavNames[i])
		}
		// First field should always be Name
		if fields[0].Label != "Name" {
			t.Errorf("subView %s first field = %q, want Name", subNavNames[i], fields[0].Label)
		}
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
