package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
)

func TestProjectsTab_LoadData(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt = applyProjectsWindowSize(t, pt, 100, 30)

	msg := projectsLoadedMsg{
		projects: []model.Project{
			{Name: "myproject", Path: "/test/myproject", ActiveProfile: "dev"},
		},
	}
	updated, _ := pt.Update(msg)
	pt = updated.(*projectsTab)

	view := pt.View()
	if !strings.Contains(view, "myproject") {
		t.Error("view should contain project name after loading")
	}
}

func TestProjectsTab_CursorNavigation(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt = applyProjectsWindowSize(t, pt, 100, 30)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "aaa", Path: "/a", ActiveProfile: "dev"},
			{Name: "bbb", Path: "/b", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)

	// Load detail to avoid nil issues
	detailMsg := projectDetailMsg{
		profileNames: []string{"dev", "empty"},
		allMCPs:      []string{"github"},
	}
	updated, _ := pt.Update(detailMsg)
	pt = updated.(*projectsTab)

	// Both items should be visible
	view := pt.View()
	if !strings.Contains(view, "aaa") {
		t.Error("view should contain aaa")
	}
	if !strings.Contains(view, "bbb") {
		t.Error("view should contain bbb")
	}

	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ = pt.Update(downMsg)
	pt = updated.(*projectsTab)

	view = pt.View()
	if !strings.Contains(view, "bbb") {
		t.Error("after down, view should still contain bbb")
	}
}

func TestProjectsTab_PaneSwitch(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt = applyProjectsWindowSize(t, pt, 100, 30)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "proj", Path: "/test", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)

	detailMsg := projectDetailMsg{
		profileNames: []string{"dev"},
		allMCPs:      []string{"github"},
	}
	updated, _ := pt.Update(detailMsg)
	pt = updated.(*projectsTab)

	// Enter moves to detail pane
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ = pt.Update(enterMsg)
	pt = updated.(*projectsTab)

	// Esc returns to projects pane
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = pt.Update(escMsg)
	pt = updated.(*projectsTab)

	// View should still show project list
	view := pt.View()
	if !strings.Contains(view, "proj") {
		t.Error("view should contain project name after returning to projects pane")
	}
}

func TestProjectsTab_ProfilePicker(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt = applyProjectsWindowSize(t, pt, 100, 30)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "proj", Path: "/test", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)

	detailMsg := projectDetailMsg{
		profileNames: []string{"dev", "review", "empty"},
	}
	updated, _ := pt.Update(detailMsg)
	pt = updated.(*projectsTab)

	// Press P to open picker
	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
	updated, _ = pt.Update(pMsg)
	pt = updated.(*projectsTab)

	// View should show profile picker
	view := pt.View()
	if !strings.Contains(view, "Select Profile") {
		t.Error("profile picker view should contain 'Select Profile'")
	}
	if !strings.Contains(view, "dev") {
		t.Error("profile picker should list 'dev'")
	}
	if !strings.Contains(view, "review") {
		t.Error("profile picker should list 'review'")
	}

	// Esc closes picker
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = pt.Update(escMsg)
	pt = updated.(*projectsTab)

	// View should return to normal
	view = pt.View()
	if strings.Contains(view, "Select Profile") {
		t.Error("profile picker should be closed after Esc")
	}
}

func TestProjectsTab_View_TwoPane(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt = applyProjectsWindowSize(t, pt, 100, 30)

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

func TestProjectsTab_ErrorMsg_ShowsInView(t *testing.T) {
	app := setupTestApp(t)
	pt := app.tabs[TabProjects].(*projectsTab)
	pt = applyProjectsWindowSize(t, pt, 100, 30)

	loaded, _ := pt.Update(projectsLoadedMsg{
		projects: []model.Project{
			{Name: "proj", Path: "/test", ActiveProfile: "dev"},
		},
	})
	pt = loaded.(*projectsTab)

	detailMsg := projectDetailMsg{profileNames: []string{"dev"}}
	updated, _ := pt.Update(detailMsg)
	pt = updated.(*projectsTab)

	// Send error message
	errMsg := projectsErrorMsg{err: fmt.Errorf("project sync failed")}
	updated, _ = pt.Update(errMsg)
	pt = updated.(*projectsTab)

	view := pt.View()
	if !strings.Contains(view, "project sync failed") {
		t.Error("error message should appear in View()")
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

// applyProjectsWindowSize sends a WindowSizeMsg and returns the updated projectsTab.
func applyProjectsWindowSize(t *testing.T, pt *projectsTab, w, h int) *projectsTab {
	t.Helper()
	updated, _ := pt.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(*projectsTab)
}
