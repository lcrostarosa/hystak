package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// DiscoveredResource is a resource found in a project directory during scanning.
type DiscoveredResource struct {
	Type     string // "skill"
	Name     string
	Source   string // absolute path to the discovered file
	Conflict bool   // true if a resource with this name already exists in the registry
}

// discoveryPhase tracks whether scanning is in progress or results are shown.
type discoveryPhase int

const (
	discoveryScanning discoveryPhase = iota
	discoveryPreview
)

// RequestDiscoveryMsg is sent by the profiles tab to open the discovery overlay.
type RequestDiscoveryMsg struct {
	ProjectName string
	ProjectPath string
}

// DiscoveryCompletedMsg is sent after successful import.
type DiscoveryCompletedMsg struct {
	Imported int
}

// DiscoveryCancelledMsg is sent when the user cancels the discovery overlay.
type DiscoveryCancelledMsg struct{}

// DiscoveryModel is the overlay for discovering existing config in a project directory.
type DiscoveryModel struct {
	service     *service.Service
	projectName string
	projectPath string
	resources   []DiscoveredResource
	selected    []bool
	cursor      int
	phase       discoveryPhase
	err         string
	width       int
	height      int
}

// NewDiscoveryModel creates a DiscoveryModel and immediately scans for resources.
func NewDiscoveryModel(svc *service.Service, projectName, projectPath string) DiscoveryModel {
	m := DiscoveryModel{
		service:     svc,
		projectName: projectName,
		projectPath: projectPath,
		phase:       discoveryScanning,
	}
	m.scan()
	return m
}

// scan discovers skills in the project directory.
func (m *DiscoveryModel) scan() {
	skillsDir := filepath.Join(m.projectPath, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		// Directory doesn't exist — nothing to discover.
		m.phase = discoveryPreview
		return
	}

	var resources []DiscoveredResource
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		skillFile := filepath.Join(skillsDir, name, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue // no SKILL.md in this directory
		}

		_, conflict := m.service.GetSkill(name)
		resources = append(resources, DiscoveredResource{
			Type:     "skill",
			Name:     name,
			Source:   skillFile,
			Conflict: conflict,
		})
	}

	m.resources = resources
	m.selected = make([]bool, len(resources))
	// Pre-select non-conflicting resources.
	for i, r := range resources {
		m.selected[i] = !r.Conflict
	}
	m.phase = discoveryPreview
}

// SetSize updates dimensions for the discovery overlay.
func (m *DiscoveryModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key input for the discovery overlay.
func (m DiscoveryModel) Update(msg tea.Msg) (DiscoveryModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, func() tea.Msg { return DiscoveryCancelledMsg{} }

	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, len(m.resources))
		return m, nil

	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, len(m.resources))
		return m, nil

	case " ":
		if m.cursor < len(m.selected) && !m.resources[m.cursor].Conflict {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
		return m, nil

	case "enter":
		return m.applyImport()
	}

	return m, nil
}

// applyImport adds selected non-conflicting skills to the registry and assigns them.
func (m DiscoveryModel) applyImport() (DiscoveryModel, tea.Cmd) {
	imported := 0
	for i, r := range m.resources {
		if !m.selected[i] || r.Conflict {
			continue
		}
		skill := model.SkillDef{
			Name:   r.Name,
			Source: r.Source,
		}
		if err := m.service.AddSkill(skill); err != nil {
			m.err = fmt.Sprintf("adding skill %q: %v", r.Name, err)
			return m, nil
		}
		// Assign the newly imported skill to the project.
		if err := m.service.AssignSkill(m.projectName, r.Name); err != nil {
			// Non-fatal: skill was added to registry; assignment failed.
			m.err = fmt.Sprintf("assigning skill %q: %v", r.Name, err)
		}
		imported++
	}

	count := imported
	return m, func() tea.Msg { return DiscoveryCompletedMsg{Imported: count} }
}

// View renders the discovery overlay.
func (m DiscoveryModel) View() string {
	var b strings.Builder

	b.WriteString(formTitleStyle.Render("Discover Config"))
	b.WriteString("\n")
	b.WriteString(formHintStyle.Render(m.projectPath))
	b.WriteString("\n\n")

	if len(m.resources) == 0 {
		b.WriteString(sectionDimStyle.Render("No discoverable resources found."))
		b.WriteString("\n")
		b.WriteString(sectionDimStyle.Render("(Skills require a SKILL.md inside .claude/skills/<name>/)"))
		b.WriteString("\n\n")
		b.WriteString(formHintStyle.Render("esc: cancel"))
	} else {
		b.WriteString(formLabelStyle.Render("Skills:"))
		b.WriteString("\n")

		for i, r := range m.resources {
			cur := "  "
			if i == m.cursor {
				cur = "\u25b8 "
			}

			check := "[x]"
			if !m.selected[i] {
				check = "[ ]"
			}

			line := fmt.Sprintf("%s%s %-24s", cur, check, r.Name)

			if r.Conflict {
				line += " " + conflictWarningStyle.Render("(!) exists in registry")
			} else {
				line += " " + sectionDimStyle.Render("(new)")
			}

			b.WriteString(line)
			b.WriteString("\n")
		}

		b.WriteString("\n")

		if m.err != "" {
			b.WriteString(errorStyle.Render(m.err))
			b.WriteString("\n\n")
		}

		b.WriteString(formHintStyle.Render("enter: import selected | space: toggle | esc: cancel"))
	}

	formWidth := clamp(m.width-4, 50, 76)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

var conflictWarningStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("220")).
	Bold(true)
