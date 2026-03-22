package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lcrostarosa/hystak/internal/backup"
	"github.com/lcrostarosa/hystak/internal/deploy"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
)

// mockDeployer implements deploy.Deployer with in-memory storage.
type mockDeployer struct {
	ct      model.ClientType
	servers map[string]map[string]model.ServerDef // projectPath -> name -> server
}

// Compile-time check that mockDeployer satisfies deploy.Deployer.
var _ deploy.Deployer = (*mockDeployer)(nil)

func newMockDeployer(ct model.ClientType) *mockDeployer {
	return &mockDeployer{
		ct:      ct,
		servers: make(map[string]map[string]model.ServerDef),
	}
}

func (m *mockDeployer) ClientType() model.ClientType { return m.ct }

func (m *mockDeployer) ConfigPath(projectPath string) string {
	return filepath.Join(projectPath, ".mcp.json")
}

func (m *mockDeployer) ReadServers(projectPath string) (map[string]model.ServerDef, error) {
	if m.servers[projectPath] == nil {
		return map[string]model.ServerDef{}, nil
	}
	result := make(map[string]model.ServerDef)
	for k, v := range m.servers[projectPath] {
		result[k] = v
	}
	return result, nil
}

func (m *mockDeployer) WriteServers(projectPath string, servers map[string]model.ServerDef) error {
	m.servers[projectPath] = make(map[string]model.ServerDef)
	for k, v := range servers {
		m.servers[projectPath][k] = v
	}
	return nil
}

func (m *mockDeployer) Bootstrap(projectPath string) error {
	if m.servers[projectPath] == nil {
		m.servers[projectPath] = make(map[string]model.ServerDef)
	}
	return nil
}

// setDeployed pre-populates the mock deployer with servers at a given path.
func (m *mockDeployer) setDeployed(projectPath string, servers map[string]model.ServerDef) {
	m.servers[projectPath] = make(map[string]model.ServerDef)
	for k, v := range servers {
		m.servers[projectPath][k] = v
	}
}

func newTestRegistry() *registry.Registry {
	return &registry.Registry{
		Servers: map[string]model.ServerDef{
			"github": {
				Name:      "github",
				Transport: model.TransportStdio,
				Command:   "npx",
				Args:      []string{"-y", "@modelcontextprotocol/server-github"},
				Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
			},
			"filesystem": {
				Name:      "filesystem",
				Transport: model.TransportStdio,
				Command:   "npx",
				Args:      []string{"-y", "@modelcontextprotocol/server-filesystem", "/"},
			},
			"qdrant": {
				Name:      "qdrant",
				Transport: model.TransportStdio,
				Command:   "uvx",
				Args:      []string{"mcp-server-qdrant"},
				Env:       map[string]string{"QDRANT_URL": "${QDRANT_URL}", "COLLECTION_NAME": "${COLLECTION_NAME}"},
			},
		},
		Skills:      make(map[string]model.SkillDef),
		Hooks:       make(map[string]model.HookDef),
		Permissions: make(map[string]model.PermissionRule),
		Templates:   make(map[string]model.TemplateDef),
		Tags: map[string][]string{
			"core": {"github", "filesystem"},
		},
	}
}

func newTestStore() *project.Store {
	return &project.Store{
		Projects: map[string]model.Project{
			"myproject": {
				Name:    "myproject",
				Path:    "/tmp/myproject",
				Clients: []model.ClientType{model.ClientClaudeCode},
				Tags:    []string{"core"},
				MCPs: []model.MCPAssignment{
					{
						Name: "qdrant",
						Overrides: &model.ServerOverride{
							Env: map[string]string{"COLLECTION_NAME": "test-data"},
						},
					},
				},
			},
		},
	}
}

func setupService(t *testing.T) (*Service, *mockDeployer) {
	t.Helper()
	reg := newTestRegistry()
	store := newTestStore()
	mock := newMockDeployer(model.ClientClaudeCode)

	configDir := t.TempDir()
	if err := reg.Save(filepath.Join(configDir, "registry.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(filepath.Join(configDir, "projects.yaml")); err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		registry:         reg,
		projects:         store,
		deployers:        map[model.ClientType]deploy.Deployer{model.ClientClaudeCode: mock},
		skillsDeployer:   &deploy.SkillsDeployer{},
		settingsDeployer: &deploy.SettingsDeployer{},
		claudeMDDeployer: &deploy.ClaudeMDDeployer{},
		profiles:         profile.NewManager(filepath.Join(configDir, "profiles")),
		backups:          backup.NewManager(filepath.Join(configDir, "backups")),
		configDir:        configDir,
	}

	return svc, mock
}

// ---- TESTS ----

func TestSyncProject_WritesCorrectServers(t *testing.T) {
	svc, mock := setupService(t)

	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	// Should have 3 servers: github, filesystem (from core tag), qdrant (from mcps).
	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 3 {
		t.Fatalf("expected 3 deployed servers, got %d", len(deployed))
	}

	gh, ok := deployed["github"]
	if !ok {
		t.Fatal("github not deployed")
	}
	if gh.Command != "npx" {
		t.Errorf("github command = %q, want %q", gh.Command, "npx")
	}

	qd, ok := deployed["qdrant"]
	if !ok {
		t.Fatal("qdrant not deployed")
	}
	if qd.Env["COLLECTION_NAME"] != "test-data" {
		t.Errorf("qdrant COLLECTION_NAME = %q, want %q", qd.Env["COLLECTION_NAME"], "test-data")
	}
	if qd.Env["QDRANT_URL"] != "${QDRANT_URL}" {
		t.Errorf("qdrant QDRANT_URL = %q, want %q", qd.Env["QDRANT_URL"], "${QDRANT_URL}")
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Action != SyncAdded {
			t.Errorf("server %q action = %q, want %q (first sync)", r.ServerName, r.Action, SyncAdded)
		}
	}
}

func TestSyncProject_PreservesUnmanaged(t *testing.T) {
	svc, mock := setupService(t)

	mock.setDeployed("/tmp/myproject", map[string]model.ServerDef{
		"custom-tool": {
			Name:      "custom-tool",
			Transport: model.TransportStdio,
			Command:   "my-tool",
			Args:      []string{"serve"},
		},
	})

	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 4 {
		t.Fatalf("expected 4 deployed servers, got %d", len(deployed))
	}

	ct, ok := deployed["custom-tool"]
	if !ok {
		t.Fatal("unmanaged server custom-tool was not preserved")
	}
	if ct.Command != "my-tool" {
		t.Errorf("custom-tool command = %q, want %q", ct.Command, "my-tool")
	}

	var unmanagedCount int
	for _, r := range results {
		if r.Action == SyncUnmanaged {
			unmanagedCount++
			if r.ServerName != "custom-tool" {
				t.Errorf("unmanaged server = %q, want %q", r.ServerName, "custom-tool")
			}
		}
	}
	if unmanagedCount != 1 {
		t.Errorf("expected 1 unmanaged result, got %d", unmanagedCount)
	}
}

func TestSyncProject_UnchangedServers(t *testing.T) {
	svc, _ := setupService(t)

	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}

	for _, r := range results {
		if r.Action != SyncUnchanged {
			t.Errorf("server %q action = %q, want %q (second sync)", r.ServerName, r.Action, SyncUnchanged)
		}
	}
}

func TestDriftReport_Synced(t *testing.T) {
	svc, _ := setupService(t)

	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	reports, err := svc.DriftReport("myproject")
	if err != nil {
		t.Fatalf("DriftReport: %v", err)
	}

	if len(reports) != 3 {
		t.Fatalf("expected 3 reports, got %d", len(reports))
	}

	for _, r := range reports {
		if r.Status != model.DriftSynced {
			t.Errorf("server %q status = %q, want %q", r.ServerName, r.Status, model.DriftSynced)
		}
	}
}

func TestDriftReport_Missing(t *testing.T) {
	svc, _ := setupService(t)

	reports, err := svc.DriftReport("myproject")
	if err != nil {
		t.Fatalf("DriftReport: %v", err)
	}

	for _, r := range reports {
		if r.Status != model.DriftMissing {
			t.Errorf("server %q status = %q, want %q", r.ServerName, r.Status, model.DriftMissing)
		}
		if r.Deployed != nil {
			t.Errorf("server %q should have nil Deployed", r.ServerName)
		}
		if r.Expected == nil {
			t.Errorf("server %q should have non-nil Expected", r.ServerName)
		}
	}
}

func TestDriftReport_Drifted(t *testing.T) {
	svc, mock := setupService(t)

	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	// Modify deployed github to simulate drift.
	deployed := mock.servers["/tmp/myproject"]
	gh := deployed["github"]
	gh.Command = "node"
	deployed["github"] = gh

	reports, err := svc.DriftReport("myproject")
	if err != nil {
		t.Fatalf("DriftReport: %v", err)
	}

	var ghReport *model.ServerDriftReport
	for i, r := range reports {
		if r.ServerName == "github" {
			ghReport = &reports[i]
			break
		}
	}

	if ghReport == nil {
		t.Fatal("github not found in drift report")
	}
	if ghReport.Status != model.DriftDrifted {
		t.Errorf("github status = %q, want %q", ghReport.Status, model.DriftDrifted)
	}
}

func TestDriftReport_Unmanaged(t *testing.T) {
	svc, mock := setupService(t)

	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	mock.servers["/tmp/myproject"]["rogue"] = model.ServerDef{
		Name:      "rogue",
		Transport: model.TransportStdio,
		Command:   "rogue-cmd",
	}

	reports, err := svc.DriftReport("myproject")
	if err != nil {
		t.Fatalf("DriftReport: %v", err)
	}

	var rogueReport *model.ServerDriftReport
	for i, r := range reports {
		if r.ServerName == "rogue" {
			rogueReport = &reports[i]
			break
		}
	}

	if rogueReport == nil {
		t.Fatal("rogue not found in drift report")
	}
	if rogueReport.Status != model.DriftUnmanaged {
		t.Errorf("rogue status = %q, want %q", rogueReport.Status, model.DriftUnmanaged)
	}
	if rogueReport.Expected != nil {
		t.Error("unmanaged server should have nil Expected")
	}
}

func TestImportFromFile_ExtractsCandidates(t *testing.T) {
	svc, mock := setupService(t)

	projectPath := "/tmp/importtest"
	mock.setDeployed(projectPath, map[string]model.ServerDef{
		"new-server": {
			Name:      "new-server",
			Transport: model.TransportStdio,
			Command:   "new-cmd",
		},
		"github": {
			Name:      "github",
			Transport: model.TransportStdio,
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-github"},
			Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
		},
	})

	candidates, err := svc.ImportFromFile(filepath.Join(projectPath, ".mcp.json"))
	if err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	if candidates[0].Name != "github" {
		t.Errorf("first candidate = %q, want %q", candidates[0].Name, "github")
	}
	if !candidates[0].Conflict {
		t.Error("github should be a conflict")
	}

	if candidates[1].Name != "new-server" {
		t.Errorf("second candidate = %q, want %q", candidates[1].Name, "new-server")
	}
	if candidates[1].Conflict {
		t.Error("new-server should not be a conflict")
	}
}

func TestApplyImport_AddsNonConflicting(t *testing.T) {
	svc, _ := setupService(t)

	candidates := []ImportCandidate{
		{
			Name: "brand-new",
			Server: model.ServerDef{
				Name:      "brand-new",
				Transport: model.TransportStdio,
				Command:   "brand-new-cmd",
			},
			Conflict:   false,
			Resolution: ImportPending,
		},
	}

	if err := svc.ApplyImport(candidates); err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}

	srv, ok := svc.GetServer("brand-new")
	if !ok {
		t.Fatal("brand-new not added to registry")
	}
	if srv.Command != "brand-new-cmd" {
		t.Errorf("brand-new command = %q, want %q", srv.Command, "brand-new-cmd")
	}
}

func TestApplyImport_ReplacesConflict(t *testing.T) {
	svc, _ := setupService(t)

	candidates := []ImportCandidate{
		{
			Name: "github",
			Server: model.ServerDef{
				Name:      "github",
				Transport: model.TransportStdio,
				Command:   "replaced-cmd",
			},
			Conflict:   true,
			Resolution: ImportReplace,
		},
	}

	if err := svc.ApplyImport(candidates); err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}

	srv, _ := svc.GetServer("github")
	if srv.Command != "replaced-cmd" {
		t.Errorf("github command = %q, want %q", srv.Command, "replaced-cmd")
	}
}

func TestApplyImport_KeepsConflict(t *testing.T) {
	svc, _ := setupService(t)

	originalCmd := svc.registry.Servers["github"].Command

	candidates := []ImportCandidate{
		{
			Name: "github",
			Server: model.ServerDef{
				Name:      "github",
				Transport: model.TransportStdio,
				Command:   "imported-cmd",
			},
			Conflict:   true,
			Resolution: ImportKeep,
		},
	}

	if err := svc.ApplyImport(candidates); err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}

	srv, _ := svc.GetServer("github")
	if srv.Command != originalCmd {
		t.Errorf("github command = %q, want %q (kept)", srv.Command, originalCmd)
	}
}

func TestApplyImport_RenamesConflict(t *testing.T) {
	svc, _ := setupService(t)

	candidates := []ImportCandidate{
		{
			Name: "github",
			Server: model.ServerDef{
				Name:      "github",
				Transport: model.TransportStdio,
				Command:   "imported-github",
			},
			Conflict:   true,
			Resolution: ImportRename,
			RenameTo:   "github-imported",
		},
	}

	if err := svc.ApplyImport(candidates); err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}

	orig, _ := svc.GetServer("github")
	if orig.Command != "npx" {
		t.Errorf("original github command = %q, want %q", orig.Command, "npx")
	}

	renamed, ok := svc.GetServer("github-imported")
	if !ok {
		t.Fatal("github-imported not added to registry")
	}
	if renamed.Command != "imported-github" {
		t.Errorf("github-imported command = %q, want %q", renamed.Command, "imported-github")
	}
}

func TestDiff_ProducesOutput(t *testing.T) {
	svc, mock := setupService(t)

	mock.setDeployed("/tmp/myproject", map[string]model.ServerDef{
		"github": {
			Name:      "github",
			Transport: model.TransportStdio,
			Command:   "node",
			Args:      []string{"-y", "@modelcontextprotocol/server-github"},
			Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
		},
	})

	diff, err := svc.Diff("myproject")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff")
	}

	if !strings.Contains(diff, "---") {
		t.Error("diff should contain --- marker")
	}
	if !strings.Contains(diff, "+++") {
		t.Error("diff should contain +++ marker")
	}
	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ hunk header")
	}
}

func TestDiff_EmptyWhenSynced(t *testing.T) {
	svc, _ := setupService(t)

	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	diff, err := svc.Diff("myproject")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if diff != "" {
		t.Errorf("expected empty diff after sync, got: %s", diff)
	}
}

func TestNew_LoadsFromDisk(t *testing.T) {
	configDir := t.TempDir()

	regContent := `servers:
  test-server:
    transport: stdio
    command: test-cmd
`
	projContent := `projects:
  test-project:
    path: /tmp/test
    clients: [claude-code]
    mcps:
      - test-server
`
	if err := os.WriteFile(filepath.Join(configDir, "registry.yaml"), []byte(regContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "projects.yaml"), []byte(projContent), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := New(configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, ok := svc.GetServer("test-server"); !ok {
		t.Error("registry should contain test-server")
	}
	if _, ok := svc.GetProject("test-project"); !ok {
		t.Error("projects should contain test-project")
	}
	if _, ok := svc.deployers[model.ClientClaudeCode]; !ok {
		t.Error("should have claude-code deployer")
	}
}

func TestServerDef_Equal_IdenticalServers(t *testing.T) {
	a := model.ServerDef{
		Name:      "a",
		Transport: model.TransportStdio,
		Command:   "cmd",
		Args:      []string{"arg1", "arg2"},
		Env:       map[string]string{"K": "V"},
	}
	b := model.ServerDef{
		Name:        "b",
		Description: "desc",
		Transport:   model.TransportStdio,
		Command:     "cmd",
		Args:        []string{"arg1", "arg2"},
		Env:         map[string]string{"K": "V"},
	}

	if !a.Equal(b) {
		t.Error("servers with same fields (ignoring name/description) should be equal")
	}
}

func TestServerDef_Equal_DifferentCommand(t *testing.T) {
	a := model.ServerDef{Transport: model.TransportStdio, Command: "cmd1"}
	b := model.ServerDef{Transport: model.TransportStdio, Command: "cmd2"}
	if a.Equal(b) {
		t.Error("servers with different commands should not be equal")
	}
}

func TestServerDef_Equal_DifferentArgs(t *testing.T) {
	a := model.ServerDef{Transport: model.TransportStdio, Args: []string{"a", "b"}}
	b := model.ServerDef{Transport: model.TransportStdio, Args: []string{"a", "c"}}
	if a.Equal(b) {
		t.Error("servers with different args should not be equal")
	}
}

func TestServerDef_Equal_DifferentEnv(t *testing.T) {
	a := model.ServerDef{Transport: model.TransportStdio, Env: map[string]string{"K": "V1"}}
	b := model.ServerDef{Transport: model.TransportStdio, Env: map[string]string{"K": "V2"}}
	if a.Equal(b) {
		t.Error("servers with different env should not be equal")
	}
}

func TestServerDef_Equal_NilVsEmptyMaps(t *testing.T) {
	a := model.ServerDef{Transport: model.TransportStdio}
	b := model.ServerDef{Transport: model.TransportStdio, Env: map[string]string{}}
	if !a.Equal(b) {
		t.Error("nil and empty map should be equal")
	}
}

func TestDetectClientType(t *testing.T) {
	tests := []struct {
		path       string
		wantClient model.ClientType
		wantPath   string
		wantErr    bool
	}{
		{"/tmp/project/.mcp.json", model.ClientClaudeCode, "/tmp/project", false},
		{"/home/user/.claude.json", model.ClientClaudeCode, "", false},
		{"/tmp/unknown.json", "", "", true},
	}

	for _, tt := range tests {
		ct, pp, err := detectClientType(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("detectClientType(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			continue
		}
		if ct != tt.wantClient {
			t.Errorf("detectClientType(%q) client = %q, want %q", tt.path, ct, tt.wantClient)
		}
		if pp != tt.wantPath {
			t.Errorf("detectClientType(%q) path = %q, want %q", tt.path, pp, tt.wantPath)
		}
	}
}

// setupServiceWithConfig creates a service with real config files on disk
// so that backup operations can actually copy files.
func setupServiceWithConfig(t *testing.T) (*Service, *mockDeployer, string) {
	t.Helper()
	configDir := t.TempDir()
	projectDir := filepath.Join(t.TempDir(), "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a real .mcp.json so backup can copy it.
	mcpContent := `{"mcpServers":{"existing":{"type":"stdio","command":"existing-cmd"}}}`
	if err := os.WriteFile(filepath.Join(projectDir, ".mcp.json"), []byte(mcpContent), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistry()
	store := &project.Store{
		Projects: map[string]model.Project{
			"myproject": {
				Name:    "myproject",
				Path:    projectDir,
				Clients: []model.ClientType{model.ClientClaudeCode},
				Tags:    []string{"core"},
				MCPs: []model.MCPAssignment{
					{
						Name: "qdrant",
						Overrides: &model.ServerOverride{
							Env: map[string]string{"COLLECTION_NAME": "test-data"},
						},
					},
				},
			},
		},
	}
	mock := newMockDeployer(model.ClientClaudeCode)

	if err := reg.Save(filepath.Join(configDir, "registry.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(filepath.Join(configDir, "projects.yaml")); err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		registry:         reg,
		projects:         store,
		deployers:        map[model.ClientType]deploy.Deployer{model.ClientClaudeCode: mock},
		skillsDeployer:   &deploy.SkillsDeployer{},
		settingsDeployer: &deploy.SettingsDeployer{},
		claudeMDDeployer: &deploy.ClaudeMDDeployer{},
		profiles:         profile.NewManager(filepath.Join(configDir, "profiles")),
		backups:          backup.NewManager(filepath.Join(configDir, "backups")),
		configDir:        configDir,
	}

	return svc, mock, projectDir
}

func TestBackupConfigs_CreatesBackup(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	entries, err := svc.BackupConfigs("myproject")
	if err != nil {
		t.Fatalf("BackupConfigs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup entry, got %d", len(entries))
	}
	if entries[0].BackupPath == "" {
		t.Error("expected non-empty BackupPath")
	}
	if entries[0].ClientType != model.ClientClaudeCode {
		t.Errorf("expected client type %q, got %q", model.ClientClaudeCode, entries[0].ClientType)
	}
}

func TestBackupConfigs_ProjectNotFound(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	_, err := svc.BackupConfigs("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestBackupConfigs_NoConfigFile(t *testing.T) {
	svc, _, projectDir := setupServiceWithConfig(t)

	// Remove the config file.
	_ = os.Remove(filepath.Join(projectDir, ".mcp.json"))

	entries, err := svc.BackupConfigs("myproject")
	if err != nil {
		t.Fatalf("BackupConfigs: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 backup entries when config file missing, got %d", len(entries))
	}
}

func TestBackupConfigs_SkipsMissingDeployer(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	// Add a client type that has no deployer.
	proj := svc.projects.Projects["myproject"]
	proj.Clients = append(proj.Clients, model.ClientCursor)
	svc.projects.Projects["myproject"] = proj

	entries, err := svc.BackupConfigs("myproject")
	if err != nil {
		t.Fatalf("BackupConfigs: %v", err)
	}
	for _, e := range entries {
		if e.ClientType == model.ClientCursor {
			t.Error("should not have backup entry for client with no deployer")
		}
	}
}

func TestListBackups_PopulatesSourcePath(t *testing.T) {
	svc, _, projectDir := setupServiceWithConfig(t)

	// Create a backup first.
	if _, err := svc.BackupConfigs("myproject"); err != nil {
		t.Fatalf("BackupConfigs: %v", err)
	}

	entries, err := svc.ListBackups("myproject")
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 backup entry")
	}
	expectedSource := filepath.Join(projectDir, ".mcp.json")
	if entries[0].SourcePath != expectedSource {
		t.Errorf("SourcePath = %q, want %q", entries[0].SourcePath, expectedSource)
	}
}

func TestListBackups_ProjectNotFound(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	_, err := svc.ListBackups("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestListBackups_Empty(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	entries, err := svc.ListBackups("myproject")
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListAllBackups_Delegates(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	// Create a backup.
	if _, err := svc.BackupConfigs("myproject"); err != nil {
		t.Fatal(err)
	}

	entries, err := svc.ListAllBackups()
	if err != nil {
		t.Fatalf("ListAllBackups: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 backup from ListAllBackups")
	}
}

func TestRestoreBackup_Delegates(t *testing.T) {
	svc, _, projectDir := setupServiceWithConfig(t)

	// Create a backup.
	backupEntries, err := svc.BackupConfigs("myproject")
	if err != nil {
		t.Fatal(err)
	}
	if len(backupEntries) == 0 {
		t.Fatal("expected backup to be created")
	}

	// Modify the original file.
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(`{"modified": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Restore the backup.
	entry := backupEntries[0]
	if err := svc.RestoreBackup(entry); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}

	// Verify the original file was restored.
	content, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "modified") {
		t.Error("expected restored content, got modified content")
	}
}

func TestSyncProject_CreatesBackup(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	_, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	// Verify that a backup was created during sync.
	entries, err := svc.ListBackups("myproject")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected sync to create a backup")
	}
}

func TestBackupConfigs_MultipleClients(t *testing.T) {
	svc, _, _ := setupServiceWithConfig(t)

	// Add a second mock deployer.
	mock2 := newMockDeployer(model.ClientCursor)
	svc.deployers[model.ClientCursor] = mock2

	// Add cursor to project clients.
	proj := svc.projects.Projects["myproject"]
	proj.Clients = append(proj.Clients, model.ClientCursor)
	svc.projects.Projects["myproject"] = proj

	entries, err := svc.BackupConfigs("myproject")
	if err != nil {
		t.Fatalf("BackupConfigs: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 backup entries for 2 clients, got %d", len(entries))
	}
	clients := map[model.ClientType]bool{}
	for _, e := range entries {
		clients[e.ClientType] = true
	}
	if !clients[model.ClientClaudeCode] {
		t.Error("expected backup for claude-code")
	}
	if !clients[model.ClientCursor] {
		t.Error("expected backup for cursor")
	}
}

func TestSyncAll(t *testing.T) {
	svc, _ := setupService(t)

	all, err := svc.SyncAll()
	if err != nil {
		t.Fatalf("SyncAll: %v", err)
	}

	results, ok := all["myproject"]
	if !ok {
		t.Fatal("myproject not in SyncAll results")
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results for myproject, got %d", len(results))
	}
}

// ---- Skills sync integration tests ----

func TestSyncProject_SyncsSkills(t *testing.T) {
	svc, _ := setupService(t)

	// Create a skill source file.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "go-reviewer.md")
	if err := os.WriteFile(sourceFile, []byte("# Go Reviewer\nReview Go code."), 0o644); err != nil {
		t.Fatal(err)
	}

	// Add skill to registry.
	svc.registry.Skills["go-reviewer"] = model.SkillDef{
		Name:   "go-reviewer",
		Source: sourceFile,
	}

	// Update project to reference the skill.
	proj := svc.projects.Projects["myproject"]
	proj.Skills = []string{"go-reviewer"}
	svc.projects.Projects["myproject"] = proj

	_, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	// Verify skill was deployed.
	skillPath := filepath.Join(proj.Path, ".claude", "skills", "go-reviewer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading skill: %v", err)
	}
	if string(content) != "# Go Reviewer\nReview Go code." {
		t.Errorf("unexpected skill content: %q", string(content))
	}
}

func TestSyncProject_SkillNotInRegistrySkipped(t *testing.T) {
	svc, _ := setupService(t)

	proj := svc.projects.Projects["myproject"]
	proj.Skills = []string{"nonexistent-skill"}
	svc.projects.Projects["myproject"] = proj

	// Skills not in registry (e.g., discovered from filesystem) should be
	// silently skipped during sync, not cause an error.
	_, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("expected no error for missing skill, got: %v", err)
	}
}

// ---- Settings sync integration tests ----

func TestSyncProject_SyncsHooksAndPermissions(t *testing.T) {
	svc, _ := setupService(t)

	// Add hook and permission to registry.
	svc.registry.Hooks["lint-bash"] = model.HookDef{
		Name:    "lint-bash",
		Event:   "PreToolUse",
		Matcher: "Bash",
		Command: "echo lint",
		Timeout: 5000,
	}
	svc.registry.Permissions["allow-bash"] = model.PermissionRule{
		Name: "allow-bash",
		Rule: "Bash(*)",
	}

	// Update project.
	proj := svc.projects.Projects["myproject"]
	proj.Hooks = []string{"lint-bash"}
	proj.Permissions = []string{"allow-bash"}
	svc.projects.Projects["myproject"] = proj

	_, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	// Verify settings file was created.
	settingsPath := filepath.Join(proj.Path, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}
	if !strings.Contains(string(data), "PreToolUse") {
		t.Error("settings should contain PreToolUse hook")
	}
	if !strings.Contains(string(data), "Bash(*)") {
		t.Error("settings should contain Bash(*) permission")
	}
}

// ---- CLAUDE.md sync integration tests ----

func TestSyncProject_SyncsClaudeMD(t *testing.T) {
	svc, _ := setupService(t)

	// Create template source.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "go-project.md")
	if err := os.WriteFile(sourceFile, []byte("# Go Project Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc.registry.Templates["go-project"] = model.TemplateDef{
		Name:   "go-project",
		Source: sourceFile,
	}

	proj := svc.projects.Projects["myproject"]
	proj.ClaudeMD = "go-project"
	svc.projects.Projects["myproject"] = proj

	_, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	target := filepath.Join(proj.Path, "CLAUDE.md")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(content), "# Go Project Template") {
		t.Error("CLAUDE.md should contain template content")
	}
}

// ---- Profile-aware sync tests ----

func TestSyncProject_AutoMigrate(t *testing.T) {
	svc, mock := setupService(t)

	// myproject has MCPs and tags but no active profile — should auto-migrate.
	_, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	// Verify 3 servers deployed (same as before migration).
	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 3 {
		t.Fatalf("expected 3 deployed servers, got %d", len(deployed))
	}

	// Verify migration created the "default" profile.
	proj, ok := svc.GetProject("myproject")
	if !ok {
		t.Fatal("project not found after sync")
	}
	if proj.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want %q", proj.ActiveProfile, "default")
	}
	pp, ok := proj.Profiles["default"]
	if !ok {
		t.Fatal("default profile not created")
	}
	if len(pp.MCPs) != 3 {
		t.Errorf("default profile MCPs = %d, want 3 (github, filesystem, qdrant)", len(pp.MCPs))
	}

	// Second sync should use the profile and not re-migrate.
	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("second SyncProject: %v", err)
	}
	for _, r := range results {
		if r.Action != SyncUnchanged {
			t.Errorf("server %q action = %q on second sync, want unchanged", r.ServerName, r.Action)
		}
	}
}

func TestSyncProfile_DeploysProfileItems(t *testing.T) {
	svc, mock := setupService(t)

	// Create a global profile with only 2 MCPs.
	prof := profile.Profile{
		Name: "light",
		MCPs: []string{"github", "filesystem"},
	}
	if err := svc.profiles.Save(prof); err != nil {
		t.Fatalf("saving profile: %v", err)
	}

	results, err := svc.SyncProfile("myproject", "light")
	if err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}

	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 2 {
		t.Fatalf("expected 2 deployed servers, got %d", len(deployed))
	}
	if _, ok := deployed["github"]; !ok {
		t.Error("github not deployed")
	}
	if _, ok := deployed["filesystem"]; !ok {
		t.Error("filesystem not deployed")
	}
	if _, ok := deployed["qdrant"]; ok {
		t.Error("qdrant should not be deployed with 'light' profile")
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSyncProfile_VanillaRemovesManaged(t *testing.T) {
	svc, mock := setupService(t)

	// First sync with full profile.
	fullProf := profile.Profile{
		Name: "full",
		MCPs: []string{"github", "filesystem", "qdrant"},
	}
	if err := svc.profiles.Save(fullProf); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SyncProfile("myproject", "full"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if len(mock.servers["/tmp/myproject"]) != 3 {
		t.Fatalf("expected 3 servers after full sync")
	}

	// Add an unmanaged server.
	mock.servers["/tmp/myproject"]["custom"] = model.ServerDef{
		Name:      "custom",
		Transport: model.TransportStdio,
		Command:   "custom-cmd",
	}

	// Sync with vanilla profile — should deploy 0 managed, preserve unmanaged.
	results, err := svc.SyncProfile("myproject", "vanilla")
	if err != nil {
		t.Fatalf("vanilla sync: %v", err)
	}

	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 1 {
		t.Fatalf("expected 1 server (unmanaged only), got %d", len(deployed))
	}
	if _, ok := deployed["custom"]; !ok {
		t.Error("unmanaged 'custom' server should be preserved")
	}

	var unmanagedCount int
	for _, r := range results {
		if r.Action == SyncUnmanaged {
			unmanagedCount++
		}
	}
	if unmanagedCount != 1 {
		t.Errorf("expected 1 unmanaged result, got %d", unmanagedCount)
	}
}

func TestSyncProfile_SwitchProfileReplacesItems(t *testing.T) {
	svc, mock := setupService(t)

	// Create two profiles.
	lightProf := profile.Profile{
		Name: "light",
		MCPs: []string{"github"},
	}
	fullProf := profile.Profile{
		Name: "full",
		MCPs: []string{"github", "filesystem", "qdrant"},
	}
	if err := svc.profiles.Save(lightProf); err != nil {
		t.Fatal(err)
	}
	if err := svc.profiles.Save(fullProf); err != nil {
		t.Fatal(err)
	}

	// Sync with light profile.
	if _, err := svc.SyncProfile("myproject", "light"); err != nil {
		t.Fatal(err)
	}
	if len(mock.servers["/tmp/myproject"]) != 1 {
		t.Fatalf("expected 1 server with light profile, got %d", len(mock.servers["/tmp/myproject"]))
	}

	// Switch to full profile.
	results, err := svc.SyncProfile("myproject", "full")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.servers["/tmp/myproject"]) != 3 {
		t.Fatalf("expected 3 servers with full profile, got %d", len(mock.servers["/tmp/myproject"]))
	}

	// github should be unchanged, others added.
	actionMap := make(map[string]SyncAction)
	for _, r := range results {
		actionMap[r.ServerName] = r.Action
	}
	if actionMap["github"] != SyncUnchanged {
		t.Errorf("github action = %q, want unchanged", actionMap["github"])
	}
	if actionMap["filesystem"] != SyncAdded {
		t.Errorf("filesystem action = %q, want added", actionMap["filesystem"])
	}
	if actionMap["qdrant"] != SyncAdded {
		t.Errorf("qdrant action = %q, want added", actionMap["qdrant"])
	}
}

func TestSyncProfile_PreservesProjectOverrides(t *testing.T) {
	svc, mock := setupService(t)

	// Project has an override for qdrant (COLLECTION_NAME=test-data).
	// Create a profile that includes qdrant.
	prof := profile.Profile{
		Name: "with-qdrant",
		MCPs: []string{"qdrant"},
	}
	if err := svc.profiles.Save(prof); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SyncProfile("myproject", "with-qdrant"); err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}

	deployed := mock.servers["/tmp/myproject"]
	qd, ok := deployed["qdrant"]
	if !ok {
		t.Fatal("qdrant not deployed")
	}
	if qd.Env["COLLECTION_NAME"] != "test-data" {
		t.Errorf("qdrant COLLECTION_NAME = %q, want %q", qd.Env["COLLECTION_NAME"], "test-data")
	}
	if qd.Env["QDRANT_URL"] != "${QDRANT_URL}" {
		t.Errorf("qdrant QDRANT_URL = %q, want %q", qd.Env["QDRANT_URL"], "${QDRANT_URL}")
	}
}

func TestSyncProfile_ProjectScopedProfile(t *testing.T) {
	svc, mock := setupService(t)

	// Add a project-scoped profile.
	proj := svc.projects.Projects["myproject"]
	proj.Profiles = map[string]model.ProjectProfile{
		"local": {
			Description: "Project-local profile",
			MCPs:        []string{"filesystem"},
		},
	}
	proj.ActiveProfile = "local"
	svc.projects.Projects["myproject"] = proj

	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 1 {
		t.Fatalf("expected 1 deployed server, got %d", len(deployed))
	}
	if _, ok := deployed["filesystem"]; !ok {
		t.Error("filesystem not deployed")
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSyncProfile_ProfileNotFound(t *testing.T) {
	svc, _ := setupService(t)

	_, err := svc.SyncProfile("myproject", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSyncProject_EmptyProject_NoMigration(t *testing.T) {
	svc, _ := setupService(t)

	// Add an empty project.
	emptyProj := model.Project{
		Name:    "empty",
		Path:    "/tmp/empty",
		Clients: []model.ClientType{model.ClientClaudeCode},
	}
	svc.projects.Projects["empty"] = emptyProj

	results, err := svc.SyncProject("empty")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	// No migration should happen for empty projects.
	proj, _ := svc.GetProject("empty")
	if proj.ActiveProfile != "" {
		t.Errorf("empty project should not get auto-migrated, got ActiveProfile=%q", proj.ActiveProfile)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty project, got %d", len(results))
	}
}

func TestHasLaunched(t *testing.T) {
	svc, _ := setupService(t)

	if svc.HasLaunched("myproject") {
		t.Error("project should not be launched initially")
	}
	if svc.HasLaunched("nonexistent") {
		t.Error("nonexistent project should return false")
	}

	if err := svc.MarkLaunched("myproject"); err != nil {
		t.Fatalf("MarkLaunched: %v", err)
	}
	if !svc.HasLaunched("myproject") {
		t.Error("project should be launched after MarkLaunched")
	}
}

func TestSetActiveProfile(t *testing.T) {
	svc, _ := setupService(t)

	// Create a profile.
	prof := profile.Profile{
		Name: "test-profile",
		MCPs: []string{"github"},
	}
	if err := svc.profiles.Save(prof); err != nil {
		t.Fatal(err)
	}

	if err := svc.SetActiveProfile("myproject", "test-profile"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}

	active, err := svc.GetActiveProfile("myproject")
	if err != nil {
		t.Fatalf("GetActiveProfile: %v", err)
	}
	if active != "test-profile" {
		t.Errorf("ActiveProfile = %q, want %q", active, "test-profile")
	}
}

func TestSetActiveProfile_Vanilla(t *testing.T) {
	svc, _ := setupService(t)

	if err := svc.SetActiveProfile("myproject", "vanilla"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}

	active, err := svc.GetActiveProfile("myproject")
	if err != nil {
		t.Fatal(err)
	}
	if active != "vanilla" {
		t.Errorf("ActiveProfile = %q, want %q", active, "vanilla")
	}
}

func TestSetActiveProfile_NotFound(t *testing.T) {
	svc, _ := setupService(t)

	err := svc.SetActiveProfile("myproject", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestSetActiveProfile_ProjectNotFound(t *testing.T) {
	svc, _ := setupService(t)

	err := svc.SetActiveProfile("nonexistent", "vanilla")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestSyncProfile_WithSkillsAndHooks(t *testing.T) {
	svc, _ := setupService(t)

	// Create skill source.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "reviewer.md")
	if err := os.WriteFile(sourceFile, []byte("# Reviewer"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Add skill and hook to registry.
	svc.registry.Skills["reviewer"] = model.SkillDef{
		Name:   "reviewer",
		Source: sourceFile,
	}
	svc.registry.Hooks["lint-bash"] = model.HookDef{
		Name:    "lint-bash",
		Event:   "PreToolUse",
		Matcher: "Bash",
		Command: "echo lint",
	}

	// Create profile with skills and hooks.
	prof := profile.Profile{
		Name:   "dev",
		MCPs:   []string{"github"},
		Skills: []string{"reviewer"},
		Hooks:  []string{"lint-bash"},
	}
	if err := svc.profiles.Save(prof); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SyncProfile("myproject", "dev"); err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}

	// Verify skill was deployed.
	proj, _ := svc.GetProject("myproject")
	skillPath := filepath.Join(proj.Path, ".claude", "skills", "reviewer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading skill: %v", err)
	}
	if string(content) != "# Reviewer" {
		t.Errorf("unexpected skill content: %q", string(content))
	}

	// Verify settings were deployed.
	settingsPath := filepath.Join(proj.Path, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}
	if !strings.Contains(string(data), "PreToolUse") {
		t.Error("settings should contain PreToolUse hook")
	}
}

func TestSaveProjectProfile(t *testing.T) {
	svc, _ := setupService(t)

	prof := profile.Profile{
		Name:        "frontend",
		Description: "Frontend work",
		MCPs:        []string{"github"},
		Skills:      []string{"reviewer"},
		EnvVars:     map[string]string{"MODE": "dev"},
		Isolation:   profile.IsolationNone,
	}

	if err := svc.SaveProjectProfile("myproject", "frontend", prof); err != nil {
		t.Fatalf("SaveProjectProfile: %v", err)
	}

	// Verify the profile was saved to the project.
	proj, ok := svc.projects.Get("myproject")
	if !ok {
		t.Fatal("project not found after save")
	}
	pp, ok := proj.Profiles["frontend"]
	if !ok {
		t.Fatal("profile 'frontend' not found in project profiles")
	}
	if pp.Description != "Frontend work" {
		t.Errorf("description: got %q, want %q", pp.Description, "Frontend work")
	}
	if len(pp.MCPs) != 1 || pp.MCPs[0] != "github" {
		t.Errorf("MCPs: got %v, want [github]", pp.MCPs)
	}
	if len(pp.Skills) != 1 || pp.Skills[0] != "reviewer" {
		t.Errorf("Skills: got %v, want [reviewer]", pp.Skills)
	}
	if pp.EnvVars["MODE"] != "dev" {
		t.Errorf("EnvVars[MODE]: got %q, want %q", pp.EnvVars["MODE"], "dev")
	}
}

func TestSaveProjectProfile_NotFound(t *testing.T) {
	svc, _ := setupService(t)
	err := svc.SaveProjectProfile("nonexistent", "test", profile.Profile{})
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSaveProjectProfile_SyncRoundTrip(t *testing.T) {
	svc, mock := setupService(t)

	// Save a profile to the project.
	prof := profile.Profile{
		Name: "light",
		MCPs: []string{"github"},
	}
	if err := svc.SaveProjectProfile("myproject", "light", prof); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetActiveProfile("myproject", "light"); err != nil {
		t.Fatal(err)
	}

	// Sync should now deploy only github (not qdrant or filesystem).
	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	deployed := mock.servers["/tmp/myproject"]
	if len(deployed) != 1 {
		t.Fatalf("expected 1 deployed server, got %d", len(deployed))
	}
	if _, ok := deployed["github"]; !ok {
		t.Error("expected github to be deployed")
	}
	_ = results
}

func TestSyncProjectToPath_DeploysToAlternatePath(t *testing.T) {
	svc, mock := setupService(t)

	altPath := t.TempDir()

	results, err := svc.SyncProjectToPath("myproject", altPath)
	if err != nil {
		t.Fatalf("SyncProjectToPath: %v", err)
	}

	// Servers should be deployed to the alternate path, not the project's original path.
	deployed := mock.servers[altPath]
	if len(deployed) == 0 {
		t.Fatal("expected servers deployed to alternate path")
	}

	// Original path should not have any servers (mock starts empty).
	origDeployed := mock.servers["/tmp/myproject"]
	if len(origDeployed) > 0 {
		t.Fatal("should not deploy to original project path")
	}

	// Should have the expected servers (github, filesystem, qdrant from tag expansion + MCPs).
	if _, ok := deployed["github"]; !ok {
		t.Error("expected github deployed to alt path")
	}

	_ = results
}

func TestSyncProjectToPath_ProjectNotFound(t *testing.T) {
	svc, _ := setupService(t)

	_, err := svc.SyncProjectToPath("nonexistent", "/tmp/alt")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}
