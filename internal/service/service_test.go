package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/project"
	"github.com/hystak/hystak/internal/registry"
)

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func setupTestService(t *testing.T) (*Service, string) {
	t.Helper()
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "myproject")
	mkdirAll(t, projDir)

	reg := registry.New()
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@anthropic/mcp-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "postgres",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@anthropic/mcp-postgres"},
	}); err != nil {
		t.Fatal(err)
	}

	projStore := project.NewStore()
	if err := projStore.Add(model.Project{
		Name:          "myproject",
		Path:          projDir,
		ActiveProfile: "dev",
	}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	if err := profMgr.Save(model.ProjectProfile{
		Name: "dev",
		MCPs: []model.MCPAssignment{
			{Name: "github"},
			{Name: "postgres"},
		},
		Isolation: model.IsolationNone,
	}); err != nil {
		t.Fatal(err)
	}

	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	return svc, projDir
}

func TestService_SyncProject(t *testing.T) {
	svc, _ := setupTestService(t)

	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	actions := make(map[string]SyncAction)
	for _, r := range results {
		actions[r.Name] = r.Action
	}

	if actions["github"] != SyncAdded {
		t.Errorf("github action = %q, want added", actions["github"])
	}
	if actions["postgres"] != SyncAdded {
		t.Errorf("postgres action = %q, want added", actions["postgres"])
	}
}

func TestService_SyncProject_SecondSync_Unchanged(t *testing.T) {
	svc, _ := setupTestService(t)

	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	actions := make(map[string]SyncAction)
	for _, r := range results {
		actions[r.Name] = r.Action
	}

	if actions["github"] != SyncUnchanged {
		t.Errorf("github action = %q, want unchanged", actions["github"])
	}
	if actions["postgres"] != SyncUnchanged {
		t.Errorf("postgres action = %q, want unchanged", actions["postgres"])
	}
}

func TestService_SyncProject_PreservesUnmanaged(t *testing.T) {
	svc, projDir := setupTestService(t)

	// Pre-populate with an unmanaged server
	dep := &deploy.ClaudeCodeDeployer{}
	if err := dep.Bootstrap(projDir); err != nil {
		t.Fatal(err)
	}
	// Write unmanaged + fake managed so we can test preservation
	allServers := map[string]model.ServerDef{
		"manual": {Transport: model.TransportStdio, Command: "node"},
	}
	if err := dep.WriteServers(projDir, allServers); err != nil {
		t.Fatal(err)
	}

	// Sync — should preserve "manual"
	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	// Check results report unmanaged
	actions := make(map[string]SyncAction)
	for _, r := range results {
		actions[r.Name] = r.Action
	}
	if actions["manual"] != SyncUnmanaged {
		t.Errorf("manual action = %q, want unmanaged", actions["manual"])
	}

	// Verify on disk
	servers, err := dep.ReadServers(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["manual"]; !ok {
		t.Error("unmanaged server 'manual' was removed during sync")
	}
	if _, ok := servers["github"]; !ok {
		t.Error("managed server 'github' missing after sync")
	}
}

func TestService_SyncProject_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.SyncProject("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestService_SyncProject_MissingServer(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "proj")
	mkdirAll(t, projDir)

	reg := registry.New()

	projStore := project.NewStore()
	if err := projStore.Add(model.Project{
		Name:          "proj",
		Path:          projDir,
		ActiveProfile: "bad",
	}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	if err := profMgr.Save(model.ProjectProfile{
		Name:      "bad",
		MCPs:      []model.MCPAssignment{{Name: "nonexistent"}},
		Isolation: model.IsolationNone,
	}); err != nil {
		t.Fatal(err)
	}

	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	_, err := svc.SyncProject("proj")
	if err == nil {
		t.Fatal("expected error for missing server reference")
	}
}

func TestService_SyncProject_NoActiveProfile(t *testing.T) {
	tmp := t.TempDir()
	reg := registry.New()
	projStore := project.NewStore()
	if err := projStore.Add(model.Project{
		Name: "proj",
		Path: filepath.Join(tmp, "proj"),
	}); err != nil {
		t.Fatal(err)
	}

	profMgr := profile.NewManager(filepath.Join(tmp, "profiles"))
	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	_, err := svc.SyncProject("proj")
	if err == nil {
		t.Fatal("expected error for missing active profile")
	}
}

func TestService_SyncProject_WithOverrides(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "proj")
	mkdirAll(t, projDir)

	reg := registry.New()
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Env:       map[string]string{"A": "1", "B": "2"},
	}); err != nil {
		t.Fatal(err)
	}

	projStore := project.NewStore()
	if err := projStore.Add(model.Project{
		Name:          "proj",
		Path:          projDir,
		ActiveProfile: "custom",
	}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	cmd := "node"
	if err := profMgr.Save(model.ProjectProfile{
		Name: "custom",
		MCPs: []model.MCPAssignment{
			{
				Name: "github",
				Overrides: &model.ServerOverride{
					Command: &cmd,
					Env:     map[string]string{"B": "3", "C": "4"},
				},
			},
		},
		Isolation: model.IsolationNone,
	}); err != nil {
		t.Fatal(err)
	}

	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	_, err := svc.SyncProject("proj")
	if err != nil {
		t.Fatal(err)
	}

	servers, err := dep.ReadServers(projDir)
	if err != nil {
		t.Fatal(err)
	}

	gh, ok := servers["github"]
	if !ok {
		t.Fatal("github not found in deployed servers")
	}
	if gh.Command != "node" {
		t.Errorf("Command = %q, want node (override)", gh.Command)
	}
	if gh.Env["A"] != "1" {
		t.Errorf("Env[A] = %q, want 1 (preserved from base)", gh.Env["A"])
	}
	if gh.Env["B"] != "3" {
		t.Errorf("Env[B] = %q, want 3 (override wins)", gh.Env["B"])
	}
	if gh.Env["C"] != "4" {
		t.Errorf("Env[C] = %q, want 4 (added by override)", gh.Env["C"])
	}
}

func TestService_SyncProject_RemovesPreviouslyManaged(t *testing.T) {
	svc, projDir := setupTestService(t)

	// First sync — deploys github + postgres
	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	// Change profile to only have github (remove postgres)
	proj, _ := svc.Projects.Get("myproject")
	profMgr := svc.Profiles
	if err := profMgr.Save(model.ProjectProfile{
		Name:      "dev",
		MCPs:      []model.MCPAssignment{{Name: "github"}},
		Isolation: model.IsolationNone,
	}); err != nil {
		t.Fatal(err)
	}
	// Ensure managed_mcps reflects first sync
	_ = proj

	// Second sync — postgres should be removed
	results, err := svc.SyncProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	actions := make(map[string]SyncAction)
	for _, r := range results {
		actions[r.Name] = r.Action
	}
	if actions["postgres"] != SyncRemoved {
		t.Errorf("postgres action = %q, want removed", actions["postgres"])
	}

	// Verify on disk
	dep := &deploy.ClaudeCodeDeployer{}
	servers, err := dep.ReadServers(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["postgres"]; ok {
		t.Error("postgres should have been removed from deployed config")
	}
	if _, ok := servers["github"]; !ok {
		t.Error("github should still be deployed")
	}
}
