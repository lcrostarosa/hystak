package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// envEntry is a key-value pair in the env editor.
type envEntry struct {
	key   string
	value string
}

// envEditorModel is a key-value editor for environment variables (S-062).
type envEditorModel struct {
	keys    KeyMap
	entries []envEntry
	cursor  int
	done    bool

	// Add mode
	adding   bool
	keyInput textinput.Model
	valInput textinput.Model
	addFocus int // 0=key, 1=value
}

func newEnvEditorModel(keys KeyMap, envVars map[string]string) envEditorModel {
	entries := make([]envEntry, 0, len(envVars))
	sortedKeys := make([]string, 0, len(envVars))
	for k := range envVars {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		entries = append(entries, envEntry{key: k, value: envVars[k]})
	}

	ki := textinput.New()
	ki.Placeholder = "KEY"
	ki.CharLimit = 128

	vi := textinput.New()
	vi.Placeholder = "value"
	vi.CharLimit = 256

	return envEditorModel{
		keys:     keys,
		entries:  entries,
		keyInput: ki,
		valInput: vi,
	}
}

func (m envEditorModel) helpKeys() []HelpEntry {
	if m.adding {
		return []HelpEntry{{"Tab", "Next"}, {"Enter", "Add"}, {"Esc", "Cancel"}}
	}
	return []HelpEntry{{"A", "Add"}, {"D", "Delete"}, {"Enter", "Done"}, {"Esc", "Cancel"}}
}

func (m envEditorModel) update(msg tea.KeyMsg) (envEditorModel, tea.Cmd) {
	if m.adding {
		return m.updateAdd(msg)
	}

	switch {
	case msg.String() == "esc" || key.Matches(msg, m.keys.Confirm):
		m.done = true
		return m, nil
	case key.Matches(msg, m.keys.ListUp):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.ListDown):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Add):
		m.adding = true
		m.keyInput.SetValue("")
		m.valInput.SetValue("")
		m.addFocus = 0
		return m, m.keyInput.Focus()
	case key.Matches(msg, m.keys.Delete):
		if len(m.entries) > 0 && m.cursor < len(m.entries) {
			m.entries = append(m.entries[:m.cursor], m.entries[m.cursor+1:]...)
			if m.cursor >= len(m.entries) && m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m envEditorModel) updateAdd(msg tea.KeyMsg) (envEditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.adding = false
		return m, nil
	case "tab":
		if m.addFocus == 0 {
			m.keyInput.Blur()
			m.addFocus = 1
			return m, m.valInput.Focus()
		}
		m.valInput.Blur()
		m.addFocus = 0
		return m, m.keyInput.Focus()
	case "enter":
		k := strings.TrimSpace(m.keyInput.Value())
		v := strings.TrimSpace(m.valInput.Value())
		if k != "" {
			m.entries = append(m.entries, envEntry{key: k, value: v})
		}
		m.adding = false
		return m, nil
	}

	var cmd tea.Cmd
	if m.addFocus == 0 {
		m.keyInput, cmd = m.keyInput.Update(msg)
	} else {
		m.valInput, cmd = m.valInput.Update(msg)
	}
	return m, cmd
}

func (m envEditorModel) toMap() map[string]string {
	result := make(map[string]string, len(m.entries))
	for _, e := range m.entries {
		result[e.key] = e.value
	}
	return result
}

func (m envEditorModel) view(width, height int) string {
	var b strings.Builder
	b.WriteString("Environment Variables\n\n")

	if len(m.entries) == 0 && !m.adding {
		b.WriteString("  (no env vars)\n")
	}

	for i, e := range m.entries {
		line := "  " + e.key + " = " + e.value
		if i == m.cursor && !m.adding {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	if m.adding {
		b.WriteString("\n")
		b.WriteString("  Key:   " + m.keyInput.View() + "\n")
		b.WriteString("  Value: " + m.valInput.View() + "\n")
	}

	b.WriteString("\n  ")
	if m.adding {
		b.WriteString(styleHelpKey.Render("Tab") + styleHelpDesc.Render(":Switch  "))
		b.WriteString(styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Add  "))
		b.WriteString(styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))
	} else {
		b.WriteString(styleHelpKey.Render("A") + styleHelpDesc.Render(":Add  "))
		b.WriteString(styleHelpKey.Render("D") + styleHelpDesc.Render(":Delete  "))
		b.WriteString(styleHelpKey.Render("Enter/Esc") + styleHelpDesc.Render(":Done"))
	}

	content := b.String()
	box := styleOverlayBorder.Width(min(width-4, 60)).Render(content)
	return centerOverlay(box, width, height)
}
