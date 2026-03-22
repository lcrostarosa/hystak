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

// testDiffService creates a service with a project that has drifted servers.
func testDiffService(t *testing.T, registryServers map[string]model.ServerDef, deployedServers map[string]interface{}) (*service.Service, string) {
	t.Helper()

	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	_ = os.MkdirAll(configDir, 0o755)

	projectDir := filepath.Join(dir, "project")
	_ = os.MkdirAll(projectDir, 0o755)

	// Write deployed .mcp.json.
	mcpData := map[string]interface{}{"mcpServers": deployedServers}
	data, _ := json.Marshal(mcpData)
	_ = os.WriteFile(filepath.Join(projectDir, ".mcp.json"), data, 0o644)

	reg := &registry.Registry{
		Servers: registryServers,
		Tags:    make(map[string][]string),
	}

	store := &project.Store{
		Projects: map[string]model.Project{
			"test-project": {
				Name:    "test-project",
				Path:    projectDir,
				Clients: []model.ClientType{model.ClientClaudeCode},
				MCPs:    []model.MCPAssignment{{Name: "my-server"}},
			},
		},
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

func TestNewDiffModelNoDrift(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "my-cmd",
		},
	}
	deployedServers := map[string]interface{}{
		"my-server": map[string]interface{}{
			"type":    "stdio",
			"command": "my-cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, deployedServers)
	m := NewDiffModel(svc, "test-project")

	if m.rawDiff != "" {
		t.Errorf("expected empty diff for synced project, got: %s", m.rawDiff)
	}
	if m.err != "" {
		t.Errorf("unexpected error: %s", m.err)
	}
}

func TestNewDiffModelWithDrift(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "new-cmd",
		},
	}
	deployedServers := map[string]interface{}{
		"my-server": map[string]interface{}{
			"type":    "stdio",
			"command": "old-cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, deployedServers)
	m := NewDiffModel(svc, "test-project")

	if m.rawDiff == "" {
		t.Fatal("expected non-empty diff for drifted project")
	}
	if !strings.Contains(m.rawDiff, "old-cmd") || !strings.Contains(m.rawDiff, "new-cmd") {
		t.Errorf("diff should contain old and new commands, got: %s", m.rawDiff)
	}
}

func TestDiffEscCloses(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, map[string]interface{}{})
	m := NewDiffModel(svc, "test-project")
	m.SetSize(80, 24)

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEscape}))
	if cmd == nil {
		t.Fatal("expected command from esc")
	}
	msg := cmd()
	if _, ok := msg.(DiffClosedMsg); !ok {
		t.Errorf("expected DiffClosedMsg, got %T", msg)
	}
}

func TestDiffQCloses(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, map[string]interface{}{})
	m := NewDiffModel(svc, "test-project")
	m.SetSize(80, 24)

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}}))
	if cmd == nil {
		t.Fatal("expected command from q")
	}
	msg := cmd()
	if _, ok := msg.(DiffClosedMsg); !ok {
		t.Errorf("expected DiffClosedMsg, got %T", msg)
	}
}

func TestDiffSyncResolvesAndRefreshes(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "expected-cmd",
		},
	}
	deployedServers := map[string]interface{}{
		"my-server": map[string]interface{}{
			"type":    "stdio",
			"command": "old-cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, deployedServers)
	m := NewDiffModel(svc, "test-project")
	m.SetSize(80, 24)

	// Verify drift exists before sync.
	if m.rawDiff == "" {
		t.Fatal("expected drift before sync")
	}

	// Press 's' to sync.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'s'}}))

	if !m.synced {
		t.Error("expected synced flag to be true after sync")
	}
	if m.rawDiff != "" {
		t.Errorf("expected empty diff after sync, got: %s", m.rawDiff)
	}
	if m.err != "" {
		t.Errorf("unexpected error after sync: %s", m.err)
	}
}

func TestDiffViewContainsProjectName(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, map[string]interface{}{})
	m := NewDiffModel(svc, "test-project")
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "test-project") {
		t.Error("view should contain the project name")
	}
}

func TestDiffViewportScrolling(t *testing.T) {
	regServers := map[string]model.ServerDef{
		"my-server": {
			Name:      "my-server",
			Transport: model.TransportStdio,
			Command:   "new-cmd",
			Args:      []string{"a", "b", "c"},
			Env:       map[string]string{"A": "1", "B": "2", "C": "3"},
		},
	}
	deployedServers := map[string]interface{}{
		"my-server": map[string]interface{}{
			"type":    "stdio",
			"command": "old-cmd",
		},
	}

	svc, _ := testDiffService(t, regServers, deployedServers)
	m := NewDiffModel(svc, "test-project")
	m.SetSize(80, 10) // small viewport to force scrolling

	// Should not panic when scrolling.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyUp}))

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view after scrolling")
	}
}

func TestDiffInvalidProject(t *testing.T) {
	regServers := map[string]model.ServerDef{}

	svc, _ := testDiffService(t, regServers, map[string]interface{}{})
	m := NewDiffModel(svc, "nonexistent-project")

	if m.err == "" {
		t.Error("expected error for nonexistent project")
	}
}

func TestColorizeDiff(t *testing.T) {
	input := "--- a\n+++ b\n@@ -1,2 +1,2 @@\n-old\n+new\n context"
	result := colorizeDiff(input)

	// Should contain styled versions of all line types.
	if !strings.Contains(result, "old") {
		t.Error("result should contain 'old'")
	}
	if !strings.Contains(result, "new") {
		t.Error("result should contain 'new'")
	}
	if !strings.Contains(result, "context") {
		t.Error("result should contain 'context'")
	}
}
