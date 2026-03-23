package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestScanFile_WithServers(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".mcp.json")

	content := `{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-github"],
      "env": {"GITHUB_TOKEN": "tok"}
    },
    "remote": {
      "type": "sse",
      "url": "https://example.com/sse",
      "headers": {"Auth": "Bearer tok"}
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates, err := ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}

	byName := make(map[string]Candidate)
	for _, c := range candidates {
		byName[c.Name] = c
	}

	gh, ok := byName["github"]
	if !ok {
		t.Fatal("missing candidate 'github'")
	}
	if gh.Server.Transport != model.TransportStdio {
		t.Errorf("github.Transport = %q", gh.Server.Transport)
	}
	if gh.Server.Command != "npx" {
		t.Errorf("github.Command = %q", gh.Server.Command)
	}
	if gh.Server.Env["GITHUB_TOKEN"] != "tok" {
		t.Errorf("github.Env[GITHUB_TOKEN] = %q", gh.Server.Env["GITHUB_TOKEN"])
	}
	if gh.Source != path {
		t.Errorf("github.Source = %q, want %q", gh.Source, path)
	}

	remote, ok := byName["remote"]
	if !ok {
		t.Fatal("missing candidate 'remote'")
	}
	if remote.Server.Transport != model.TransportSSE {
		t.Errorf("remote.Transport = %q", remote.Server.Transport)
	}
	if remote.Server.URL != "https://example.com/sse" {
		t.Errorf("remote.URL = %q", remote.Server.URL)
	}
	if remote.Server.Headers["Auth"] != "Bearer tok" {
		t.Errorf("remote.Headers[Auth] = %q", remote.Server.Headers["Auth"])
	}
}

func TestScanFile_NonexistentFile(t *testing.T) {
	candidates, err := ScanFile("/nonexistent/.mcp.json")
	if err != nil {
		t.Fatal(err)
	}
	if candidates != nil {
		t.Errorf("expected nil, got %v", candidates)
	}
}

func TestScanFile_EmptyMCPServers(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".mcp.json")

	if err := os.WriteFile(path, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates, err := ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Errorf("got %d candidates, want 0", len(candidates))
	}
}

func TestScanFile_NoMCPServersKey(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".claude.json")

	if err := os.WriteFile(path, []byte(`{"otherKey": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates, err := ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if candidates != nil {
		t.Errorf("expected nil, got %v", candidates)
	}
}

func TestScanFile_MalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".mcp.json")

	if err := os.WriteFile(path, []byte(`{invalid`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ScanFile(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestScanProject(t *testing.T) {
	tmp := t.TempDir()
	content := `{"mcpServers":{"github":{"type":"stdio","command":"npx"}}}`
	if err := os.WriteFile(filepath.Join(tmp, ".mcp.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates, err := ScanProject(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].Name != "github" {
		t.Errorf("Name = %q, want github", candidates[0].Name)
	}
}

func TestScanProject_NoFile(t *testing.T) {
	tmp := t.TempDir()

	candidates, err := ScanProject(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if candidates != nil {
		t.Errorf("expected nil, got %v", candidates)
	}
}
