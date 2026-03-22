package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/service"
)

// conflictDescriptions maps resource types to their conflict description renderers.
// Adding a new resource type with conflicts only requires adding an entry here.
var conflictDescriptions = map[string]func(name string) string{
	"skill": func(name string) string {
		return fmt.Sprintf("A skill named %s already exists in this project\nbut was NOT placed by hystak.\n", detailTitleStyle.Render(name))
	},
	"hook": func(_ string) string {
		return "The \"hooks\" key already exists in settings.local.json\nbut was NOT placed by hystak.\n"
	},
	"permission": func(_ string) string {
		return "The \"permissions\" key already exists in settings.local.json\nbut was NOT placed by hystak.\n"
	},
	"claude_md": func(_ string) string {
		return "CLAUDE.md already exists in this project\nbut was NOT placed by hystak.\n"
	},
}

// RequestConflictResolveMsg is sent when sync detects conflicts that need resolution.
type RequestConflictResolveMsg struct {
	ProjectName string
	Conflicts   []service.SyncConflict
}

// ConflictResolvedMsg is sent when all conflicts have been assigned a resolution.
type ConflictResolvedMsg struct {
	ProjectName string
	Resolutions []service.SyncConflict
}

// ConflictCancelledMsg is sent when the user cancels conflict resolution.
type ConflictCancelledMsg struct{}

// ConflictModel is the overlay for resolving sync conflicts one at a time.
type ConflictModel struct {
	projectName string
	conflicts   []service.SyncConflict
	current     int
	err         string
	width       int
	height      int
}

// NewConflictModel creates a new ConflictModel for the given conflicts.
func NewConflictModel(projectName string, conflicts []service.SyncConflict) ConflictModel {
	return ConflictModel{
		projectName: projectName,
		conflicts:   conflicts,
		current:     0,
	}
}

// SetSize updates dimensions for the conflict overlay.
func (m *ConflictModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key input for the conflict overlay.
func (m ConflictModel) Update(msg tea.Msg) (ConflictModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, func() tea.Msg { return ConflictCancelledMsg{} }

	case "k":
		m.conflicts[m.current].Resolution = service.ConflictKeep
		return m.advance()

	case "r":
		m.conflicts[m.current].Resolution = service.ConflictReplace
		return m.advance()

	case "s":
		m.conflicts[m.current].Resolution = service.ConflictSkip
		return m.advance()
	}

	return m, nil
}

// advance moves to the next conflict or finishes when all are resolved.
func (m ConflictModel) advance() (ConflictModel, tea.Cmd) {
	m.current++
	if m.current >= len(m.conflicts) {
		resolutions := make([]service.SyncConflict, len(m.conflicts))
		copy(resolutions, m.conflicts)
		return m, func() tea.Msg {
			return ConflictResolvedMsg{
				ProjectName: m.projectName,
				Resolutions: resolutions,
			}
		}
	}
	return m, nil
}

// View renders the conflict resolution overlay.
func (m ConflictModel) View() string {
	if len(m.conflicts) == 0 {
		return ""
	}

	c := m.conflicts[m.current]
	var b strings.Builder

	title := fmt.Sprintf("Sync Conflict: %s %q", c.ResourceType, c.Name)
	b.WriteString(conflictTitleStyle.Render(title))
	b.WriteString("\n\n")

	if desc, ok := conflictDescriptions[c.ResourceType]; ok {
		b.WriteString(desc(c.Name))
	} else {
		b.WriteString(fmt.Sprintf("Resource %q (%s) already exists\n", c.Name, c.ResourceType))
		b.WriteString("but was NOT placed by hystak.\n")
	}

	b.WriteString("\n")
	b.WriteString(formLabelStyle.Render("Existing: "))
	b.WriteString(c.ExistingPath)
	b.WriteString("\n\n")

	b.WriteString(conflictOptionStyle.Render("(k)"))
	b.WriteString(" Keep existing — don't deploy registry version\n")
	b.WriteString(conflictOptionStyle.Render("(r)"))
	b.WriteString(" Replace — overwrite with registry version\n")
	b.WriteString(conflictOptionStyle.Render("(s)"))
	b.WriteString(" Skip — remove this assignment from the profile\n")
	b.WriteString("\n")
	b.WriteString(formHintStyle.Render(fmt.Sprintf("Conflict %d of %d", m.current+1, len(m.conflicts))))
	b.WriteString("\n\n")
	b.WriteString(formHintStyle.Render("k: keep | r: replace | s: skip | esc: cancel"))

	if m.err != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(m.err))
	}

	formWidth := clamp(m.width-4, 50, 72)
	content := conflictBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

var (
	conflictTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("202")).
				MarginBottom(1)

	conflictOptionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("220"))

	conflictBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("202")).
				Padding(1, 2)
)
