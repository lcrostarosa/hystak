package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// RequestSkillFormMsg is sent to open the skill add/edit overlay.
type RequestSkillFormMsg struct {
	EditSkill *model.SkillDef // nil for add, non-nil for edit
}

// SkillFormSubmittedMsg is sent when the skill form is submitted with a valid SkillDef.
type SkillFormSubmittedMsg struct {
	Skill  model.SkillDef
	IsEdit bool
}

// SkillFormCancelledMsg is sent when the skill form is cancelled.
type SkillFormCancelledMsg struct{}

// skillFormField identifies a field in the skill form.
type skillFormField int

const (
	skillFieldName skillFormField = iota
	skillFieldDescription
	skillFieldSource
	skillFieldCount
)

// SkillFormModel is the overlay for adding/editing skills.
type SkillFormModel struct {
	inputs   [skillFieldCount]textinput.Model
	focused  skillFormField
	isEdit   bool
	editName string
	err      string
	width    int
	height   int
}

// NewSkillFormModel creates a new form for adding a skill.
func NewSkillFormModel() SkillFormModel {
	m := SkillFormModel{}
	m.initInputs()
	return m
}

// NewEditSkillFormModel creates a new form pre-populated for editing a skill.
func NewEditSkillFormModel(skill model.SkillDef) SkillFormModel {
	m := SkillFormModel{
		isEdit:   true,
		editName: skill.Name,
	}
	m.initInputs()
	m.inputs[skillFieldName].SetValue(skill.Name)
	m.inputs[skillFieldDescription].SetValue(skill.Description)
	m.inputs[skillFieldSource].SetValue(skill.Source)
	m.focusField(skillFieldName)
	return m
}

func (m *SkillFormModel) initInputs() {
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].CharLimit = 256
		m.inputs[i].Prompt = "  "
	}

	m.inputs[skillFieldName].Placeholder = "skill-name"
	m.inputs[skillFieldDescription].Placeholder = "optional description"
	m.inputs[skillFieldSource].Placeholder = "path/to/skill.md"

	m.focusField(skillFieldName)
}

// SetSize updates dimensions for the skill form overlay.
func (m *SkillFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 60)
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}

func (m *SkillFormModel) focusField(f skillFormField) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focused = f
	m.inputs[f].Focus()
}

func (m *SkillFormModel) nextField() {
	if int(m.focused) < int(skillFieldCount)-1 {
		m.focusField(m.focused + 1)
	}
}

func (m *SkillFormModel) prevField() {
	if m.focused > 0 {
		m.focusField(m.focused - 1)
	}
}

var skillFieldLabels = [skillFieldCount]string{
	skillFieldName:        "Name",
	skillFieldDescription: "Description (optional)",
	skillFieldSource:      "Source",
}

// validate checks required fields and returns an error message or empty string.
func (m SkillFormModel) validate() string {
	if strings.TrimSpace(m.inputs[skillFieldName].Value()) == "" {
		return "Name is required"
	}
	if strings.TrimSpace(m.inputs[skillFieldSource].Value()) == "" {
		return "Source is required"
	}
	return ""
}

// buildSkillDef constructs a SkillDef from the form fields.
func (m SkillFormModel) buildSkillDef() model.SkillDef {
	return model.SkillDef{
		Name:        strings.TrimSpace(m.inputs[skillFieldName].Value()),
		Description: strings.TrimSpace(m.inputs[skillFieldDescription].Value()),
		Source:      strings.TrimSpace(m.inputs[skillFieldSource].Value()),
	}
}

// Update handles messages for the skill form overlay.
func (m SkillFormModel) Update(msg tea.Msg) (SkillFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return SkillFormCancelledMsg{} }
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
			skill := m.buildSkillDef()
			return m, func() tea.Msg {
				return SkillFormSubmittedMsg{Skill: skill, IsEdit: m.isEdit}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the skill form overlay.
func (m SkillFormModel) View() string {
	var b strings.Builder

	title := "Add Skill"
	if m.isEdit {
		title = "Edit Skill"
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	for i := skillFormField(0); i < skillFieldCount; i++ {
		b.WriteString(formLabelStyle.Render(skillFieldLabels[i]))
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
