package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rbbydotdev/hystak/internal/model"
)

func TestClaudeCodeDeployer_ClientType(t *testing.T) {
	d := &ClaudeCodeDeployer{}
	if d.ClientType() != model.ClientClaudeCode {
		t.Errorf("expected %s, got %s", model.ClientClaudeCode, d.ClientType())
	}
}

func TestClaudeCodeDeployer_ConfigPath_Project(t *testing.T) {
	d := &ClaudeCodeDeployer{}
	got := d.ConfigPath("/some/project")
	want := "/some/project/.mcp.json"
	if got != want {
		t.Errorf("ConfigPath(/some/project) = %q, want %q", got, want)
	}
}

func TestClaudeCodeDeployer_ConfigPath_GlobalEmpty(t *testing.T) {
	d := &ClaudeCodeDeployer{}
	got := d.ConfigPath("")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".claude.json")
	if got != want {
		t.Errorf("ConfigPath('') = %q, want %q", got, want)
	}
}

func TestClaudeCodeDeployer_ConfigPath_GlobalTilde(t *testing.T) {
	d := &ClaudeCodeDeployer{}
	got := d.ConfigPath("~")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".claude.json")
	if got != want {
		t.Errorf("ConfigPath('~') = %q, want %q", got, want)
	}
}

func TestClaudeCodeDeployer_ReadServers_StdioAndHTTP(t *testing.T) {
	dir := t.TempDir()
	mcpJSON := `{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "remote-api": {
      "type": "http",
      "url": "https://mcp.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}"
      }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeCodeDeployer{}
	servers, err := d.ReadServers(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	gh := servers["github"]
	if gh.Name != "github" {
		t.Errorf("github.Name = %q, want %q", gh.Name, "github")
	}
	if gh.Transport != model.TransportStdio {
		t.Errorf("github.Transport = %q, want %q", gh.Transport, model.TransportStdio)
	}
	if gh.Command != "npx" {
		t.Errorf("github.Command = %q, want %q", gh.Command, "npx")
	}
	if len(gh.Args) != 2 || gh.Args[0] != "-y" {
		t.Errorf("github.Args = %v, want [-y @modelcontextprotocol/server-github]", gh.Args)
	}
	if gh.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("github.Env[GITHUB_TOKEN] = %q, want %q", gh.Env["GITHUB_TOKEN"], "${GITHUB_TOKEN}")
	}

	api := servers["remote-api"]
	if api.Transport != model.TransportHTTP {
		t.Errorf("remote-api.Transport = %q, want %q", api.Transport, model.TransportHTTP)
	}
	if api.URL != "https://mcp.example.com/mcp" {
		t.Errorf("remote-api.URL = %q, want expected", api.URL)
	}
	if api.Headers["Authorization"] != "Bearer ${API_TOKEN}" {
		t.Errorf("remote-api.Headers[Authorization] = %q", api.Headers["Authorization"])
	}
}

func TestClaudeCodeDeployer_ReadServers_MissingFile(t *testing.T) {
	dir := t.TempDir()
	d := &ClaudeCodeDeployer{}
	servers, err := d.ReadServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Errorf("expected empty map, got %d servers", len(servers))
	}
}

func TestClaudeCodeDeployer_ReadServers_NoMCPServers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"other": "data"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeCodeDeployer{}
	servers, err := d.ReadServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Errorf("expected empty map, got %d servers", len(servers))
	}
}

func TestClaudeCodeDeployer_ReadServers_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{invalid`), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeCodeDeployer{}
	_, err := d.ReadServers(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestClaudeCodeDeployer_WriteServers_CreatesCorrectJSON(t *testing.T) {
	dir := t.TempDir()
	d := &ClaudeCodeDeployer{}

	servers := map[string]model.ServerDef{
		"github": {
			Name:      "github",
			Transport: model.TransportStdio,
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-github"},
			Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
		},
		"remote-api": {
			Name:      "remote-api",
			Transport: model.TransportHTTP,
			URL:       "https://mcp.example.com/mcp",
			Headers:   map[string]string{"Authorization": "Bearer ${API_TOKEN}"},
		},
	}

	if err := d.WriteServers(dir, servers); err != nil {
		t.Fatal(err)
	}

	// Read back and verify structure.
	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	var ccServers map[string]claudeCodeServer
	if err := json.Unmarshal(raw["mcpServers"], &ccServers); err != nil {
		t.Fatal(err)
	}

	gh := ccServers["github"]
	if gh.Type != "stdio" {
		t.Errorf("github.type = %q, want %q", gh.Type, "stdio")
	}
	if gh.Command != "npx" {
		t.Errorf("github.command = %q, want %q", gh.Command, "npx")
	}

	api := ccServers["remote-api"]
	if api.Type != "http" {
		t.Errorf("remote-api.type = %q, want %q", api.Type, "http")
	}
	if api.URL != "https://mcp.example.com/mcp" {
		t.Errorf("remote-api.url = %q", api.URL)
	}
}

func TestClaudeCodeDeployer_WriteServers_PreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	// Simulate existing ~/.claude.json with other keys.
	existing := `{
  "numStartups": 42,
  "theme": "dark",
  "mcpServers": {
    "old-server": {
      "type": "stdio",
      "command": "old"
    }
  },
  "projects": {}
}`
	configPath := filepath.Join(dir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeCodeDeployer{}
	servers := map[string]model.ServerDef{
		"github": {
			Name:      "github",
			Transport: model.TransportStdio,
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-github"},
		},
	}

	if err := d.WriteServers(dir, servers); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	// Verify other keys preserved.
	var numStartups int
	if err := json.Unmarshal(raw["numStartups"], &numStartups); err != nil {
		t.Fatal(err)
	}
	if numStartups != 42 {
		t.Errorf("numStartups = %d, want 42", numStartups)
	}

	var theme string
	if err := json.Unmarshal(raw["theme"], &theme); err != nil {
		t.Fatal(err)
	}
	if theme != "dark" {
		t.Errorf("theme = %q, want %q", theme, "dark")
	}

	if _, ok := raw["projects"]; !ok {
		t.Error("projects key was not preserved")
	}

	// Verify mcpServers was updated (old-server gone, github present).
	var ccServers map[string]claudeCodeServer
	if err := json.Unmarshal(raw["mcpServers"], &ccServers); err != nil {
		t.Fatal(err)
	}
	if _, ok := ccServers["old-server"]; ok {
		t.Error("old-server should have been replaced")
	}
	if _, ok := ccServers["github"]; !ok {
		t.Error("github should be present")
	}
}

func TestClaudeCodeDeployer_Bootstrap_Project(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeCodeDeployer{}
	if err := d.Bootstrap(projDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(projDir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := raw["mcpServers"]; !ok {
		t.Error("mcpServers key missing from bootstrapped file")
	}
}

func TestClaudeCodeDeployer_Bootstrap_Idempotent(t *testing.T) {
	dir := t.TempDir()
	existing := `{"mcpServers":{"github":{"type":"stdio","command":"npx"}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeCodeDeployer{}
	if err := d.Bootstrap(dir); err != nil {
		t.Fatal(err)
	}

	// Verify existing content was not overwritten.
	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Error("bootstrap overwrote existing file")
	}
}

func TestClaudeCodeDeployer_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	d := &ClaudeCodeDeployer{}

	original := map[string]model.ServerDef{
		"github": {
			Name:      "github",
			Transport: model.TransportStdio,
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-github"},
			Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
		},
		"qdrant": {
			Name:      "qdrant",
			Transport: model.TransportStdio,
			Command:   "uvx",
			Args:      []string{"mcp-server-qdrant"},
			Env: map[string]string{
				"QDRANT_URL":      "${QDRANT_URL}",
				"COLLECTION_NAME": "agent-context",
			},
		},
		"remote-api": {
			Name:      "remote-api",
			Transport: model.TransportHTTP,
			URL:       "https://mcp.example.com/mcp",
			Headers:   map[string]string{"Authorization": "Bearer ${API_TOKEN}"},
		},
	}

	if err := d.WriteServers(dir, original); err != nil {
		t.Fatal(err)
	}

	readBack, err := d.ReadServers(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(readBack) != len(original) {
		t.Fatalf("round-trip: got %d servers, want %d", len(readBack), len(original))
	}

	for name, orig := range original {
		got, ok := readBack[name]
		if !ok {
			t.Errorf("round-trip: server %q missing", name)
			continue
		}
		if got.Name != name {
			t.Errorf("round-trip: %s.Name = %q, want %q", name, got.Name, name)
		}
		if got.Transport != orig.Transport {
			t.Errorf("round-trip: %s.Transport = %q, want %q", name, got.Transport, orig.Transport)
		}
		if got.Command != orig.Command {
			t.Errorf("round-trip: %s.Command = %q, want %q", name, got.Command, orig.Command)
		}
		if got.URL != orig.URL {
			t.Errorf("round-trip: %s.URL = %q, want %q", name, got.URL, orig.URL)
		}
		if len(got.Args) != len(orig.Args) {
			t.Errorf("round-trip: %s.Args = %v, want %v", name, got.Args, orig.Args)
		}
		for k, v := range orig.Env {
			if got.Env[k] != v {
				t.Errorf("round-trip: %s.Env[%s] = %q, want %q", name, k, got.Env[k], v)
			}
		}
		for k, v := range orig.Headers {
			if got.Headers[k] != v {
				t.Errorf("round-trip: %s.Headers[%s] = %q, want %q", name, k, got.Headers[k], v)
			}
		}
		// Description should NOT round-trip (stripped in JSON).
		if got.Description != "" {
			t.Errorf("round-trip: %s.Description should be empty, got %q", name, got.Description)
		}
	}
}

func TestClaudeCodeDeployer_WriteServers_StripsDescription(t *testing.T) {
	dir := t.TempDir()
	d := &ClaudeCodeDeployer{}

	servers := map[string]model.ServerDef{
		"github": {
			Name:        "github",
			Description: "GitHub API integration",
			Transport:   model.TransportStdio,
			Command:     "npx",
		},
	}

	if err := d.WriteServers(dir, servers); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	// Verify description is not in the JSON.
	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	serverRaw := raw["mcpServers"]["github"]
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(serverRaw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["description"]; ok {
		t.Error("description field should not be present in Claude Code JSON output")
	}
	if _, ok := fields["name"]; ok {
		t.Error("name field should not be present in Claude Code JSON output (name is the map key)")
	}
}

func TestNewDeployer(t *testing.T) {
	tests := []struct {
		ct      model.ClientType
		wantErr bool
	}{
		{model.ClientClaudeCode, false},
		{model.ClientClaudeDesktop, true},
		{model.ClientCursor, true},
		{model.ClientType("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			d, err := NewDeployer(tt.ct)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if d.ClientType() != tt.ct {
				t.Errorf("ClientType() = %s, want %s", d.ClientType(), tt.ct)
			}
		})
	}
}

func TestClaudeCodeDeployer_WriteServers_EmptyEnvAndArgs(t *testing.T) {
	dir := t.TempDir()
	d := &ClaudeCodeDeployer{}

	servers := map[string]model.ServerDef{
		"minimal": {
			Name:      "minimal",
			Transport: model.TransportStdio,
			Command:   "echo",
		},
	}

	if err := d.WriteServers(dir, servers); err != nil {
		t.Fatal(err)
	}

	readBack, err := d.ReadServers(dir)
	if err != nil {
		t.Fatal(err)
	}

	got := readBack["minimal"]
	if got.Command != "echo" {
		t.Errorf("Command = %q, want %q", got.Command, "echo")
	}
	if got.Transport != model.TransportStdio {
		t.Errorf("Transport = %q, want %q", got.Transport, model.TransportStdio)
	}
}
