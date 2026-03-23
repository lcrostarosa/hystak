package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

// diffViewModel displays drift results for a project.
type diffViewModel struct {
	results []service.DiffResult
	cursor  int
	keys    KeyMap
}

func newDiffViewModel(results []service.DiffResult, keys KeyMap) diffViewModel {
	return diffViewModel{results: results, keys: keys}
}

func (m diffViewModel) helpKeys() []HelpEntry {
	return []HelpEntry{{"Esc", "Close"}}
}

func (m diffViewModel) update(msg tea.KeyMsg) diffViewModel {
	switch {
	case key.Matches(msg, m.keys.ListUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.ListDown):
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
	}
	return m
}

func (m diffViewModel) view(width, height int) string {
	var b strings.Builder

	if len(m.results) == 0 {
		b.WriteString("  No drift detected.\n")
		b.WriteString("\n  " + styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Close"))
		content := b.String()
		box := styleOverlayBorder.Width(min(width-4, 60)).Render(content)
		return centerOverlay(box, width, height)
	}

	b.WriteString(styleListHeader.Render(fmt.Sprintf("  %-20s  %s", "SERVER", "STATUS")) + "\n")

	for i, r := range m.results {
		icon := statusIcon(r.Status)
		line := fmt.Sprintf("  %s %-20s  %s", icon, truncate(r.ServerName, 20), r.Status)
		if i == m.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Show detail for selected server
	if m.cursor < len(m.results) {
		r := m.results[m.cursor]
		if r.Status == model.DriftDrifted {
			b.WriteString("\n" + styleDrifted.Render("  --- "+r.ServerName+": diff ---") + "\n")
			b.WriteString(renderServerDiff(r.Expected, r.Deployed))
		}
	}

	b.WriteString("\n  " + styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Close"))

	content := b.String()
	box := styleOverlayBorder.Width(min(width-4, 70)).Render(content)
	return centerOverlay(box, width, height)
}

func statusIcon(s model.DriftStatus) string {
	switch s {
	case model.DriftSynced:
		return styleSynced.Render("✓")
	case model.DriftDrifted:
		return styleDrifted.Render("~")
	case model.DriftMissing:
		return styleMissing.Render("+")
	case model.DriftUnmanaged:
		return styleUnmanaged.Render("?")
	default:
		return " "
	}
}

func renderServerDiff(expected, deployed model.ServerDef) string {
	var b strings.Builder
	diffField := func(name, exp, dep string) {
		if exp != dep {
			b.WriteString(styleError.Render(fmt.Sprintf("  - %s: %s\n", name, dep)))
			b.WriteString(styleSynced.Render(fmt.Sprintf("  + %s: %s\n", name, exp)))
		}
	}
	diffField("command", expected.Command, deployed.Command)
	diffField("url", expected.URL, deployed.URL)
	diffField("transport", string(expected.Transport), string(deployed.Transport))
	if !slicesEqualStr(expected.Args, deployed.Args) {
		b.WriteString(styleError.Render(fmt.Sprintf("  - args: %v\n", deployed.Args)))
		b.WriteString(styleSynced.Render(fmt.Sprintf("  + args: %v\n", expected.Args)))
	}
	return b.String()
}

func slicesEqualStr(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
