package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// helpTab is the Help tab — shows keybindings and CLI reference.
type helpTab struct {
	keys      KeyMap
	version   string
	commit    string
	buildDate string
	width     int
	height    int
	scroll    int
}

func newHelpTab(keys KeyMap, version, commit, buildDate string) *helpTab {
	return &helpTab{
		keys:      keys,
		version:   version,
		commit:    commit,
		buildDate: buildDate,
	}
}

func (t *helpTab) Title() string { return "Help" }

func (t *helpTab) HelpKeys() []HelpEntry {
	return []HelpEntry{
		{"q", "Quit"},
	}
}

func (t *helpTab) Init() tea.Cmd { return nil }

func (t *helpTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, t.keys.ListUp):
			if t.scroll > 0 {
				t.scroll--
			}
		case key.Matches(msg, t.keys.ListDown):
			t.scroll++
		}
	}
	return t, nil
}

func (t *helpTab) View() string {
	var b strings.Builder

	fmt.Fprintf(&b, "  hystak %s (%s, %s)\n\n", t.version, t.commit, t.buildDate)

	section := func(title string) {
		b.WriteString("  " + styleTitle.Render(title) + "\n")
	}
	entry := func(key, desc string) {
		fmt.Fprintf(&b, "  %-20s  %s\n", styleHelpKey.Render(key), desc)
	}

	section("Navigation")
	entry("Tab / Shift+Tab", "Switch tabs")
	entry("Up / Down", "Navigate lists")
	entry("Enter", "Select / confirm")
	entry("Esc", "Cancel / close overlay")
	entry("Space", "Toggle selection")
	entry("/", "Filter mode")
	entry("q", "Quit")
	b.WriteString("\n")

	section("Actions")
	entry("A", "Add new item")
	entry("E", "Edit selected item")
	entry("D", "Delete selected item")
	entry("I", "Import (Registry tab)")
	entry("L", "Launch (Projects tab)")
	entry("S", "Sync (Projects tab)")
	entry("P", "Preview / Profile")
	b.WriteString("\n")

	section("CLI Commands")
	entry("hystak", "Launch TUI")
	entry("hystak setup", "Re-run first-time setup")
	entry("hystak list", "List registry servers")
	entry("hystak sync <project>", "Sync project configs")
	entry("hystak diff <project>", "Show config drift")
	entry("hystak run <project>", "Sync + launch Claude Code")
	entry("hystak version", "Show version info")
	b.WriteString("\n")

	fmt.Fprintf(&b, "  Config: %s\n", "~/.hystak/")

	content := b.String()
	lines := strings.Split(content, "\n")

	// Apply scroll
	start := t.scroll
	if start > len(lines) {
		start = len(lines)
	}
	visible := lines[start:]

	return strings.Join(visible, "\n")
}
