package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// PromptFormSubmittedMsg is sent when the prompt form is submitted with a valid PromptDef.
type PromptFormSubmittedMsg struct {
	Prompt model.PromptDef
	IsEdit bool
}

// PromptFormCancelledMsg is sent when the prompt form is cancelled.
type PromptFormCancelledMsg struct{}

// promptFormField identifies a field in the prompt form.
type promptFormField int

const (
	promptFieldName promptFormField = iota
	promptFieldDescription
	promptFieldSource
	promptFieldCategory
	promptFieldOrder
	promptFieldTags
	promptFieldCount
)

// PromptFormModel is the overlay for adding/editing prompts.
type PromptFormModel struct {
	inputs   [promptFieldCount]textinput.Model
	focused  promptFormField
	isEdit   bool
	editName string
	err      string
	width    int
	height   int
}

// NewPromptFormModel creates a new form for adding a prompt.
func NewPromptFormModel() PromptFormModel {
	m := PromptFormModel{}
	m.initInputs()
	return m
}

// NewEditPromptFormModel creates a new form pre-populated for editing a prompt.
func NewEditPromptFormModel(prompt model.PromptDef) PromptFormModel {
	m := PromptFormModel{
		isEdit:   true,
		editName: prompt.Name,
	}
	m.initInputs()
	m.inputs[promptFieldName].SetValue(prompt.Name)
	m.inputs[promptFieldDescription].SetValue(prompt.Description)
	m.inputs[promptFieldSource].SetValue(prompt.Source)
	m.inputs[promptFieldCategory].SetValue(prompt.Category)
	if prompt.Order != 0 {
		m.inputs[promptFieldOrder].SetValue(strconv.Itoa(prompt.Order))
	}
	if len(prompt.Tags) > 0 {
		m.inputs[promptFieldTags].SetValue(strings.Join(prompt.Tags, ", "))
	}
	m.focusField(promptFieldName)
	return m
}

func (m *PromptFormModel) initInputs() {
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].CharLimit = 256
		m.inputs[i].Prompt = "  "
	}

	m.inputs[promptFieldName].Placeholder = "prompt-name"
	m.inputs[promptFieldDescription].Placeholder = "Brief description"
	m.inputs[promptFieldSource].Placeholder = "prompts/my-prompt.md"
	m.inputs[promptFieldCategory].Placeholder = "safety"
	m.inputs[promptFieldOrder].Placeholder = "10"
	m.inputs[promptFieldTags].Placeholder = "security, guardrail"

	m.focusField(promptFieldName)
}

// SetSize updates dimensions for the prompt form overlay.
func (m *PromptFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 60)
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}

func (m *PromptFormModel) focusField(f promptFormField) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focused = f
	m.inputs[f].Focus()
}

func (m *PromptFormModel) nextField() {
	if int(m.focused) < int(promptFieldCount)-1 {
		m.focusField(m.focused + 1)
	}
}

func (m *PromptFormModel) prevField() {
	if m.focused > 0 {
		m.focusField(m.focused - 1)
	}
}

var promptFieldLabels = [promptFieldCount]string{
	promptFieldName:        "Name",
	promptFieldDescription: "Description",
	promptFieldSource:      "Source",
	promptFieldCategory:    "Category",
	promptFieldOrder:       "Order",
	promptFieldTags:        "Tags",
}

// validate checks required fields and returns an error message or empty string.
func (m PromptFormModel) validate() string {
	if strings.TrimSpace(m.inputs[promptFieldName].Value()) == "" {
		return "Name is required"
	}
	if strings.TrimSpace(m.inputs[promptFieldSource].Value()) == "" {
		return "Source is required"
	}
	orderStr := strings.TrimSpace(m.inputs[promptFieldOrder].Value())
	if orderStr != "" {
		if _, err := strconv.Atoi(orderStr); err != nil {
			return "Order must be an integer"
		}
	}
	return ""
}

// buildPromptDef constructs a PromptDef from the form fields.
func (m PromptFormModel) buildPromptDef() model.PromptDef {
	order := 0
	orderStr := strings.TrimSpace(m.inputs[promptFieldOrder].Value())
	if orderStr != "" {
		order, _ = strconv.Atoi(orderStr) // already validated
	}

	var tags []string
	tagsStr := strings.TrimSpace(m.inputs[promptFieldTags].Value())
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	return model.PromptDef{
		Name:        strings.TrimSpace(m.inputs[promptFieldName].Value()),
		Description: strings.TrimSpace(m.inputs[promptFieldDescription].Value()),
		Source:      strings.TrimSpace(m.inputs[promptFieldSource].Value()),
		Category:    strings.TrimSpace(m.inputs[promptFieldCategory].Value()),
		Order:       order,
		Tags:        tags,
	}
}

// Update handles messages for the prompt form overlay.
func (m PromptFormModel) Update(msg tea.Msg) (PromptFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return PromptFormCancelledMsg{} }
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
			prompt := m.buildPromptDef()
			return m, func() tea.Msg {
				return PromptFormSubmittedMsg{Prompt: prompt, IsEdit: m.isEdit}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the prompt form overlay.
func (m PromptFormModel) View() string {
	var b strings.Builder

	title := "Add Prompt"
	if m.isEdit {
		title = "Edit Prompt"
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	for i := promptFormField(0); i < promptFieldCount; i++ {
		b.WriteString(formLabelStyle.Render(promptFieldLabels[i]))
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
