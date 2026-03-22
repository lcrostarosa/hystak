package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestClaudeCodeDeployer_ClientType(t *testing.T) {
	d := &ClaudeCodeDeployer{}
	if got := d.ClientType(); got != model.ClientClaudeCode {
		t.Errorf("ClientType() = %q, want %q", got, model.ClientClaudeCode)
	}
}

func TestClaudeCodeDeployer_ConfigPath(t *testing.T) {
	d := &ClaudeCodeDeployer{}
	got := d.ConfigPath("/test/project")
	want := filepath.Join("/test/project", ".mcp.json")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestClaudeCodeDeployer_Bootstrap_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

	if err := d.Bootstrap(tmp); err != nil {
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
	if _, ok := parsed["mcpServers"]; !ok {
		t.Error("missing mcpServers key")
	}
}

func TestClaudeCodeDeployer_Bootstrap_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

	if err := d.Bootstrap(tmp); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmp, ".mcp.json")
	custom := []byte(`{"mcpServers":{"github":{}},"other":"keep"}`)
	if err := os.WriteFile(path, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := d.Bootstrap(tmp); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(custom) {
		t.Error("Bootstrap overwrote existing file")
	}
}

func TestClaudeCodeDeployer_ReadServers_Empty(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

	servers, err := d.ReadServers(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Errorf("expected empty map, got %d servers", len(servers))
	}
}

func TestClaudeCodeDeployer_ReadServers(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

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
	if err := os.WriteFile(filepath.Join(tmp, ".mcp.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	servers, err := d.ReadServers(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(servers))
	}

	gh := servers["github"]
	if gh.Transport != model.TransportStdio {
		t.Errorf("github.Transport = %q", gh.Transport)
	}
	if gh.Command != "npx" {
		t.Errorf("github.Command = %q", gh.Command)
	}
	if !reflect.DeepEqual(gh.Args, []string{"-y", "@anthropic/mcp-github"}) {
		t.Errorf("github.Args = %v", gh.Args)
	}
	if gh.Env["GITHUB_TOKEN"] != "tok" {
		t.Errorf("github.Env[GITHUB_TOKEN] = %q, want tok", gh.Env["GITHUB_TOKEN"])
	}

	remote := servers["remote"]
	if remote.Transport != model.TransportSSE {
		t.Errorf("remote.Transport = %q", remote.Transport)
	}
	if remote.URL != "https://example.com/sse" {
		t.Errorf("remote.URL = %q", remote.URL)
	}
	if remote.Headers["Auth"] != "Bearer tok" {
		t.Errorf("remote.Headers[Auth] = %q, want \"Bearer tok\"", remote.Headers["Auth"])
	}
}

func TestClaudeCodeDeployer_WriteServers_PreservesNonMCPKeys(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

	initial := `{"mcpServers":{},"otherKey":"preserved"}`
	if err := os.WriteFile(filepath.Join(tmp, ".mcp.json"), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	servers := map[string]model.ServerDef{
		"github": {Transport: model.TransportStdio, Command: "npx"},
	}

	if err := d.WriteServers(tmp, servers); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if _, ok := parsed["otherKey"]; !ok {
		t.Error("non-mcpServers key 'otherKey' was not preserved")
	}
}

func TestClaudeCodeDeployer_WriteServers_NewFile(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

	servers := map[string]model.ServerDef{
		"github": {Transport: model.TransportStdio, Command: "npx"},
	}

	if err := d.WriteServers(tmp, servers); err != nil {
		t.Fatal(err)
	}

	read, err := d.ReadServers(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != 1 {
		t.Errorf("got %d servers, want 1", len(read))
	}
}

func TestClaudeCodeDeployer_WriteRead_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeCodeDeployer{}

	original := map[string]model.ServerDef{
		"github": {
			Name:      "github",
			Transport: model.TransportStdio,
			Command:   "npx",
			Args:      []string{"-y", "server"},
			Env:       map[string]string{"TOKEN": "abc"},
		},
		"remote": {
			Name:      "remote",
			Transport: model.TransportSSE,
			URL:       "https://example.com",
			Headers:   map[string]string{"Auth": "Bearer tok"},
		},
	}

	if err := d.WriteServers(tmp, original); err != nil {
		t.Fatal(err)
	}

	read, err := d.ReadServers(tmp)
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range original {
		got, ok := read[name]
		if !ok {
			t.Errorf("missing server %q", name)
			continue
		}
		if !want.Equal(got) {
			t.Errorf("server %q mismatch:\n  got:  %+v\n  want: %+v", name, got, want)
		}
	}
}

func TestNewDeployer_ClaudeCode(t *testing.T) {
	d, ok := NewDeployer(model.ClientClaudeCode)
	if !ok {
		t.Fatal("NewDeployer returned false for claude-code")
	}
	if d.ClientType() != model.ClientClaudeCode {
		t.Errorf("ClientType() = %q", d.ClientType())
	}
}

func TestNewDeployer_Unknown(t *testing.T) {
	_, ok := NewDeployer(model.ClientType("unknown"))
	if ok {
		t.Error("NewDeployer returned true for unknown client")
	}
}
