package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// importPhase tracks where the user is in the import flow.
type importPhase int

const (
	phaseInputPath importPhase = iota
	phasePreview
	phaseConflict
)

var cursorResolution = map[int]service.ImportResolution{
	0: service.ImportKeep,
	1: service.ImportReplace,
	2: service.ImportSkip,
}

// ImportCompletedMsg is sent when the import flow finishes successfully.
type ImportCompletedMsg struct {
	Imported int
}

// ImportCancelledMsg is sent when the user cancels the import flow.
type ImportCancelledMsg struct{}

// RequestImportMsg is sent by the servers tab to open the import overlay.
type RequestImportMsg struct{}

// ImportModel is the overlay for importing servers from client config files.
type ImportModel struct {
	service    *service.Service
	phase      importPhase
	pathInput  textinput.Model
	candidates []service.ImportCandidate
	selected   []bool // per-candidate selection (preview phase)
	cursor     int    // cursor position in preview/conflict lists
	conflictAt int    // index of the current conflict being resolved
	renameInput textinput.Model
	err        string
	width      int
	height     int
}

// NewImportModel creates a new import flow model.
func NewImportModel(svc *service.Service) ImportModel {
	pi := textinput.New()
	pi.Placeholder = "path/to/.mcp.json or ~/.claude.json"
	pi.Prompt = "  "
	pi.CharLimit = 512
	pi.Focus()

	ri := textinput.New()
	ri.Placeholder = "new-server-name"
	ri.Prompt = "  "
	ri.CharLimit = 256

	return ImportModel{
		service:     svc,
		phase:       phaseInputPath,
		pathInput:   pi,
		renameInput: ri,
	}
}

// SetSize updates dimensions for the import overlay.
func (m *ImportModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 70)
	m.pathInput.Width = inputWidth
	m.renameInput.Width = inputWidth
}

// Update handles messages for the import overlay.
func (m ImportModel) Update(msg tea.Msg) (ImportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.phase {
		case phaseInputPath:
			return m.updateInputPath(msg)
		case phasePreview:
			return m.updatePreview(msg)
		case phaseConflict:
			return m.updateConflict(msg)
		}
	}

	// Forward to active text input.
	switch m.phase {
	case phaseInputPath:
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	case phaseConflict:
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ImportModel) updateInputPath(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return ImportCancelledMsg{} }
	case "enter":
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			m.err = "Path is required"
			return m, nil
		}
		candidates, err := m.service.ImportFromFile(path)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		if len(candidates) == 0 {
			m.err = "No MCPs found in file"
			return m, nil
		}
		m.err = ""
		m.candidates = candidates
		m.selected = make([]bool, len(candidates))
		for i := range m.selected {
			m.selected[i] = true
		}
		m.cursor = 0
		m.phase = phasePreview
		return m, nil
	}

	// Forward to text input.
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m ImportModel) updatePreview(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return ImportCancelledMsg{} }
	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, len(m.candidates))
		return m, nil
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, len(m.candidates))
		return m, nil
	case " ":
		if m.cursor < len(m.selected) {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
		return m, nil
	case "enter":
		return m.applyPreview()
	}
	return m, nil
}

// applyPreview processes selected candidates: non-conflicting are marked,
// then advances to conflict resolution if any selected candidates have conflicts.
func (m ImportModel) applyPreview() (ImportModel, tea.Cmd) {
	// Filter to only selected candidates.
	var kept []service.ImportCandidate
	for i, c := range m.candidates {
		if m.selected[i] {
			kept = append(kept, c)
		}
	}

	if len(kept) == 0 {
		m.err = "No MCPs selected"
		return m, nil
	}

	m.candidates = kept

	// Check for conflicts among selected.
	if idx := m.nextConflict(0); idx >= 0 {
		m.conflictAt = idx
		m.cursor = 0
		m.renameInput.SetValue("")
		m.phase = phaseConflict
		return m, nil
	}

	// No conflicts — apply immediately.
	return m.finishImport()
}

func (m ImportModel) updateConflict(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return ImportCancelledMsg{} }
	case "k", "up":
		m.cursor = moveCursor(m.cursor, -1, 4)
		return m, nil
	case "s":
		m.candidates[m.conflictAt].Resolution = service.ImportSkip
		return m.advanceConflict()
	case "r":
		m.candidates[m.conflictAt].Resolution = service.ImportReplace
		return m.advanceConflict()
	case "n":
		m.renameInput.SetValue("")
		m.renameInput.Focus()
		m.cursor = 3 // rename option
		return m, nil
	case "enter":
		if m.cursor == 3 {
			// Rename — validate
			newName := strings.TrimSpace(m.renameInput.Value())
			if newName == "" {
				m.err = "Name is required for rename"
				return m, nil
			}
			m.candidates[m.conflictAt].Resolution = service.ImportRename
			m.candidates[m.conflictAt].RenameTo = newName
			m.err = ""
			return m.advanceConflict()
		}
		// Map cursor position to resolution.
		if res, ok := cursorResolution[m.cursor]; ok {
			m.candidates[m.conflictAt].Resolution = res
		}
		return m.advanceConflict()
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, 4)
		return m, nil
	}

	// If on rename option, forward to rename input.
	if m.cursor == 3 {
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// nextConflict returns the index of the next conflicting candidate starting at from, or -1.
func (m ImportModel) nextConflict(from int) int {
	for i := from; i < len(m.candidates); i++ {
		if m.candidates[i].Conflict {
			return i
		}
	}
	return -1
}

// advanceConflict moves to the next unresolved conflict or finishes the import.
func (m ImportModel) advanceConflict() (ImportModel, tea.Cmd) {
	next := m.nextConflict(m.conflictAt + 1)
	if next >= 0 {
		m.conflictAt = next
		m.cursor = 0
		m.renameInput.SetValue("")
		m.err = ""
		return m, nil
	}
	return m.finishImport()
}

// finishImport applies all resolved candidates to the registry.
func (m ImportModel) finishImport() (ImportModel, tea.Cmd) {
	if err := m.service.ApplyImport(m.candidates); err != nil {
		m.err = err.Error()
		return m, nil
	}
	imported := 0
	for _, c := range m.candidates {
		if c.WasImported() {
			imported++
		}
	}
	return m, func() tea.Msg { return ImportCompletedMsg{Imported: imported} }
}

// View renders the import overlay.
func (m ImportModel) View() string {
	var b strings.Builder

	switch m.phase {
	case phaseInputPath:
		m.renderInputPath(&b)
	case phasePreview:
		m.renderPreview(&b)
	case phaseConflict:
		m.renderConflict(&b)
	}

	formWidth := clamp(m.width-4, 40, 70)

	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m ImportModel) renderInputPath(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Import MCPs"))
	b.WriteString("\n\n")
	b.WriteString(formLabelStyle.Render("Config file path"))
	b.WriteString("\n")
	b.WriteString(m.pathInput.View())
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(formHintStyle.Render("enter: load | esc: cancel"))
}

func (m ImportModel) renderPreview(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Select MCPs to Import"))
	b.WriteString("\n\n")

	for i, c := range m.candidates {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		check := "[x]"
		if !m.selected[i] {
			check = "[ ]"
		}

		conflict := ""
		if c.Conflict {
			conflict = " " + errorStyle.Render("(conflict)")
		}

		line := fmt.Sprintf("%s%s %s%s", cursor, check, c.Name, conflict)
		b.WriteString(line)
		b.WriteString("\n")

		// Show server details in a compact line.
		detail := formatServerCompact(c.Server)
		fmt.Fprintf(b, "       %s\n", formHintStyle.Render(detail))
	}

	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(formHintStyle.Render("space: toggle | enter: import | esc: cancel"))
}

func (m ImportModel) renderConflict(b *strings.Builder) {
	c := m.candidates[m.conflictAt]

	b.WriteString(formTitleStyle.Render("Resolve Conflict"))
	b.WriteString("\n\n")
	fmt.Fprintf(b, "MCP %s already exists in the registry.\n\n",
		detailTitleStyle.Render(c.Name))

	// Show imported server details.
	b.WriteString(formLabelStyle.Render("Imported:"))
	b.WriteString("\n")
	b.WriteString(formatServerDetail(c.Server))
	b.WriteString("\n")

	// Show existing server details.
	if existing, ok := m.service.GetServer(c.Name); ok {
		b.WriteString(formLabelStyle.Render("Existing:"))
		b.WriteString("\n")
		b.WriteString(formatServerDetail(existing))
		b.WriteString("\n")
	}

	// Options
	options := []string{"Keep existing", "Replace with imported", "Skip", "Rename imported"}
	for i, opt := range options {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		fmt.Fprintf(b, "%s%s\n", cursor, opt)
	}

	// Show rename input when on rename option.
	if m.cursor == 3 {
		b.WriteString("\n")
		b.WriteString(formLabelStyle.Render("New name"))
		b.WriteString("\n")
		b.WriteString(m.renameInput.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(formHintStyle.Render("enter: confirm | s: skip | r: replace | n: rename | esc: cancel"))
}

// formatServerCompact returns a one-line summary of a server.
func formatServerCompact(srv model.ServerDef) string {
	switch srv.Transport {
	case model.TransportStdio:
		if len(srv.Args) > 0 {
			return fmt.Sprintf("%s: %s %s", srv.Transport, srv.Command, strings.Join(srv.Args, " "))
		}
		return fmt.Sprintf("%s: %s", srv.Transport, srv.Command)
	case model.TransportSSE, model.TransportHTTP:
		return fmt.Sprintf("%s: %s", srv.Transport, srv.URL)
	}
	return string(srv.Transport)
}

// formatServerDetail returns a multi-line detail view of a server.
func formatServerDetail(srv model.ServerDef) string {
	var b strings.Builder
	writeServerFields(&b, srv, detailLabelStyle)
	return b.String()
}
