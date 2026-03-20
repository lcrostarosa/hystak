package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/model"
)

func TestNewFormModelDefaults(t *testing.T) {
	m := NewFormModel()
	if m.transport != model.TransportStdio {
		t.Errorf("expected default transport stdio, got %s", m.transport)
	}
	if m.isEdit {
		t.Error("expected isEdit to be false for new form")
	}
	if m.focused != fieldName {
		t.Errorf("expected focus on fieldName, got %d", m.focused)
	}
}

func TestNewEditFormModelPopulates(t *testing.T) {
	srv := model.ServerDef{
		Name:        "github",
		Description: "GitHub server",
		Transport:   model.TransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-github"},
		Env:         map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}

	m := NewEditFormModel(srv)
	if !m.isEdit {
		t.Error("expected isEdit to be true")
	}
	if m.editName != "github" {
		t.Errorf("expected editName 'github', got %q", m.editName)
	}
	if m.transport != model.TransportStdio {
		t.Errorf("expected transport stdio, got %s", m.transport)
	}
	if v := m.inputs[fieldName].Value(); v != "github" {
		t.Errorf("expected name 'github', got %q", v)
	}
	if v := m.inputs[fieldDescription].Value(); v != "GitHub server" {
		t.Errorf("expected description, got %q", v)
	}
	if v := m.inputs[fieldCommand].Value(); v != "npx" {
		t.Errorf("expected command 'npx', got %q", v)
	}
	if v := m.inputs[fieldArgs].Value(); v != "-y, @modelcontextprotocol/server-github" {
		t.Errorf("expected args, got %q", v)
	}
}

func TestNewEditFormModelHTTP(t *testing.T) {
	srv := model.ServerDef{
		Name:      "qdrant",
		Transport: model.TransportHTTP,
		URL:       "http://localhost:6333/mcp",
		Headers:   map[string]string{"Authorization": "Bearer token"},
	}

	m := NewEditFormModel(srv)
	if m.transport != model.TransportHTTP {
		t.Errorf("expected transport http, got %s", m.transport)
	}
	if v := m.inputs[fieldURL].Value(); v != "http://localhost:6333/mcp" {
		t.Errorf("expected URL, got %q", v)
	}
	if v := m.inputs[fieldHeaders].Value(); !strings.Contains(v, "Authorization=Bearer token") {
		t.Errorf("expected headers, got %q", v)
	}
}

func TestVisibleFieldsStdio(t *testing.T) {
	m := NewFormModel()
	m.transport = model.TransportStdio
	visible := m.visibleFields()

	expected := []formField{fieldName, fieldDescription, fieldCommand, fieldArgs, fieldEnv}
	if len(visible) != len(expected) {
		t.Fatalf("expected %d visible fields, got %d", len(expected), len(visible))
	}
	for i, f := range expected {
		if visible[i] != f {
			t.Errorf("expected field %d at position %d, got %d", f, i, visible[i])
		}
	}
}

func TestVisibleFieldsHTTP(t *testing.T) {
	m := NewFormModel()
	m.transport = model.TransportHTTP
	visible := m.visibleFields()

	expected := []formField{fieldName, fieldDescription, fieldURL, fieldHeaders}
	if len(visible) != len(expected) {
		t.Fatalf("expected %d visible fields, got %d", len(expected), len(visible))
	}
	for i, f := range expected {
		if visible[i] != f {
			t.Errorf("expected field %d at position %d, got %d", f, i, visible[i])
		}
	}
}

func TestVisibleFieldsSSE(t *testing.T) {
	m := NewFormModel()
	m.transport = model.TransportSSE
	visible := m.visibleFields()

	expected := []formField{fieldName, fieldDescription, fieldURL, fieldHeaders}
	if len(visible) != len(expected) {
		t.Fatalf("expected %d visible fields, got %d", len(expected), len(visible))
	}
}

func TestCycleTransport(t *testing.T) {
	m := NewFormModel()
	if m.transport != model.TransportStdio {
		t.Fatal("expected initial transport stdio")
	}

	m.cycleTransport()
	if m.transport != model.TransportSSE {
		t.Errorf("expected sse after first cycle, got %s", m.transport)
	}

	m.cycleTransport()
	if m.transport != model.TransportHTTP {
		t.Errorf("expected http after second cycle, got %s", m.transport)
	}

	m.cycleTransport()
	if m.transport != model.TransportStdio {
		t.Errorf("expected stdio after third cycle, got %s", m.transport)
	}
}

func TestCycleTransportRefocuses(t *testing.T) {
	m := NewFormModel()
	m.focusField(fieldCommand) // stdio-specific field

	// Cycle to SSE — command is no longer visible
	m.cycleTransport()
	visible := m.visibleFields()
	found := false
	for _, f := range visible {
		if f == m.focused {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected focus to move to a visible field after transport cycle")
	}
}

func TestValidateEmptyName(t *testing.T) {
	m := NewFormModel()
	// Name is empty
	if err := m.validate(); err == "" {
		t.Error("expected validation error for empty name")
	} else if !strings.Contains(err, "Name is required") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestValidateStdioMissingCommand(t *testing.T) {
	m := NewFormModel()
	m.inputs[fieldName].SetValue("test-server")
	m.transport = model.TransportStdio
	// Command is empty

	if err := m.validate(); err == "" {
		t.Error("expected validation error for missing command")
	} else if !strings.Contains(err, "Command is required") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestValidateHTTPMissingURL(t *testing.T) {
	m := NewFormModel()
	m.inputs[fieldName].SetValue("test-server")
	m.transport = model.TransportHTTP
	// URL is empty

	if err := m.validate(); err == "" {
		t.Error("expected validation error for missing URL")
	} else if !strings.Contains(err, "URL is required") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestValidateStdioSuccess(t *testing.T) {
	m := NewFormModel()
	m.inputs[fieldName].SetValue("test-server")
	m.inputs[fieldCommand].SetValue("npx")
	m.transport = model.TransportStdio

	if err := m.validate(); err != "" {
		t.Errorf("expected no validation error, got: %s", err)
	}
}

func TestValidateHTTPSuccess(t *testing.T) {
	m := NewFormModel()
	m.inputs[fieldName].SetValue("test-server")
	m.inputs[fieldURL].SetValue("http://localhost:8080")
	m.transport = model.TransportHTTP

	if err := m.validate(); err != "" {
		t.Errorf("expected no validation error, got: %s", err)
	}
}

func TestBuildServerDefStdio(t *testing.T) {
	m := NewFormModel()
	m.transport = model.TransportStdio
	m.inputs[fieldName].SetValue("github")
	m.inputs[fieldDescription].SetValue("GitHub MCP")
	m.inputs[fieldCommand].SetValue("npx")
	m.inputs[fieldArgs].SetValue("-y, @modelcontextprotocol/server-github")
	m.inputs[fieldEnv].SetValue("GITHUB_TOKEN=${GITHUB_TOKEN}")

	srv := m.buildServerDef()
	if srv.Name != "github" {
		t.Errorf("expected name 'github', got %q", srv.Name)
	}
	if srv.Description != "GitHub MCP" {
		t.Errorf("expected description, got %q", srv.Description)
	}
	if srv.Transport != model.TransportStdio {
		t.Errorf("expected stdio, got %s", srv.Transport)
	}
	if srv.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", srv.Command)
	}
	if len(srv.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(srv.Args))
	}
	if srv.Args[0] != "-y" || srv.Args[1] != "@modelcontextprotocol/server-github" {
		t.Errorf("unexpected args: %v", srv.Args)
	}
	if srv.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("unexpected env: %v", srv.Env)
	}
}

func TestBuildServerDefHTTP(t *testing.T) {
	m := NewFormModel()
	m.transport = model.TransportHTTP
	m.inputs[fieldName].SetValue("qdrant")
	m.inputs[fieldURL].SetValue("http://localhost:6333/mcp")
	m.inputs[fieldHeaders].SetValue("Authorization=Bearer token")

	srv := m.buildServerDef()
	if srv.URL != "http://localhost:6333/mcp" {
		t.Errorf("expected URL, got %q", srv.URL)
	}
	if srv.Headers["Authorization"] != "Bearer token" {
		t.Errorf("unexpected headers: %v", srv.Headers)
	}
	if srv.Command != "" {
		t.Error("expected no command for HTTP server")
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"  ", nil},
		{"a, b, c", []string{"a", "b", "c"}},
		{"-y, @modelcontextprotocol/server-github", []string{"-y", "@modelcontextprotocol/server-github"}},
		{"single", []string{"single"}},
		{" , , ", nil},
	}

	for _, tt := range tests {
		result := parseCSV(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseCSV(%q): expected %d items, got %d: %v", tt.input, len(tt.expected), len(result), result)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("parseCSV(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], v)
			}
		}
	}
}

func TestParseKVPairs(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{"", nil},
		{"KEY=VALUE", map[string]string{"KEY": "VALUE"}},
		{"K1=V1, K2=V2", map[string]string{"K1": "V1", "K2": "V2"}},
		{"AUTH=Bearer token", map[string]string{"AUTH": "Bearer token"}},
		{"invalid", nil},
		{" , , ", nil},
	}

	for _, tt := range tests {
		result := parseKVPairs(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseKVPairs(%q): expected %d items, got %d: %v", tt.input, len(tt.expected), len(result), result)
			continue
		}
		for k, v := range tt.expected {
			if result[k] != v {
				t.Errorf("parseKVPairs(%q)[%q]: expected %q, got %q", tt.input, k, v, result[k])
			}
		}
	}
}

func TestFormEscCancels(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEscape}))
	if cmd == nil {
		t.Fatal("expected command from esc")
	}
	msg := cmd()
	if _, ok := msg.(FormCancelledMsg); !ok {
		t.Errorf("expected FormCancelledMsg, got %T", msg)
	}
}

func TestFormSubmitValid(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)
	m.inputs[fieldName].SetValue("test-server")
	m.inputs[fieldCommand].SetValue("cmd")

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected command from enter")
	}
	msg := cmd()
	submitted, ok := msg.(FormSubmittedMsg)
	if !ok {
		t.Fatalf("expected FormSubmittedMsg, got %T", msg)
	}
	if submitted.Server.Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %q", submitted.Server.Name)
	}
	if submitted.IsEdit {
		t.Error("expected IsEdit to be false")
	}
}

func TestFormSubmitInvalidShowsError(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)
	// Name is empty

	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd != nil {
		t.Error("expected no command for invalid submit")
	}
	if m.err == "" {
		t.Error("expected error message on invalid submit")
	}
}

func TestFormTabNavigation(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)

	if m.focused != fieldName {
		t.Fatal("expected initial focus on name")
	}

	// Tab to next field
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	if m.focused != fieldDescription {
		t.Errorf("expected focus on description after tab, got %d", m.focused)
	}

	// Shift+tab back
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyShiftTab}))
	if m.focused != fieldName {
		t.Errorf("expected focus on name after shift+tab, got %d", m.focused)
	}
}

func TestFormCtrlTCyclesTransport(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)

	if m.transport != model.TransportStdio {
		t.Fatal("expected initial transport stdio")
	}

	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlT}))
	if m.transport != model.TransportSSE {
		t.Errorf("expected sse after ctrl+t, got %s", m.transport)
	}
}

func TestFormViewContainsTitle(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "Add MCP") {
		t.Error("expected 'Add MCP' in form view")
	}
}

func TestFormViewEditTitle(t *testing.T) {
	srv := model.ServerDef{Name: "test", Transport: model.TransportStdio, Command: "cmd"}
	m := NewEditFormModel(srv)
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "Edit MCP") {
		t.Error("expected 'Edit MCP' in edit form view")
	}
}

func TestFormViewShowsTransportOptions(t *testing.T) {
	m := NewFormModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "stdio") {
		t.Error("expected 'stdio' in form view")
	}
	if !strings.Contains(view, "sse") {
		t.Error("expected 'sse' in form view")
	}
	if !strings.Contains(view, "http") {
		t.Error("expected 'http' in form view")
	}
}

func TestAppFormIntegration(t *testing.T) {
	svc := testService()
	app := NewApp(svc)

	// Simulate window size.
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = updated.(AppModel)

	// Simulate RequestFormMsg (as would be sent by servers tab pressing 'a').
	updated, _ = app.Update(RequestFormMsg{})
	app = updated.(AppModel)

	if app.mode != ModeForm {
		t.Errorf("expected ModeForm, got %d", app.mode)
	}

	// Cancel the form.
	updated, _ = app.Update(FormCancelledMsg{})
	app = updated.(AppModel)

	if app.mode != ModeBrowse {
		t.Errorf("expected ModeBrowse after cancel, got %d", app.mode)
	}
}

func TestAppFormSubmitAddsServer(t *testing.T) {
	svc := testService()
	app := NewApp(svc)

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = updated.(AppModel)

	// Open form.
	updated, _ = app.Update(RequestFormMsg{})
	app = updated.(AppModel)

	// Submit a new server.
	newServer := model.ServerDef{
		Name:      "new-server",
		Transport: model.TransportStdio,
		Command:   "new-cmd",
	}
	updated, _ = app.Update(FormSubmittedMsg{Server: newServer, IsEdit: false})
	app = updated.(AppModel)

	if app.mode != ModeBrowse {
		t.Errorf("expected ModeBrowse after submit, got %d", app.mode)
	}

	// Verify server was added to registry.
	if _, ok := svc.GetServer("new-server"); !ok {
		t.Error("expected new-server to be in registry")
	}

	// Verify list was refreshed.
	if len(app.mcps.list.Items()) != 3 {
		t.Errorf("expected 3 items after add, got %d", len(app.mcps.list.Items()))
	}
}

func TestAppFormSubmitEditsServer(t *testing.T) {
	svc := testService()
	app := NewApp(svc)

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = updated.(AppModel)

	// Open edit form for github.
	github, _ := svc.GetServer("github")
	updated, _ = app.Update(RequestFormMsg{EditServer: &github})
	app = updated.(AppModel)

	// Submit with updated command.
	editedServer := github
	editedServer.Command = "updated-cmd"
	app.form.editName = "github"
	updated, _ = app.Update(FormSubmittedMsg{Server: editedServer, IsEdit: true})
	app = updated.(AppModel)

	// Verify server was updated.
	srv, ok := svc.GetServer("github")
	if !ok {
		t.Fatal("expected github to still exist")
	}
	if srv.Command != "updated-cmd" {
		t.Errorf("expected updated command, got %q", srv.Command)
	}
}

func TestFieldLabel(t *testing.T) {
	labels := map[formField]string{
		fieldName:        "Name",
		fieldDescription: "Description",
		fieldCommand:     "Command",
		fieldArgs:        "Args (comma-separated)",
		fieldEnv:         "Environment (KEY=VALUE, ...)",
		fieldURL:         "URL",
		fieldHeaders:     "Headers (Key=Value, ...)",
	}
	for f, expected := range labels {
		if got := fieldLabel(f); got != expected {
			t.Errorf("fieldLabel(%d): expected %q, got %q", f, expected, got)
		}
	}
}
