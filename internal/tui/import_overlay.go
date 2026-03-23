package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/service"
)

// importMode tracks the import overlay state.
type importMode int

const (
	importModePathInput importMode = iota
	importModeCandidates
	importModeConflict
)

// importModel is the TUI import overlay (S-009).
type importModel struct {
	keys       KeyMap
	svc        *service.Service
	mode       importMode
	pathInput  textinput.Model
	candidates []service.ImportCandidate
	cursor     int
	selected   map[int]bool
	err        string

	// Conflict resolution
	conflictIdx int
}

func newImportModel(keys KeyMap, svc *service.Service) importModel {
	ti := textinput.New()
	ti.Placeholder = "path to .mcp.json or .claude.json"
	ti.CharLimit = 256
	ti.Focus()
	return importModel{
		keys:      keys,
		svc:       svc,
		pathInput: ti,
		selected:  make(map[int]bool),
	}
}

// --- Messages ---

type importScanDoneMsg struct {
	candidates []service.ImportCandidate
	err        error
}
type importApplyDoneMsg struct {
	imported int
	err      error
}
type importDismissMsg struct{}

func (m importModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m importModel) update(msg tea.Msg) (importModel, tea.Cmd) {
	switch msg := msg.(type) {
	case importScanDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.mode = importModePathInput
			return m, nil
		}
		m.candidates = msg.candidates
		m.cursor = 0
		m.selected = make(map[int]bool)
		// Pre-select non-conflicting candidates
		for i, c := range m.candidates {
			if !c.Conflict {
				m.selected[i] = true
			}
		}
		m.mode = importModeCandidates
		return m, nil

	case importApplyDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = fmt.Sprintf("Imported %d server(s)", msg.imported)
		return m, func() tea.Msg { return importDismissMsg{} }

	case tea.KeyMsg:
		switch m.mode {
		case importModePathInput:
			return m.handlePathKey(msg)
		case importModeCandidates:
			return m.handleCandidatesKey(msg)
		case importModeConflict:
			return m.handleConflictKey(msg)
		}
	}
	return m, nil
}

// --- Path input ---

func (m importModel) handlePathKey(msg tea.KeyMsg) (importModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return importDismissMsg{} }
	case "enter":
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			m.err = "path is required"
			return m, nil
		}
		svc := m.svc
		return m, func() tea.Msg {
			candidates, err := svc.PrepareImport(path)
			return importScanDoneMsg{candidates: candidates, err: err}
		}
	}
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

// --- Candidates list ---

func (m importModel) handleCandidatesKey(msg tea.KeyMsg) (importModel, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		return m, func() tea.Msg { return importDismissMsg{} }
	case key.Matches(msg, m.keys.ListUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.ListDown):
		if m.cursor < len(m.candidates)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Select):
		if m.cursor < len(m.candidates) {
			c := m.candidates[m.cursor]
			if c.Conflict {
				// Open conflict resolution for this candidate
				m.conflictIdx = m.cursor
				m.mode = importModeConflict
			} else {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		}
	case key.Matches(msg, m.keys.Confirm):
		return m, m.applyImport()
	}
	return m, nil
}

func (m importModel) applyImport() tea.Cmd {
	// Build resolved candidates
	resolved := make([]service.ImportCandidate, len(m.candidates))
	copy(resolved, m.candidates)

	for i := range resolved {
		if resolved[i].Conflict {
			if resolved[i].Resolution == service.ImportPending {
				resolved[i].Resolution = service.ImportSkip
			}
		} else if m.selected[i] {
			resolved[i].Resolution = service.ImportReplace
		} else {
			resolved[i].Resolution = service.ImportSkip
		}
	}

	svc := m.svc
	return func() tea.Msg {
		imported, err := svc.ApplyImport(resolved)
		return importApplyDoneMsg{imported: imported, err: err}
	}
}

// --- Conflict resolution ---

func (m importModel) handleConflictKey(msg tea.KeyMsg) (importModel, tea.Cmd) {
	switch msg.String() {
	case "k", "K":
		m.candidates[m.conflictIdx].Resolution = service.ImportKeep
		m.mode = importModeCandidates
	case "r", "R":
		m.candidates[m.conflictIdx].Resolution = service.ImportReplace
		m.selected[m.conflictIdx] = true
		m.mode = importModeCandidates
	case "s", "S":
		m.candidates[m.conflictIdx].Resolution = service.ImportSkip
		m.mode = importModeCandidates
	case "a", "A":
		// Apply same resolution (skip) to all remaining conflicts
		for i := range m.candidates {
			if m.candidates[i].Conflict && m.candidates[i].Resolution == service.ImportPending {
				m.candidates[i].Resolution = service.ImportSkip
			}
		}
		m.mode = importModeCandidates
	case "esc":
		m.mode = importModeCandidates
	}
	return m, nil
}

// --- View ---

func (m importModel) helpKeys() []HelpEntry {
	switch m.mode {
	case importModePathInput:
		return []HelpEntry{{"Enter", "Scan"}, {"Esc", "Cancel"}}
	case importModeConflict:
		return []HelpEntry{{"K", "Keep"}, {"R", "Replace"}, {"S", "Skip"}, {"A", "All skip"}}
	default:
		return []HelpEntry{{"Space", "Toggle"}, {"Enter", "Import"}, {"Esc", "Cancel"}}
	}
}

func (m importModel) view(width, height int) string {
	var b strings.Builder

	switch m.mode {
	case importModePathInput:
		b.WriteString("Import MCP Servers\n\n")
		b.WriteString("  Source file: " + m.pathInput.View() + "\n")
		if m.err != "" {
			b.WriteString("\n  " + styleError.Render(m.err) + "\n")
		}
		b.WriteString("\n  " +
			styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Scan  ") +
			styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	case importModeCandidates:
		b.WriteString("Import MCP Servers\n\n")
		b.WriteString(styleListHeader.Render(fmt.Sprintf("  %-2s %-20s  %-10s  %s", "", "NAME", "TRANSPORT", "STATUS")) + "\n")

		for i, c := range m.candidates {
			sel := "  "
			if m.selected[i] {
				sel = styleSynced.Render("x ")
			}
			status := ""
			if c.Conflict {
				switch c.Resolution {
				case service.ImportPending:
					status = styleDrifted.Render("conflict")
				case service.ImportKeep:
					status = styleHelpDesc.Render("keep")
				case service.ImportReplace:
					status = styleSynced.Render("replace")
				case service.ImportSkip:
					status = styleHelpDesc.Render("skip")
				}
			}
			line := fmt.Sprintf("  %s%-20s  %-10s  %s", sel, truncate(c.Name, 20), c.Server.Transport, status)
			if i == m.cursor {
				b.WriteString(styleListSelected.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}

		if m.err != "" {
			b.WriteString("\n  " + styleError.Render(m.err))
		}
		b.WriteString("\n  " +
			styleHelpKey.Render("Space") + styleHelpDesc.Render(":Toggle  ") +
			styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Import  ") +
			styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	case importModeConflict:
		c := m.candidates[m.conflictIdx]
		b.WriteString(fmt.Sprintf("Conflict: %q already exists in registry.\n\n", c.Name))
		b.WriteString("  " + styleHelpKey.Render("K") + styleHelpDesc.Render(":Keep existing  "))
		b.WriteString(styleHelpKey.Render("R") + styleHelpDesc.Render(":Replace  "))
		b.WriteString(styleHelpKey.Render("S") + styleHelpDesc.Render(":Skip  "))
		b.WriteString(styleHelpKey.Render("A") + styleHelpDesc.Render(":Skip all remaining"))
	}

	content := b.String()
	boxWidth := min(width-4, 70)
	if boxWidth < 40 {
		boxWidth = 40
	}
	box := styleOverlayBorder.Width(boxWidth).Render(content)
	return centerOverlay(box, width, height)
}
