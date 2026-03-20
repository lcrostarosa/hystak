package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// RequestPermissionFormMsg is sent to open the permission add/edit overlay.
type RequestPermissionFormMsg struct {
	EditPermission *model.PermissionRule // nil for add, non-nil for edit
}

// PermissionFormSubmittedMsg is sent when the permission form is submitted with a valid PermissionRule.
type PermissionFormSubmittedMsg struct {
	Permission model.PermissionRule
	IsEdit     bool
}

// PermissionFormCancelledMsg is sent when the permission form is cancelled.
type PermissionFormCancelledMsg struct{}

// permFormField identifies a field in the permission form.
type permFormField int

const (
	permFieldName permFormField = iota
	permFieldRule
	permFieldCount
)

// PermissionFormModel is the overlay for adding/editing permissions.
type PermissionFormModel struct {
	inputs   [permFieldCount]textinput.Model
	permType string // "allow" or "deny"
	focused  permFormField
	isEdit   bool
	editName string
	err      string
	width    int
	height   int
}

// NewPermissionFormModel creates a new form for adding a permission.
func NewPermissionFormModel() PermissionFormModel {
	m := PermissionFormModel{
		permType: "allow",
	}
	m.initInputs()
	return m
}

// NewEditPermissionFormModel creates a new form pre-populated for editing a permission.
func NewEditPermissionFormModel(perm model.PermissionRule) PermissionFormModel {
	m := PermissionFormModel{
		permType: perm.EffectiveType(),
		isEdit:   true,
		editName: perm.Name,
	}
	m.initInputs()
	m.inputs[permFieldName].SetValue(perm.Name)
	m.inputs[permFieldRule].SetValue(perm.Rule)
	m.focusField(permFieldName)
	return m
}

func (m *PermissionFormModel) initInputs() {
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].CharLimit = 256
		m.inputs[i].Prompt = "  "
	}

	m.inputs[permFieldName].Placeholder = "permission-name"
	m.inputs[permFieldRule].Placeholder = "Bash(*)"

	m.focusField(permFieldName)
}

// SetSize updates dimensions for the permission form overlay.
func (m *PermissionFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 60)
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}

func (m *PermissionFormModel) focusField(f permFormField) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focused = f
	m.inputs[f].Focus()
}

func (m *PermissionFormModel) nextField() {
	if int(m.focused) < int(permFieldCount)-1 {
		m.focusField(m.focused + 1)
	}
}

func (m *PermissionFormModel) prevField() {
	if m.focused > 0 {
		m.focusField(m.focused - 1)
	}
}

func (m *PermissionFormModel) cycleType() {
	if m.permType == "allow" {
		m.permType = "deny"
	} else {
		m.permType = "allow"
	}
}

var permFieldLabels = [permFieldCount]string{
	permFieldName: "Name",
	permFieldRule: "Rule",
}

// validate checks required fields and returns an error message or empty string.
func (m PermissionFormModel) validate() string {
	if strings.TrimSpace(m.inputs[permFieldName].Value()) == "" {
		return "Name is required"
	}
	if strings.TrimSpace(m.inputs[permFieldRule].Value()) == "" {
		return "Rule is required"
	}
	return ""
}

// buildPermissionRule constructs a PermissionRule from the form fields.
func (m PermissionFormModel) buildPermissionRule() model.PermissionRule {
	return model.PermissionRule{
		Name: strings.TrimSpace(m.inputs[permFieldName].Value()),
		Rule: strings.TrimSpace(m.inputs[permFieldRule].Value()),
		Type: m.permType,
	}
}

// Update handles messages for the permission form overlay.
func (m PermissionFormModel) Update(msg tea.Msg) (PermissionFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return PermissionFormCancelledMsg{} }
		case "ctrl+t":
			m.cycleType()
			return m, nil
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
			perm := m.buildPermissionRule()
			return m, func() tea.Msg {
				return PermissionFormSubmittedMsg{Permission: perm, IsEdit: m.isEdit}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the permission form overlay.
func (m PermissionFormModel) View() string {
	var b strings.Builder

	title := "Add Permission"
	if m.isEdit {
		title = "Edit Permission"
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	for i := permFormField(0); i < permFieldCount; i++ {
		b.WriteString(formLabelStyle.Render(permFieldLabels[i]))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	// Type selector
	b.WriteString(formLabelStyle.Render("Type"))
	b.WriteString("  ")
	for _, t := range []string{"allow", "deny"} {
		if t == m.permType {
			b.WriteString(formSelectedTransportStyle.Render(t))
		} else {
			b.WriteString(formTransportStyle.Render(t))
		}
		b.WriteString(" ")
	}
	b.WriteString(formHintStyle.Render("(ctrl+t to toggle)"))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(formHintStyle.Render("tab: next field | shift+tab: prev | enter: save | esc: cancel"))

	formWidth := clamp(m.width-4, 40, 70)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
