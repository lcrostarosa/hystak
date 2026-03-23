package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapMCPJSON_CreatesFile(t *testing.T) {
	tmp := t.TempDir()

	if err := BootstrapMCPJSON(tmp); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmp, ".mcp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	servers, ok := parsed["mcpServers"]
	if !ok {
		t.Fatal("missing mcpServers key")
	}
	serverMap, ok := servers.(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers is not an object, got %T", servers)
	}
	if len(serverMap) != 0 {
		t.Errorf("mcpServers should be empty, got %d entries", len(serverMap))
	}
}

func TestBootstrapMCPJSON_Idempotent(t *testing.T) {
	tmp := t.TempDir()

	if err := BootstrapMCPJSON(tmp); err != nil {
		t.Fatal(err)
	}

	// Write custom content to verify it's preserved
	path := filepath.Join(tmp, ".mcp.json")
	custom := []byte(`{"mcpServers":{"github":{}}}`)
	if err := os.WriteFile(path, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := BootstrapMCPJSON(tmp); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(custom) {
		t.Error("BootstrapMCPJSON overwrote existing file")
	}
}

func TestBootstrapMCPJSON_InvalidDir(t *testing.T) {
	err := BootstrapMCPJSON("/nonexistent/dir")
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}
