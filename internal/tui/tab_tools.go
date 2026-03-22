package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// toolItem represents a single action on the Tools grid.
type toolItem struct {
	name string
	desc string
}

var toolItems = []toolItem{
	{"Import", "Import MCPs from file"},
	{"Discover", "Scan skills in project"},
	{"Diff", "Show config drift"},
	{"Doctor", "Validate registry"},
	{"Launch", "Sync + run Claude Code"},
	{"Backup", "Backup or restore"},
}

// toolsTab is the Tools tab — action grid.
type toolsTab struct {
	keys   KeyMap
	cursor int
	width  int
	height int
}

func newToolsTab(keys KeyMap) *toolsTab {
	return &toolsTab{keys: keys}
}

func (t *toolsTab) Title() string { return "Tools" }

func (t *toolsTab) HelpKeys() []HelpEntry {
	return []HelpEntry{
		{"Enter", "Select"},
	}
}

func (t *toolsTab) Init() tea.Cmd { return nil }

func (t *toolsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t *toolsTab) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cols := 2
	switch {
	case key.Matches(msg, t.keys.ListUp):
		if t.cursor >= cols {
			t.cursor -= cols
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.cursor+cols < len(toolItems) {
			t.cursor += cols
		}
	case msg.String() == "left":
		if t.cursor%cols > 0 {
			t.cursor--
		}
	case msg.String() == "right":
		if t.cursor%cols < cols-1 && t.cursor+1 < len(toolItems) {
			t.cursor++
		}
	}
	return t, nil
}

func (t *toolsTab) View() string {
	var b strings.Builder
	cols := 2
	colWidth := 24

	for row := 0; row < (len(toolItems)+1)/cols; row++ {
		// Name row
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(toolItems) {
				break
			}
			name := fmt.Sprintf("  %-*s", colWidth-2, toolItems[idx].name)
			if idx == t.cursor {
				b.WriteString(styleListSelected.Render(name))
			} else {
				b.WriteString(styleTabActive.Render(name))
			}
			b.WriteString("  ")
		}
		b.WriteString("\n")

		// Description row
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(toolItems) {
				break
			}
			desc := fmt.Sprintf("  %-*s", colWidth-2, toolItems[idx].desc)
			if idx == t.cursor {
				b.WriteString(styleListSelected.Render(desc))
			} else {
				b.WriteString(styleHelpDesc.Render(desc))
			}
			b.WriteString("  ")
		}
		b.WriteString("\n\n")
	}

	return b.String()
}
