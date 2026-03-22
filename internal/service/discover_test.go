package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/project"
	"github.com/hystak/hystak/internal/registry"
)

func TestService_AutoDiscover_ImportsNew(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "proj")
	mkdirAll(t, projDir)

	// Write a .mcp.json with a server not in the registry
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"slack": map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@anthropic/mcp-slack"},
			},
		},
	}
	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".mcp.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	reg := registry.New()
	projStore := project.NewStore()
	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	imported, err := svc.AutoDiscover(projDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(imported) != 1 {
		t.Fatalf("expected 1 imported, got %d", len(imported))
	}
	if imported[0].Name != "slack" {
		t.Errorf("imported name = %q, want slack", imported[0].Name)
	}

	// Verify it's in the registry
	if _, ok := reg.Servers.Get("slack"); !ok {
		t.Error("slack not found in registry after auto-discover")
	}
}

func TestService_AutoDiscover_SkipsExisting(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "proj")
	mkdirAll(t, projDir)

	// Write a .mcp.json with a server already in registry
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
			},
		},
	}
	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".mcp.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	reg := registry.New()
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
	}); err != nil {
		t.Fatal(err)
	}

	projStore := project.NewStore()
	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	imported, err := svc.AutoDiscover(projDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(imported) != 0 {
		t.Errorf("expected 0 imported (already exists), got %d", len(imported))
	}
}

func TestService_AutoDiscover_NoConfigFile(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "empty")
	mkdirAll(t, projDir)

	reg := registry.New()
	projStore := project.NewStore()
	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	imported, err := svc.AutoDiscover(projDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(imported) != 0 {
		t.Errorf("expected 0 imported from empty dir, got %d", len(imported))
	}
}
