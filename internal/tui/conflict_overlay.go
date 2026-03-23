package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/service"
)

// conflictModel is the TUI conflict resolution overlay (S-047).
type conflictModel struct {
	keys      KeyMap
	conflicts []service.SyncConflict
	cursor    int
}

func newConflictModel(keys KeyMap, conflicts []service.SyncConflict) conflictModel {
	return conflictModel{
		keys:      keys,
		conflicts: conflicts,
	}
}

// --- Messages ---

type conflictResolvedMsg struct {
	conflicts []service.SyncConflict
}
type conflictCancelMsg struct{}

func (m conflictModel) helpKeys() []HelpEntry {
	return []HelpEntry{
		{"K", "Keep"},
		{"R", "Replace"},
		{"S", "Skip"},
		{"A", "Skip all"},
		{"Enter", "Apply"},
		{"Esc", "Cancel"},
	}
}

func (m conflictModel) update(msg tea.KeyMsg) (conflictModel, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		return m, func() tea.Msg { return conflictCancelMsg{} }
	case key.Matches(msg, m.keys.ListUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.ListDown):
		if m.cursor < len(m.conflicts)-1 {
			m.cursor++
		}
	case msg.String() == "k" || msg.String() == "K":
		if m.cursor < len(m.conflicts) {
			m.conflicts[m.cursor].Resolution = service.ConflictKeep
		}
	case msg.String() == "r" || msg.String() == "R":
		if m.cursor < len(m.conflicts) {
			m.conflicts[m.cursor].Resolution = service.ConflictReplace
		}
	case msg.String() == "s" || msg.String() == "S":
		if m.cursor < len(m.conflicts) {
			m.conflicts[m.cursor].Resolution = service.ConflictSkip
		}
	case msg.String() == "a" || msg.String() == "A":
		for i := range m.conflicts {
			if m.conflicts[i].Resolution == service.ConflictPending {
				m.conflicts[i].Resolution = service.ConflictSkip
			}
		}
	case key.Matches(msg, m.keys.Confirm):
		// Check all resolved
		for _, c := range m.conflicts {
			if c.Resolution == service.ConflictPending {
				return m, nil // still pending, don't submit
			}
		}
		resolved := make([]service.SyncConflict, len(m.conflicts))
		copy(resolved, m.conflicts)
		return m, func() tea.Msg { return conflictResolvedMsg{conflicts: resolved} }
	}
	return m, nil
}

func (m conflictModel) view(width, height int) string {
	var b strings.Builder
	b.WriteString("Sync Conflicts\n\n")
	b.WriteString("  The following files already exist and are not managed\n")
	b.WriteString("  by hystak. Choose how to resolve each conflict:\n\n")

	for i, c := range m.conflicts {
		status := ""
		switch c.Resolution {
		case service.ConflictPending:
			status = styleDrifted.Render("pending")
		case service.ConflictKeep:
			status = styleHelpDesc.Render("keep")
		case service.ConflictReplace:
			status = styleSynced.Render("replace")
		case service.ConflictSkip:
			status = styleHelpDesc.Render("skip")
		}

		line := fmt.Sprintf("  %d. %s  [%s]", i+1, truncate(c.Path, 40), status)
		if i == m.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
		b.WriteString("     " + styleHelpDesc.Render(c.Message) + "\n\n")
	}

	b.WriteString("  " +
		styleHelpKey.Render("K") + styleHelpDesc.Render(":Keep  ") +
		styleHelpKey.Render("R") + styleHelpDesc.Render(":Replace  ") +
		styleHelpKey.Render("S") + styleHelpDesc.Render(":Skip  ") +
		styleHelpKey.Render("A") + styleHelpDesc.Render(":Skip all  ") +
		styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Apply  ") +
		styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Abort"))

	content := b.String()
	box := styleOverlayBorder.Width(min(width-4, 70)).Render(content)
	return centerOverlay(box, width, height)
}
