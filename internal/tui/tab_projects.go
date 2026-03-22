package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

// projectsTab is the Projects tab — shows project list with detail pane.
type projectsTab struct {
	keys     KeyMap
	svc      *service.Service
	projects []model.Project
	cursor   int
	width    int
	height   int
}

func newProjectsTab(keys KeyMap, svc *service.Service) *projectsTab {
	return &projectsTab{
		keys: keys,
		svc:  svc,
	}
}

func (t *projectsTab) Title() string { return "Projects" }

func (t *projectsTab) HelpKeys() []HelpEntry {
	return []HelpEntry{
		{"A", "Add"},
		{"D", "Delete"},
		{"P", "Profile"},
		{"L", "Launch"},
		{"S", "Sync"},
	}
}

// projectsLoadedMsg is sent when project data has been loaded asynchronously.
type projectsLoadedMsg struct {
	projects []model.Project
}

func (t *projectsTab) Init() tea.Cmd {
	return t.loadData
}

func (t *projectsTab) loadData() tea.Msg {
	return projectsLoadedMsg{
		projects: t.svc.ListProjects(),
	}
}

func (t *projectsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		t.projects = msg.projects
		t.cursor = 0
		return t, nil

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t *projectsTab) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, t.keys.ListUp):
		if t.cursor > 0 {
			t.cursor--
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.cursor < len(t.projects)-1 {
			t.cursor++
		}
	}
	return t, nil
}

func (t *projectsTab) View() string {
	var b strings.Builder

	if len(t.projects) == 0 {
		b.WriteString("  No projects registered.\n")
		b.WriteString("  Use 'hystak setup' or add a project.\n")
		return b.String()
	}

	// Simple list for now; two-pane layout comes in Batch 11
	b.WriteString(styleListHeader.Render(
		fmt.Sprintf("  %-20s  %-12s  %s", "NAME", "PROFILE", "PATH"),
	))
	b.WriteString("\n")

	for i, p := range t.projects {
		prof := p.ActiveProfile
		if prof == "" {
			prof = "(none)"
		}
		line := fmt.Sprintf("  %-20s  %-12s  %s", truncate(p.Name, 20), truncate(prof, 12), truncate(p.Path, 40))
		if i == t.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(styleListNormal.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}
