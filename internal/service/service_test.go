package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lcrostarosa/hystak/internal/deploy"
	"github.com/lcrostarosa/hystak/internal/model"
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
		Registry:  reg,
		Projects:  store,
		Deployers: map[model.ClientType]deploy.Deployer{model.ClientClaudeCode: mock},
		ConfigDir: configDir,
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

	srv, ok := svc.Registry.Get("brand-new")
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

	srv, _ := svc.Registry.Get("github")
	if srv.Command != "replaced-cmd" {
		t.Errorf("github command = %q, want %q", srv.Command, "replaced-cmd")
	}
}

func TestApplyImport_KeepsConflict(t *testing.T) {
	svc, _ := setupService(t)

	originalCmd := svc.Registry.Servers["github"].Command

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

	srv, _ := svc.Registry.Get("github")
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

	orig, _ := svc.Registry.Get("github")
	if orig.Command != "npx" {
		t.Errorf("original github command = %q, want %q", orig.Command, "npx")
	}

	renamed, ok := svc.Registry.Get("github-imported")
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

	if _, ok := svc.Registry.Get("test-server"); !ok {
		t.Error("registry should contain test-server")
	}
	if _, ok := svc.Projects.Get("test-project"); !ok {
		t.Error("projects should contain test-project")
	}
	if _, ok := svc.Deployers[model.ClientClaudeCode]; !ok {
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
