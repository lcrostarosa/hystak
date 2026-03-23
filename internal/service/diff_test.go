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

func TestService_DiffProject_NoDrift(t *testing.T) {
	svc, _ := setupTestService(t)

	// First sync to deploy
	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	results, err := svc.DiffProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.Status != model.DriftSynced {
			t.Errorf("server %q status = %q, want synced", r.ServerName, r.Status)
		}
	}
}

func TestService_DiffProject_Missing(t *testing.T) {
	svc, _ := setupTestService(t)

	// Don't sync — servers should be missing from deployed
	results, err := svc.DiffProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	statuses := make(map[string]model.DriftStatus)
	for _, r := range results {
		statuses[r.ServerName] = r.Status
	}
	if statuses["github"] != model.DriftMissing {
		t.Errorf("github status = %q, want missing", statuses["github"])
	}
	if statuses["postgres"] != model.DriftMissing {
		t.Errorf("postgres status = %q, want missing", statuses["postgres"])
	}
}

func TestService_DiffProject_Drifted(t *testing.T) {
	svc, projDir := setupTestService(t)

	// Sync first
	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	// Manually modify the deployed config to introduce drift
	dep := &deploy.ClaudeCodeDeployer{}
	servers, err := dep.ReadServers(projDir)
	if err != nil {
		t.Fatal(err)
	}
	gh := servers["github"]
	gh.Command = "node" // drift from "npx"
	servers["github"] = gh
	if err := dep.WriteServers(projDir, servers); err != nil {
		t.Fatal(err)
	}

	results, err := svc.DiffProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	resultMap := make(map[string]DiffResult)
	for _, r := range results {
		resultMap[r.ServerName] = r
	}

	ghResult, ok := resultMap["github"]
	if !ok {
		t.Fatal("github not in diff results")
	}
	if ghResult.Status != model.DriftDrifted {
		t.Errorf("github status = %q, want drifted", ghResult.Status)
	}
	// Expected should have the registry command "npx"
	if ghResult.Expected.Command != "npx" {
		t.Errorf("github Expected.Command = %q, want %q", ghResult.Expected.Command, "npx")
	}
	// Deployed should have the modified command "node"
	if ghResult.Deployed.Command != "node" {
		t.Errorf("github Deployed.Command = %q, want %q", ghResult.Deployed.Command, "node")
	}
	// Expected and Deployed should NOT be equal (confirming drift)
	if ghResult.Expected.Equal(ghResult.Deployed) {
		t.Error("Expected.Equal(Deployed) = true, want false for drifted server")
	}

	pgResult, ok := resultMap["postgres"]
	if !ok {
		t.Fatal("postgres not in diff results")
	}
	if pgResult.Status != model.DriftSynced {
		t.Errorf("postgres status = %q, want synced", pgResult.Status)
	}
	// For synced servers, Expected and Deployed should be equal
	if !pgResult.Expected.Equal(pgResult.Deployed) {
		t.Error("postgres Expected.Equal(Deployed) = false, want true for synced server")
	}
}

func TestService_DiffProject_Unmanaged(t *testing.T) {
	svc, projDir := setupTestService(t)

	// Sync first
	if _, err := svc.SyncProject("myproject"); err != nil {
		t.Fatal(err)
	}

	// Add an unmanaged server directly to the deployed config
	dep := &deploy.ClaudeCodeDeployer{}
	servers, err := dep.ReadServers(projDir)
	if err != nil {
		t.Fatal(err)
	}
	servers["manual"] = model.ServerDef{Transport: model.TransportStdio, Command: "node"}
	if err := dep.WriteServers(projDir, servers); err != nil {
		t.Fatal(err)
	}

	results, err := svc.DiffProject("myproject")
	if err != nil {
		t.Fatal(err)
	}

	statuses := make(map[string]model.DriftStatus)
	for _, r := range results {
		statuses[r.ServerName] = r.Status
	}
	if statuses["manual"] != model.DriftUnmanaged {
		t.Errorf("manual status = %q, want unmanaged", statuses["manual"])
	}
}

func TestService_DiffProject_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	_, err := svc.DiffProject("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestService_DiffProject_NoActiveProfile(t *testing.T) {
	tmp := t.TempDir()
	reg := registry.New()
	projStore := project.NewStore()
	if err := projStore.Add(model.Project{
		Name: "proj",
		Path: filepath.Join(tmp, "proj"),
	}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profMgr := profile.NewManager(profDir)
	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	_, err := svc.DiffProject("proj")
	if err == nil {
		t.Fatal("expected error for missing active profile")
	}
}
