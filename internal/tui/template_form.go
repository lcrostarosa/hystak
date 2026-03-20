package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// RequestTemplateFormMsg is sent to open the template add/edit overlay.
type RequestTemplateFormMsg struct {
	EditTemplate *model.TemplateDef // nil for add, non-nil for edit
}

// TemplateFormSubmittedMsg is sent when the template form is submitted with a valid TemplateDef.
type TemplateFormSubmittedMsg struct {
	Template model.TemplateDef
	IsEdit   bool
}

// TemplateFormCancelledMsg is sent when the template form is cancelled.
type TemplateFormCancelledMsg struct{}

// tmplFormField identifies a field in the template form.
type tmplFormField int

const (
	tmplFieldName tmplFormField = iota
	tmplFieldSource
	tmplFieldCount
)

// TemplateFormModel is the overlay for adding/editing templates.
type TemplateFormModel struct {
	inputs   [tmplFieldCount]textinput.Model
	focused  tmplFormField
	isEdit   bool
	editName string
	err      string
	width    int
	height   int
}

// NewTemplateFormModel creates a new form for adding a template.
func NewTemplateFormModel() TemplateFormModel {
	m := TemplateFormModel{}
	m.initInputs()
	return m
}

// NewEditTemplateFormModel creates a new form pre-populated for editing a template.
func NewEditTemplateFormModel(tmpl model.TemplateDef) TemplateFormModel {
	m := TemplateFormModel{
		isEdit:   true,
		editName: tmpl.Name,
	}
	m.initInputs()
	m.inputs[tmplFieldName].SetValue(tmpl.Name)
	m.inputs[tmplFieldSource].SetValue(tmpl.Source)
	m.focusField(tmplFieldName)
	return m
}

func (m *TemplateFormModel) initInputs() {
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].CharLimit = 256
		m.inputs[i].Prompt = "  "
	}

	m.inputs[tmplFieldName].Placeholder = "template-name"
	m.inputs[tmplFieldSource].Placeholder = "path/to/template.md"

	m.focusField(tmplFieldName)
}

// SetSize updates dimensions for the template form overlay.
func (m *TemplateFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 60)
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}

func (m *TemplateFormModel) focusField(f tmplFormField) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focused = f
	m.inputs[f].Focus()
}

func (m *TemplateFormModel) nextField() {
	if int(m.focused) < int(tmplFieldCount)-1 {
		m.focusField(m.focused + 1)
	}
}

func (m *TemplateFormModel) prevField() {
	if m.focused > 0 {
		m.focusField(m.focused - 1)
	}
}

var tmplFieldLabels = [tmplFieldCount]string{
	tmplFieldName:   "Name",
	tmplFieldSource: "Source",
}

// validate checks required fields and returns an error message or empty string.
func (m TemplateFormModel) validate() string {
	if strings.TrimSpace(m.inputs[tmplFieldName].Value()) == "" {
		return "Name is required"
	}
	if strings.TrimSpace(m.inputs[tmplFieldSource].Value()) == "" {
		return "Source is required"
	}
	return ""
}

// buildTemplateDef constructs a TemplateDef from the form fields.
func (m TemplateFormModel) buildTemplateDef() model.TemplateDef {
	return model.TemplateDef{
		Name:   strings.TrimSpace(m.inputs[tmplFieldName].Value()),
		Source: strings.TrimSpace(m.inputs[tmplFieldSource].Value()),
	}
}

// Update handles messages for the template form overlay.
func (m TemplateFormModel) Update(msg tea.Msg) (TemplateFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return TemplateFormCancelledMsg{} }
		case "tab":
			m.nextField()
			return m, nil
		case "shift+tab":
			m.prevField()
			return m, nil
		case "enter":
			if errMsg := m.validate(); errMsg != "" {
				m.err = errMsg
				return m, nil
			}
			m.err = ""
			tmpl := m.buildTemplateDef()
			return m, func() tea.Msg {
				return TemplateFormSubmittedMsg{Template: tmpl, IsEdit: m.isEdit}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the template form overlay.
func (m TemplateFormModel) View() string {
	var b strings.Builder

	title := "Add Template"
	if m.isEdit {
		title = "Edit Template"
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	for i := tmplFormField(0); i < tmplFieldCount; i++ {
		b.WriteString(formLabelStyle.Render(tmplFieldLabels[i]))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(formHintStyle.Render("tab: next field | shift+tab: prev | enter: save | esc: cancel"))

	formWidth := clamp(m.width-4, 40, 70)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
