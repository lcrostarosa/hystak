package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
)

func TestProjectsTab_LoadData(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)

	msg := projectsLoadedMsg{
		projects: []model.Project{
			{Name: "myproject", Path: "/test/myproject", ActiveProfile: "dev"},
		},
	}
	updated, _ := pt.Update(msg)
	pt = updated.(*projectsTab)

	if len(pt.projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(pt.projects))
	}
}

func TestProjectsTab_CursorNavigation(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "a", Path: "/a", ActiveProfile: "dev"},
			{Name: "b", Path: "/b", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)

	// Need to also load detail to avoid nil issues
	detailMsg := projectDetailMsg{
		profileNames: []string{"dev", "empty"},
		allMCPs:      []string{"github"},
	}
	updated, _ := pt.Update(detailMsg)
	pt = updated.(*projectsTab)

	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ = pt.Update(downMsg)
	pt = updated.(*projectsTab)
	if pt.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", pt.cursor)
	}
}

func TestProjectsTab_PaneSwitch(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "proj", Path: "/test", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)
	pt.profileNames = []string{"dev"}

	if pt.pane != paneProjects {
		t.Fatalf("initial pane = %d, want paneProjects", pt.pane)
	}

	// Enter moves to detail
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := pt.Update(enterMsg)
	pt = updated.(*projectsTab)
	if pt.pane != paneDetail {
		t.Errorf("pane after Enter = %d, want paneDetail", pt.pane)
	}

	// Esc returns to projects
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = pt.Update(escMsg)
	pt = updated.(*projectsTab)
	if pt.pane != paneProjects {
		t.Errorf("pane after Esc = %d, want paneProjects", pt.pane)
	}
}

func TestProjectsTab_ProfilePicker(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "proj", Path: "/test", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)
	pt.profileNames = []string{"dev", "review", "empty"}

	// Press P to open picker
	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
	updated, _ := pt.Update(pMsg)
	pt = updated.(*projectsTab)
	if pt.mode != projectsModeProfilePicker {
		t.Errorf("mode after P = %d, want profilePicker", pt.mode)
	}

	// Esc closes picker
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = pt.Update(escMsg)
	pt = updated.(*projectsTab)
	if pt.mode != projectsModeNormal {
		t.Errorf("mode after Esc = %d, want normal", pt.mode)
	}
}

func TestProjectsTab_View_TwoPane(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt.width = 100
	pt.height = 30

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "myproject", Path: "/test/myproject", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)

	detailMsg := projectDetailMsg{
		profileNames: []string{"dev", "empty"},
		allMCPs:      []string{"github", "postgres"},
		allSkills:    []string{"review"},
	}
	updated, _ := pt.Update(detailMsg)
	pt = updated.(*projectsTab)

	view := pt.View()
	if !strings.Contains(view, "myproject") {
		t.Error("view should contain project name")
	}
	if !strings.Contains(view, "PROJECTS") {
		t.Error("view should contain PROJECTS header")
	}
	if !strings.Contains(view, "MCPs") {
		t.Error("view should contain MCPs section")
	}
	if !strings.Contains(view, "github") {
		t.Error("view should contain MCP name 'github'")
	}
}

func TestProjectsTab_View_Empty(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)

	view := pt.View()
	if !strings.Contains(view, "No projects") {
		t.Error("empty view should say 'No projects'")
	}
}

func TestProjectsTab_DetailSections(t *testing.T) {
	// Verify all detail sections are defined
	for i := detailSection(0); i < detailSectionCount; i++ {
		if detailSectionNames[i] == "" {
			t.Errorf("detailSectionNames[%d] is empty", i)
		}
	}
}

func TestMakeSet(t *testing.T) {
	s := makeSet([]string{"a", "b", "c"})
	if !s["a"] || !s["b"] || !s["c"] {
		t.Error("makeSet should contain all input values")
	}
	if s["d"] {
		t.Error("makeSet should not contain 'd'")
	}
}

func TestMCPNames(t *testing.T) {
	mcps := []model.MCPAssignment{
		{Name: "github"},
		{Name: "postgres"},
	}
	names := mcpNames(mcps)
	if len(names) != 2 || names[0] != "github" || names[1] != "postgres" {
		t.Errorf("mcpNames = %v, want [github postgres]", names)
	}
}
