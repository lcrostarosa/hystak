package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// formField identifies a field in the server form.
type formField int

const (
	fieldName formField = iota
	fieldDescription
	fieldCommand
	fieldArgs
	fieldEnv
	fieldURL
	fieldHeaders
	fieldCount
)

// FormSubmittedMsg is sent when the form is submitted with a valid ServerDef.
type FormSubmittedMsg struct {
	Server model.ServerDef
	IsEdit bool
}

// FormCancelledMsg is sent when the form is cancelled.
type FormCancelledMsg struct{}

// RequestFormMsg is sent by the servers tab to request a form overlay.
type RequestFormMsg struct {
	EditServer *model.ServerDef // nil for add, non-nil for edit
}

// FormModel is the overlay for adding/editing MCP servers.
type FormModel struct {
	inputs    [fieldCount]textinput.Model
	transport model.Transport
	focused   formField
	isEdit    bool
	editName  string // original name when editing
	err       string
	width     int
	height    int
}

// NewFormModel creates a new form for adding a server.
func NewFormModel() FormModel {
	m := FormModel{
		transport: model.TransportStdio,
	}
	m.initInputs()
	return m
}

// NewEditFormModel creates a new form pre-populated for editing a server.
func NewEditFormModel(server model.ServerDef) FormModel {
	m := FormModel{
		transport: server.Transport,
		isEdit:    true,
		editName:  server.Name,
	}
	m.initInputs()

	m.inputs[fieldName].SetValue(server.Name)
	m.inputs[fieldDescription].SetValue(server.Description)
	m.inputs[fieldCommand].SetValue(server.Command)
	m.inputs[fieldArgs].SetValue(strings.Join(server.Args, ", "))
	m.inputs[fieldURL].SetValue(server.URL)

	if len(server.Env) > 0 {
		var envLines []string
		for _, k := range sortedKeys(server.Env) {
			envLines = append(envLines, fmt.Sprintf("%s=%s", k, server.Env[k]))
		}
		m.inputs[fieldEnv].SetValue(strings.Join(envLines, ", "))
	}

	if len(server.Headers) > 0 {
		var headerLines []string
		for _, k := range sortedKeys(server.Headers) {
			headerLines = append(headerLines, fmt.Sprintf("%s=%s", k, server.Headers[k]))
		}
		m.inputs[fieldHeaders].SetValue(strings.Join(headerLines, ", "))
	}

	m.focusField(fieldName)
	return m
}

func (m *FormModel) initInputs() {
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].CharLimit = 256
	}

	m.inputs[fieldName].Placeholder = "server-name"
	m.inputs[fieldName].Prompt = "  "
	m.inputs[fieldDescription].Placeholder = "optional description"
	m.inputs[fieldDescription].Prompt = "  "
	m.inputs[fieldCommand].Placeholder = "e.g. npx"
	m.inputs[fieldCommand].Prompt = "  "
	m.inputs[fieldArgs].Placeholder = "e.g. -y, @modelcontextprotocol/server-github"
	m.inputs[fieldArgs].Prompt = "  "
	m.inputs[fieldEnv].Placeholder = "KEY=VALUE, KEY2=VALUE2"
	m.inputs[fieldEnv].Prompt = "  "
	m.inputs[fieldEnv].CharLimit = 1024
	m.inputs[fieldURL].Placeholder = "e.g. http://localhost:8080/mcp"
	m.inputs[fieldURL].Prompt = "  "
	m.inputs[fieldHeaders].Placeholder = "Key=Value, Key2=Value2"
	m.inputs[fieldHeaders].Prompt = "  "
	m.inputs[fieldHeaders].CharLimit = 1024

	m.focusField(fieldName)
}

// SetSize updates dimensions for the form overlay.
func (m *FormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := clamp(w-10, 30, 60)
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}
}

// visibleFields returns the fields that should be shown for the current transport.
func (m FormModel) visibleFields() []formField {
	fields := []formField{fieldName, fieldDescription}
	switch m.transport {
	case model.TransportStdio:
		fields = append(fields, fieldCommand, fieldArgs, fieldEnv)
	case model.TransportSSE, model.TransportHTTP:
		fields = append(fields, fieldURL, fieldHeaders)
	}
	return fields
}

func (m *FormModel) focusField(f formField) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focused = f
	m.inputs[f].Focus()
}

func (m *FormModel) nextField() {
	visible := m.visibleFields()
	for i, f := range visible {
		if f == m.focused && i < len(visible)-1 {
			m.focusField(visible[i+1])
			return
		}
	}
}

func (m *FormModel) prevField() {
	visible := m.visibleFields()
	for i, f := range visible {
		if f == m.focused && i > 0 {
			m.focusField(visible[i-1])
			return
		}
	}
}

var transports = []model.Transport{
	model.TransportStdio, model.TransportSSE, model.TransportHTTP,
}

func (m *FormModel) cycleTransport() {
	for i, t := range transports {
		if t == m.transport {
			m.transport = transports[(i+1)%len(transports)]
			break
		}
	}
	// Ensure focused field is still visible.
	visible := m.visibleFields()
	for _, f := range visible {
		if f == m.focused {
			return
		}
	}
	// Focus first transport-specific field.
	if len(visible) > 2 {
		m.focusField(visible[2])
	}
}

// validate checks required fields and returns an error message or empty string.
func (m FormModel) validate() string {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" {
		return "Name is required"
	}

	switch m.transport {
	case model.TransportStdio:
		if strings.TrimSpace(m.inputs[fieldCommand].Value()) == "" {
			return "Command is required for stdio transport"
		}
	case model.TransportSSE, model.TransportHTTP:
		if strings.TrimSpace(m.inputs[fieldURL].Value()) == "" {
			return "URL is required for " + string(m.transport) + " transport"
		}
	}

	return ""
}

// buildServerDef constructs a ServerDef from the form fields.
func (m FormModel) buildServerDef() model.ServerDef {
	srv := model.ServerDef{
		Name:        strings.TrimSpace(m.inputs[fieldName].Value()),
		Description: strings.TrimSpace(m.inputs[fieldDescription].Value()),
		Transport:   m.transport,
	}

	switch m.transport {
	case model.TransportStdio:
		srv.Command = strings.TrimSpace(m.inputs[fieldCommand].Value())
		srv.Args = parseCSV(m.inputs[fieldArgs].Value())
		srv.Env = parseKVPairs(m.inputs[fieldEnv].Value())
	case model.TransportSSE, model.TransportHTTP:
		srv.URL = strings.TrimSpace(m.inputs[fieldURL].Value())
		srv.Headers = parseKVPairs(m.inputs[fieldHeaders].Value())
	}

	return srv
}

// Update handles messages for the form overlay.
func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return FormCancelledMsg{} }
		case "ctrl+t":
			m.cycleTransport()
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
			srv := m.buildServerDef()
			return m, func() tea.Msg {
				return FormSubmittedMsg{Server: srv, IsEdit: m.isEdit}
			}
		}
	}

	// Update the focused input.
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the form overlay.
func (m FormModel) View() string {
	var b strings.Builder

	title := "Add Server"
	if m.isEdit {
		title = "Edit Server"
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Transport selector
	b.WriteString(formLabelStyle.Render("Transport"))
	b.WriteString("  ")
	for _, t := range []model.Transport{model.TransportStdio, model.TransportSSE, model.TransportHTTP} {
		if t == m.transport {
			b.WriteString(formSelectedTransportStyle.Render(string(t)))
		} else {
			b.WriteString(formTransportStyle.Render(string(t)))
		}
		b.WriteString(" ")
	}
	b.WriteString(formHintStyle.Render("(ctrl+t to cycle)"))
	b.WriteString("\n\n")

	// Fields
	visible := m.visibleFields()
	for _, f := range visible {
		label := fieldLabel(f)
		b.WriteString(formLabelStyle.Render(label))
		b.WriteString("\n")
		b.WriteString(m.inputs[f].View())
		b.WriteString("\n\n")
	}

	// Error
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(formHintStyle.Render("tab: next field | shift+tab: prev | enter: save | esc: cancel"))

	formWidth := clamp(m.width-4, 40, 70)

	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

var fieldLabels = map[formField]string{
	fieldName:        "Name",
	fieldDescription: "Description",
	fieldCommand:     "Command",
	fieldArgs:        "Args (comma-separated)",
	fieldEnv:         "Environment (KEY=VALUE, ...)",
	fieldURL:         "URL",
	fieldHeaders:     "Headers (Key=Value, ...)",
}

func fieldLabel(f formField) string { return fieldLabels[f] }

// parseCSV splits a comma-separated string into trimmed, non-empty parts.
func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseKVPairs parses "KEY=VALUE, KEY2=VALUE2" into a map.
func parseKVPairs(s string) map[string]string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	result := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:idx])
		val := strings.TrimSpace(pair[idx+1:])
		if key != "" {
			result[key] = val
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
