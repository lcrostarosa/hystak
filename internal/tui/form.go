package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// FormField describes a single field in a form overlay.
type FormField struct {
	Label       string
	Placeholder string
	Value       string
}

// FormResult is returned when the user submits or cancels the form.
type FormResult struct {
	Submitted bool
	Values    map[string]string // keyed by label
}

// formModel is a reusable modal form with labeled text inputs.
// All I/O is async via tea.Cmd — no I/O in Update.
type formModel struct {
	title  string
	fields []textinput.Model
	labels []string
	focus  int
	width  int
	height int
	keys   KeyMap
}

// newFormModel creates a form overlay. Returns in "ready" state.
func newFormModel(title string, fields []FormField, keys KeyMap) formModel {
	inputs := make([]textinput.Model, len(fields))
	labels := make([]string, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		ti.SetValue(f.Value)
		ti.CharLimit = 256
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
		labels[i] = f.Label
	}
	return formModel{
		title:  title,
		fields: inputs,
		labels: labels,
		keys:   keys,
	}
}

func (m formModel) Init() tea.Cmd {
	return textinput.Blink
}

// formSubmitMsg signals the form was submitted.
type formSubmitMsg struct {
	values map[string]string
}

// formCancelMsg signals the form was cancelled.
type formCancelMsg struct{}

func (m formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "esc":
			return m, func() tea.Msg { return formCancelMsg{} }

		case msg.String() == "enter":
			// If on last field, submit
			if m.focus == len(m.fields)-1 {
				return m, m.submit
			}
			// Otherwise move to next field
			return m, m.nextField()

		case msg.String() == "tab":
			return m, m.nextField()

		case msg.String() == "shift+tab":
			return m, m.prevField()
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
	return m, cmd
}

func (m formModel) submit() tea.Msg {
	values := make(map[string]string, len(m.fields))
	for i, f := range m.fields {
		values[m.labels[i]] = f.Value()
	}
	return formSubmitMsg{values: values}
}

func (m *formModel) nextField() tea.Cmd {
	m.fields[m.focus].Blur()
	m.focus = (m.focus + 1) % len(m.fields)
	return m.fields[m.focus].Focus()
}

func (m *formModel) prevField() tea.Cmd {
	m.fields[m.focus].Blur()
	m.focus = (m.focus - 1 + len(m.fields)) % len(m.fields)
	return m.fields[m.focus].Focus()
}

func (m formModel) View(width, height int) string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(m.title))
	b.WriteString("\n\n")

	for i, f := range m.fields {
		label := m.labels[i]
		cursor := " "
		if i == m.focus {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf("  %s %-14s %s\n", cursor, label+":", f.View()))
	}

	b.WriteString("\n")
	b.WriteString("  " +
		styleHelpKey.Render("Tab") + styleHelpDesc.Render(":Next  ") +
		styleHelpKey.Render("Shift+Tab") + styleHelpDesc.Render(":Prev  ") +
		styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Save  ") +
		styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	content := b.String()
	boxWidth := min(width-4, 70)
	if boxWidth < 40 {
		boxWidth = 40
	}
	box := styleOverlayBorder.Width(boxWidth).Render(content)
	return centerOverlay(box, width, height)
}
