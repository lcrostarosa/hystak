package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/backup"
	"github.com/lcrostarosa/hystak/internal/deploy"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
	"github.com/lcrostarosa/hystak/internal/service"
)

func testImportService(t *testing.T, importServers map[string]interface{}) (*service.Service, string) {
	t.Helper()

	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	_ = os.MkdirAll(configDir, 0o755)

	// Write a .mcp.json file with servers to import.
	projectDir := filepath.Join(dir, "project")
	_ = os.MkdirAll(projectDir, 0o755)
	mcpData := map[string]interface{}{"mcpServers": importServers}
	data, _ := json.Marshal(mcpData)
	_ = os.WriteFile(filepath.Join(projectDir, ".mcp.json"), data, 0o644)

	reg := &registry.Registry{
		Servers:     registry.NewStore[model.ServerDef, *model.ServerDef]("server"),
		Skills:      registry.NewStore[model.SkillDef, *model.SkillDef]("skill"),
		Hooks:       registry.NewStore[model.HookDef, *model.HookDef]("hook"),
		Permissions: registry.NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
		Templates:   registry.NewStore[model.TemplateDef, *model.TemplateDef]("template"),
		Prompts:     registry.NewStore[model.PromptDef, *model.PromptDef]("prompt"),
		Tags:        make(map[string][]string),
	}
	_ = reg.Servers.Add(model.ServerDef{
		Name:      "existing",
		Transport: model.TransportStdio,
		Command:   "existing-cmd",
	})

	store := &project.Store{
		Projects: make(map[string]model.Project),
	}

	deployer, _ := deploy.NewDeployer(model.ClientClaudeCode)

	svc := service.NewForTest(
		reg,
		store,
		map[model.ClientType]deploy.Deployer{model.ClientClaudeCode: deployer},
		backup.NewManager(filepath.Join(configDir, "backups")),
		configDir,
		nil,
	)

	return svc, projectDir
}

func TestNewImportModelDefaults(t *testing.T) {
	m := NewImportModel(nil)
	if m.phase != phaseInputPath {
		t.Errorf("expected initial phase phaseInputPath, got %d", m.phase)
	}
}

func TestImportEscCancels(t *testing.T) {
	m := NewImportModel(nil)
	m.SetSize(80, 24)

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEscape}))
	if cmd == nil {
		t.Fatal("expected command from esc")
	}
	msg := cmd()
	if _, ok := msg.(ImportCancelledMsg); !ok {
		t.Errorf("expected ImportCancelledMsg, got %T", msg)
	}
}

func TestImportEmptyPathError(t *testing.T) {
	svc := testService()
	m := NewImportModel(svc)
	m.SetSize(80, 24)

	// Press enter with empty path.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd != nil {
		t.Error("expected no command for empty path")
	}
	if m.err == "" {
		t.Error("expected error for empty path")
	}
	if !strings.Contains(m.err, "Path is required") {
		t.Errorf("unexpected error: %s", m.err)
	}
}

func TestImportInvalidPathError(t *testing.T) {
	svc := testService()
	m := NewImportModel(svc)
	m.SetSize(80, 24)

	m.pathInput.SetValue("/nonexistent/path/.mcp.json")
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd != nil {
		t.Error("expected no command for invalid path")
	}
	if m.err == "" {
		t.Error("expected error for invalid path")
	}
}

func TestImportLoadsServersToPreview(t *testing.T) {
	importServers := map[string]interface{}{
		"new-server": map[string]interface{}{
			"type":    "stdio",
			"command": "test-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)

	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	if m.phase != phasePreview {
		t.Fatalf("expected phasePreview, got %d", m.phase)
	}
	if len(m.candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(m.candidates))
	}
	if m.candidates[0].Name != "new-server" {
		t.Errorf("expected candidate name 'new-server', got %q", m.candidates[0].Name)
	}
	// All should be selected by default.
	if !m.selected[0] {
		t.Error("expected candidate to be selected by default")
	}
}

func TestImportPreviewToggleSelection(t *testing.T) {
	importServers := map[string]interface{}{
		"srv1": map[string]interface{}{"type": "stdio", "command": "cmd1"},
		"srv2": map[string]interface{}{"type": "stdio", "command": "cmd2"},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	if m.phase != phasePreview {
		t.Fatal("expected phasePreview")
	}

	// Toggle first server off.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{' '}}))
	if m.selected[0] {
		t.Error("expected first candidate to be deselected")
	}
	if !m.selected[1] {
		t.Error("expected second candidate to still be selected")
	}
}

func TestImportPreviewNavigation(t *testing.T) {
	importServers := map[string]interface{}{
		"srv1": map[string]interface{}{"type": "stdio", "command": "cmd1"},
		"srv2": map[string]interface{}{"type": "stdio", "command": "cmd2"},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	// Move down.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}

	// Move up.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyUp}))
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
}

func TestImportNoConflictImportsDirectly(t *testing.T) {
	importServers := map[string]interface{}{
		"new-server": map[string]interface{}{
			"type":    "stdio",
			"command": "new-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)

	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	if m.phase != phasePreview {
		t.Fatal("expected phasePreview")
	}

	// Press enter to confirm import.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected command from import")
	}
	msg := cmd()
	completed, ok := msg.(ImportCompletedMsg)
	if !ok {
		t.Fatalf("expected ImportCompletedMsg, got %T", msg)
	}
	if completed.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", completed.Imported)
	}

	// Verify server was added to registry.
	if _, ok := svc.GetServer("new-server"); !ok {
		t.Error("expected new-server to be in registry")
	}
}

func TestImportConflictEntersConflictPhase(t *testing.T) {
	// Import a server with the same name as an existing one.
	importServers := map[string]interface{}{
		"existing": map[string]interface{}{
			"type":    "stdio",
			"command": "new-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)

	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	if m.phase != phasePreview {
		t.Fatal("expected phasePreview")
	}

	// Confirm selection — should go to conflict phase.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	if m.phase != phaseConflict {
		t.Fatalf("expected phaseConflict, got %d", m.phase)
	}
	if m.conflictAt != 0 {
		t.Errorf("expected conflictAt 0, got %d", m.conflictAt)
	}
}

func TestImportConflictSkip(t *testing.T) {
	importServers := map[string]interface{}{
		"existing": map[string]interface{}{
			"type":    "stdio",
			"command": "new-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter})) // to conflict

	// Press 's' to skip.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'s'}}))
	if cmd == nil {
		t.Fatal("expected command after resolving last conflict")
	}
	msg := cmd()
	completed, ok := msg.(ImportCompletedMsg)
	if !ok {
		t.Fatalf("expected ImportCompletedMsg, got %T", msg)
	}
	if completed.Imported != 0 {
		t.Errorf("expected 0 imported (skipped), got %d", completed.Imported)
	}

	// Verify existing server is unchanged.
	srv, _ := svc.GetServer("existing")
	if srv.Command != "existing-cmd" {
		t.Errorf("expected existing command unchanged, got %q", srv.Command)
	}
}

func TestImportConflictReplace(t *testing.T) {
	importServers := map[string]interface{}{
		"existing": map[string]interface{}{
			"type":    "stdio",
			"command": "new-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter})) // to conflict

	// Press 'r' to replace.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'r'}}))
	if cmd == nil {
		t.Fatal("expected command after resolving last conflict")
	}
	msg := cmd()
	completed, ok := msg.(ImportCompletedMsg)
	if !ok {
		t.Fatalf("expected ImportCompletedMsg, got %T", msg)
	}
	if completed.Imported != 1 {
		t.Errorf("expected 1 imported (replaced), got %d", completed.Imported)
	}

	// Verify existing server was replaced.
	srv, _ := svc.GetServer("existing")
	if srv.Command != "new-cmd" {
		t.Errorf("expected replaced command 'new-cmd', got %q", srv.Command)
	}
}

func TestImportConflictRename(t *testing.T) {
	importServers := map[string]interface{}{
		"existing": map[string]interface{}{
			"type":    "stdio",
			"command": "imported-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter})) // to conflict

	// Press 'n' to rename.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'n'}}))
	if m.cursor != 3 {
		t.Fatalf("expected cursor at 3 (rename), got %d", m.cursor)
	}

	// Type the new name.
	m.renameInput.SetValue("existing-imported")

	// Press enter to confirm rename.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected command after rename")
	}
	msg := cmd()
	completed, ok := msg.(ImportCompletedMsg)
	if !ok {
		t.Fatalf("expected ImportCompletedMsg, got %T", msg)
	}
	if completed.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", completed.Imported)
	}

	// Verify original server unchanged and renamed exists.
	orig, _ := svc.GetServer("existing")
	if orig.Command != "existing-cmd" {
		t.Errorf("expected original unchanged, got %q", orig.Command)
	}
	renamed, ok := svc.GetServer("existing-imported")
	if !ok {
		t.Error("expected renamed server to exist in registry")
	}
	if renamed.Command != "imported-cmd" {
		t.Errorf("expected renamed command 'imported-cmd', got %q", renamed.Command)
	}
}

func TestImportConflictRenameEmptyNameError(t *testing.T) {
	importServers := map[string]interface{}{
		"existing": map[string]interface{}{
			"type":    "stdio",
			"command": "cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter})) // to conflict

	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'n'}}))
	// Leave rename empty and press enter.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd != nil {
		t.Error("expected no command for empty rename")
	}
	if m.err == "" {
		t.Error("expected error for empty rename")
	}
}

func TestImportNoneSelectedError(t *testing.T) {
	importServers := map[string]interface{}{
		"srv": map[string]interface{}{"type": "stdio", "command": "cmd"},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Deselect the only server.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{' '}}))

	// Try to confirm.
	m, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd != nil {
		t.Error("expected no command when no servers selected")
	}
	if m.err == "" {
		t.Error("expected error when no servers selected")
	}
}

func TestImportViewInputPath(t *testing.T) {
	m := NewImportModel(nil)
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "Import MCPs") {
		t.Error("expected 'Import MCPs' in view")
	}
	if !strings.Contains(view, "Config file path") {
		t.Error("expected 'Config file path' label in view")
	}
}

func TestImportViewPreview(t *testing.T) {
	m := NewImportModel(nil)
	m.SetSize(80, 24)
	m.phase = phasePreview
	m.candidates = []service.ImportCandidate{
		{Name: "test-srv", Server: model.ServerDef{Transport: model.TransportStdio, Command: "cmd"}, Conflict: false},
	}
	m.selected = []bool{true}

	view := m.View()
	if !strings.Contains(view, "Select MCPs to Import") {
		t.Error("expected 'Select MCPs to Import' in preview view")
	}
	if !strings.Contains(view, "test-srv") {
		t.Error("expected server name in preview view")
	}
	if !strings.Contains(view, "[x]") {
		t.Error("expected checkbox in preview view")
	}
}

func TestImportViewConflict(t *testing.T) {
	svc := testService()
	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.phase = phaseConflict
	m.candidates = []service.ImportCandidate{
		{Name: "github", Server: model.ServerDef{Transport: model.TransportStdio, Command: "new-cmd"}, Conflict: true},
	}
	m.conflictAt = 0

	view := m.View()
	if !strings.Contains(view, "Resolve Conflict") {
		t.Error("expected 'Resolve Conflict' in conflict view")
	}
	if !strings.Contains(view, "github") {
		t.Error("expected server name in conflict view")
	}
	if !strings.Contains(view, "Keep existing") {
		t.Error("expected 'Keep existing' option")
	}
	if !strings.Contains(view, "Replace") {
		t.Error("expected 'Replace' option")
	}
	if !strings.Contains(view, "Rename") {
		t.Error("expected 'Rename' option")
	}
}

func TestAppImportIntegration(t *testing.T) {
	svc := testService()
	app := NewApp(svc)

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = updated.(AppModel)

	// Open import overlay.
	updated, _ = app.Update(RequestImportMsg{})
	app = updated.(AppModel)

	if app.mode != ModeImport {
		t.Errorf("expected ModeImport, got %d", app.mode)
	}

	// Cancel.
	updated, _ = app.Update(ImportCancelledMsg{})
	app = updated.(AppModel)

	if app.mode != ModeBrowse {
		t.Errorf("expected ModeBrowse after cancel, got %d", app.mode)
	}
}

func TestAppImportCompleted(t *testing.T) {
	svc := testService()
	app := NewApp(svc)

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = updated.(AppModel)

	updated, _ = app.Update(RequestImportMsg{})
	app = updated.(AppModel)

	// Simulate completed import.
	updated, _ = app.Update(ImportCompletedMsg{Imported: 2})
	app = updated.(AppModel)

	if app.mode != ModeBrowse {
		t.Errorf("expected ModeBrowse after import complete, got %d", app.mode)
	}
}

func TestMCPsTabImportKey(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'i'}}))
	if cmd == nil {
		t.Fatal("expected command from 'i' key")
	}
	msg := cmd()
	if _, ok := msg.(RequestImportMsg); !ok {
		t.Errorf("expected RequestImportMsg, got %T", msg)
	}
}

func TestFormatServerCompact(t *testing.T) {
	tests := []struct {
		srv      model.ServerDef
		contains string
	}{
		{model.ServerDef{Transport: model.TransportStdio, Command: "npx", Args: []string{"-y", "foo"}}, "stdio: npx -y foo"},
		{model.ServerDef{Transport: model.TransportStdio, Command: "npx"}, "stdio: npx"},
		{model.ServerDef{Transport: model.TransportHTTP, URL: "http://example.com"}, "http: http://example.com"},
	}
	for _, tt := range tests {
		got := formatServerCompact(tt.srv)
		if got != tt.contains {
			t.Errorf("formatServerCompact: expected %q, got %q", tt.contains, got)
		}
	}
}

func TestImportPreviewEscCancels(t *testing.T) {
	m := NewImportModel(nil)
	m.SetSize(80, 24)
	m.phase = phasePreview
	m.candidates = []service.ImportCandidate{{Name: "srv"}}
	m.selected = []bool{true}

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEscape}))
	if cmd == nil {
		t.Fatal("expected command from esc")
	}
	msg := cmd()
	if _, ok := msg.(ImportCancelledMsg); !ok {
		t.Errorf("expected ImportCancelledMsg, got %T", msg)
	}
}

func TestImportConflictEscCancels(t *testing.T) {
	m := NewImportModel(nil)
	m.SetSize(80, 24)
	m.phase = phaseConflict
	m.candidates = []service.ImportCandidate{{Name: "srv", Conflict: true}}
	m.conflictAt = 0

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEscape}))
	if cmd == nil {
		t.Fatal("expected command from esc")
	}
	msg := cmd()
	if _, ok := msg.(ImportCancelledMsg); !ok {
		t.Errorf("expected ImportCancelledMsg, got %T", msg)
	}
}

func TestImportConflictKeepViaCursorEnter(t *testing.T) {
	importServers := map[string]interface{}{
		"existing": map[string]interface{}{
			"type":    "stdio",
			"command": "new-cmd",
		},
	}
	svc, projectDir := testImportService(t, importServers)

	m := NewImportModel(svc)
	m.SetSize(80, 24)
	m.pathInput.SetValue(filepath.Join(projectDir, ".mcp.json"))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter})) // to conflict

	// Cursor starts at 0 (Keep existing), press enter.
	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}
	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected command")
	}
	msg := cmd()
	completed, ok := msg.(ImportCompletedMsg)
	if !ok {
		t.Fatalf("expected ImportCompletedMsg, got %T", msg)
	}
	if completed.Imported != 0 {
		t.Errorf("expected 0 imported (kept), got %d", completed.Imported)
	}
}
