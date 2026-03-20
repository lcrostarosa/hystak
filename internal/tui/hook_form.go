package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// RequestHookFormMsg is sent to open the hook add/edit overlay.
type RequestHookFormMsg struct {
	EditHook *model.HookDef // nil for add, non-nil for edit
}

// HookFormSubmittedMsg is sent when the hook form is submitted with a valid HookDef.
type HookFormSubmittedMsg struct {
	Hook   model.HookDef
	IsEdit bool
}

// HookFormCancelledMsg is sent when the hook form is cancelled.
type HookFormCancelledMsg struct{}

// hookFormField identifies a field in the hook form.
type hookFormField int

const (
	hookFieldName hookFormField = iota
	hookFieldEvent
	hookFieldMatcher
	hookFieldCommand
	hookFieldTimeout
	hookFieldCount
)

// HookFormModel is the overlay for adding/editing hooks.
type HookFormModel struct {
	inputs   [hookFieldCount]textinput.Model
	focused  hookFormField
	isEdit   bool
	editName string
	err      string
	width    int
	height   int
}

// NewHookFormModel creates a new form for adding a hook.
func NewHookFormModel() HookFormModel {
	m := HookFormModel{}
	m.initInputs()
	return m
}

// NewEditHookFormModel creates a new form pre-populated for editing a hook.
func NewEditHookFormModel(hook model.HookDef) HookFormModel {
	m := HookFormModel{
		isEdit:   true,
		editName: hook.Name,
	}
	m.initInputs()
	m.inputs[hookFieldName].SetValue(hook.Name)
	m.inputs[hookFieldEvent].SetValue(hook.Event)
	m.inputs[hookFieldMatcher].SetValue(hook.Matcher)
	m.inputs[hookFieldCommand].SetValue(hook.Command)
	if hook.Timeout != 0 {
		m.inputs[hookFieldTimeout].SetValue(strconv.Itoa(hook.Timeout))
	}
	m.focusField(hookFieldName)
	return m
}

func (m *HookFormModel) initInputs() {
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].CharLimit = 256
		m.inputs[i].Prompt = "  "
	}

	m.inputs[hookFieldName].Placeholder = "hook-name"
	m.inputs[hookFieldEvent].Placeholder = "PreToolUse"
	m.inputs[hookFieldMatcher].Placeholder = "Bash"
	m.inputs[hookFieldCommand].Placeholder = "bash -c 'echo hello'"
	m.inputs[hookFieldTimeout].Placeholder = "10000"

	m.focusField(hookFieldName)
}

// SetSize updates dimensions for the hook form overlay.
func (m *HookFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 60)
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}

func (m *HookFormModel) focusField(f hookFormField) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focused = f
	m.inputs[f].Focus()
}

func (m *HookFormModel) nextField() {
	if int(m.focused) < int(hookFieldCount)-1 {
		m.focusField(m.focused + 1)
	}
}

func (m *HookFormModel) prevField() {
	if m.focused > 0 {
		m.focusField(m.focused - 1)
	}
}

var hookFieldLabels = [hookFieldCount]string{
	hookFieldName:    "Name",
	hookFieldEvent:   "Event",
	hookFieldMatcher: "Matcher (optional)",
	hookFieldCommand: "Command",
	hookFieldTimeout: "Timeout ms (optional)",
}

// validate checks required fields and returns an error message or empty string.
func (m HookFormModel) validate() string {
	if strings.TrimSpace(m.inputs[hookFieldName].Value()) == "" {
		return "Name is required"
	}
	if strings.TrimSpace(m.inputs[hookFieldEvent].Value()) == "" {
		return "Event is required"
	}
	if strings.TrimSpace(m.inputs[hookFieldCommand].Value()) == "" {
		return "Command is required"
	}
	if v := strings.TrimSpace(m.inputs[hookFieldTimeout].Value()); v != "" {
		if _, err := strconv.Atoi(v); err != nil {
			return "Timeout must be a valid integer"
		}
	}
	return ""
}

// buildHookDef constructs a HookDef from the form fields.
func (m HookFormModel) buildHookDef() model.HookDef {
	timeout := 0
	if v := strings.TrimSpace(m.inputs[hookFieldTimeout].Value()); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			timeout = n
		}
	}
	return model.HookDef{
		Name:    strings.TrimSpace(m.inputs[hookFieldName].Value()),
		Event:   strings.TrimSpace(m.inputs[hookFieldEvent].Value()),
		Matcher: strings.TrimSpace(m.inputs[hookFieldMatcher].Value()),
		Command: strings.TrimSpace(m.inputs[hookFieldCommand].Value()),
		Timeout: timeout,
	}
}

// Update handles messages for the hook form overlay.
func (m HookFormModel) Update(msg tea.Msg) (HookFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return HookFormCancelledMsg{} }
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
			hook := m.buildHookDef()
			return m, func() tea.Msg {
				return HookFormSubmittedMsg{Hook: hook, IsEdit: m.isEdit}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the hook form overlay.
func (m HookFormModel) View() string {
	var b strings.Builder

	title := "Add Hook"
	if m.isEdit {
		title = "Edit Hook"
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	for i := hookFormField(0); i < hookFieldCount; i++ {
		b.WriteString(formLabelStyle.Render(hookFieldLabels[i]))
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
