package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/discovery"
	"github.com/hystak/hystak/internal/model"
)

func TestService_PrepareImport(t *testing.T) {
	svc, _ := setupTestService(t)
	tmp := t.TempDir()

	// Write a .mcp.json with one new and one existing server
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{"type": "stdio", "command": "npx"},
			"slack":  map[string]interface{}{"type": "stdio", "command": "npx"},
		},
	}
	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmp, ".mcp.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	candidates, err := svc.PrepareImport(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(candidates))
	}

	// github should be a conflict (already in registry)
	conflicts := 0
	for _, c := range candidates {
		if c.Conflict {
			conflicts++
			if c.Name != "github" {
				t.Errorf("expected github to conflict, got %q", c.Name)
			}
		}
	}
	if conflicts != 1 {
		t.Errorf("conflicts = %d, want 1", conflicts)
	}
}

func TestService_ApplyImport_NewServer(t *testing.T) {
	svc, _ := setupTestService(t)

	candidates := []ImportCandidate{
		{
			Candidate:  newCandidate("slack", model.TransportStdio, "npx"),
			Resolution: ImportReplace,
		},
	}

	imported, err := svc.ApplyImport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 1 {
		t.Errorf("imported = %d, want 1", imported)
	}

	if _, ok := svc.GetServer("slack"); !ok {
		t.Error("slack should exist after import")
	}
}

func TestService_ApplyImport_Replace(t *testing.T) {
	svc, _ := setupTestService(t)

	candidates := []ImportCandidate{
		{
			Candidate:  newCandidate("github", model.TransportStdio, "node"),
			Conflict:   true,
			Resolution: ImportReplace,
		},
	}

	imported, err := svc.ApplyImport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 1 {
		t.Errorf("imported = %d, want 1", imported)
	}

	srv, ok := svc.GetServer("github")
	if !ok {
		t.Fatal("github should exist")
	}
	if srv.Command != "node" {
		t.Errorf("Command = %q, want node (replaced)", srv.Command)
	}
}

func TestService_ApplyImport_Keep(t *testing.T) {
	svc, _ := setupTestService(t)

	candidates := []ImportCandidate{
		{
			Candidate:  newCandidate("github", model.TransportStdio, "node"),
			Conflict:   true,
			Resolution: ImportKeep,
		},
	}

	imported, err := svc.ApplyImport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 0 {
		t.Errorf("imported = %d, want 0 (keep)", imported)
	}

	srv, _ := svc.GetServer("github")
	if srv.Command != "npx" {
		t.Errorf("Command = %q, want npx (kept original)", srv.Command)
	}
}

func TestService_ApplyImport_Skip(t *testing.T) {
	svc, _ := setupTestService(t)

	candidates := []ImportCandidate{
		{
			Candidate:  newCandidate("slack", model.TransportStdio, "npx"),
			Resolution: ImportSkip,
		},
	}

	imported, err := svc.ApplyImport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 0 {
		t.Errorf("imported = %d, want 0 (skip)", imported)
	}
}

func TestService_ApplyImport_Pending_Errors(t *testing.T) {
	svc, _ := setupTestService(t)

	candidates := []ImportCandidate{
		{
			Candidate:  newCandidate("slack", model.TransportStdio, "npx"),
			Resolution: ImportPending,
		},
	}

	_, err := svc.ApplyImport(candidates)
	if err == nil {
		t.Fatal("expected error for pending resolution")
	}
}

func TestService_ApplyImport_Rename(t *testing.T) {
	svc, _ := setupTestService(t)

	candidates := []ImportCandidate{
		{
			Candidate:  newCandidate("github", model.TransportStdio, "node"),
			Conflict:   true,
			Resolution: ImportRename,
			RenameTo:   "github-v2",
		},
	}

	imported, err := svc.ApplyImport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 1 {
		t.Errorf("imported = %d, want 1", imported)
	}

	if _, ok := svc.GetServer("github-v2"); !ok {
		t.Error("github-v2 should exist after rename import")
	}
	// Original should still exist
	if _, ok := svc.GetServer("github"); !ok {
		t.Error("original github should still exist")
	}
}

func newCandidate(name string, transport model.Transport, command string) discovery.Candidate {
	return discovery.Candidate{
		Name: name,
		Server: model.ServerDef{
			Name:      name,
			Transport: transport,
			Command:   command,
		},
		Source: "/test",
	}
}
