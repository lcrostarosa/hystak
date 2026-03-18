package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/service"
)

// RequestDiffMsg is sent by the projects tab to open the diff overlay.
type RequestDiffMsg struct {
	ProjectName string
}

// DiffClosedMsg is sent when the diff overlay is closed.
type DiffClosedMsg struct{}

// DiffModel is the overlay for viewing unified diffs with sync-to-resolve.
type DiffModel struct {
	service     *service.Service
	projectName string
	viewport    viewport.Model
	rawDiff     string
	synced      bool
	err         string
	width       int
	height      int
}

// NewDiffModel creates a new diff view model for a project.
func NewDiffModel(svc *service.Service, projectName string) DiffModel {
	vp := viewport.New(0, 0)
	vp.SetContent("")

	m := DiffModel{
		service:     svc,
		projectName: projectName,
		viewport:    vp,
	}
	m.loadDiff()
	return m
}

// SetSize updates dimensions for the diff overlay.
func (m *DiffModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	boxWidth := clamp(w-4, 40, 100)

	// viewport height: total height minus box border/padding (4) minus title/hints (5)
	vpHeight := h - 10
	if vpHeight < 4 {
		vpHeight = 4
	}

	m.viewport.Width = boxWidth - 4 // subtract padding
	m.viewport.Height = vpHeight
}

// loadDiff fetches the diff from the service and sets the viewport content.
func (m *DiffModel) loadDiff() {
	diff, err := m.service.Diff(m.projectName)
	if err != nil {
		m.err = err.Error()
		m.rawDiff = ""
		m.viewport.SetContent(errorStyle.Render(m.err))
		return
	}

	m.err = ""
	m.rawDiff = diff

	if diff == "" {
		m.viewport.SetContent(syncMsgStyle.Render("No drift detected — everything is in sync."))
		return
	}

	m.viewport.SetContent(colorizeDiff(diff))
}

// Update handles messages for the diff overlay.
func (m DiffModel) Update(msg tea.Msg) (DiffModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return DiffClosedMsg{} }
		case "s":
			m.synced = false
			m.err = ""
			_, err := m.service.SyncProject(m.projectName)
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.synced = true
			m.loadDiff()
			m.viewport.GotoTop()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the diff overlay.
func (m DiffModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Diff: %s", m.projectName)
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	if m.synced && m.rawDiff == "" {
		b.WriteString(syncMsgStyle.Render("Synced successfully!"))
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n")
	}

	b.WriteString(formHintStyle.Render("s: sync | ↑/↓: scroll | esc: close"))

	boxWidth := clamp(m.width-4, 40, 100)

	content := formBoxStyle.Width(boxWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// colorizeDiff applies syntax coloring to unified diff lines.
func colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	styled := make([]string, 0, len(lines))

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			styled = append(styled, diffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			styled = append(styled, diffHunkStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			styled = append(styled, diffAddStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			styled = append(styled, diffDelStyle.Render(line))
		default:
			styled = append(styled, line)
		}
	}

	return strings.Join(styled, "\n")
}
