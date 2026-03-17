package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestServerDefYAMLRoundTrip_Stdio(t *testing.T) {
	input := `name: github
description: GitHub API integration
transport: stdio
command: npx
args:
    - "-y"
    - "@modelcontextprotocol/server-github"
env:
    GITHUB_TOKEN: ${GITHUB_TOKEN}
`
	var s ServerDef
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if s.Name != "github" {
		t.Errorf("Name = %q, want %q", s.Name, "github")
	}
	if s.Transport != TransportStdio {
		t.Errorf("Transport = %q, want %q", s.Transport, TransportStdio)
	}
	if s.Command != "npx" {
		t.Errorf("Command = %q, want %q", s.Command, "npx")
	}
	if len(s.Args) != 2 || s.Args[0] != "-y" || s.Args[1] != "@modelcontextprotocol/server-github" {
		t.Errorf("Args = %v, want [-y @modelcontextprotocol/server-github]", s.Args)
	}
	if s.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("Env[GITHUB_TOKEN] = %q, want %q", s.Env["GITHUB_TOKEN"], "${GITHUB_TOKEN}")
	}
	if s.URL != "" {
		t.Errorf("URL = %q, want empty", s.URL)
	}

	// Re-marshal and unmarshal to verify round-trip
	out, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s2 ServerDef
	if err := yaml.Unmarshal(out, &s2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if s2.Name != s.Name || s2.Transport != s.Transport || s2.Command != s.Command {
		t.Errorf("round-trip mismatch: got %+v, want %+v", s2, s)
	}
}

func TestServerDefYAMLRoundTrip_HTTP(t *testing.T) {
	input := `name: remote-api
description: Remote API server
transport: http
url: https://mcp.example.com/mcp
headers:
    Authorization: Bearer ${API_TOKEN}
`
	var s ServerDef
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if s.Name != "remote-api" {
		t.Errorf("Name = %q, want %q", s.Name, "remote-api")
	}
	if s.Transport != TransportHTTP {
		t.Errorf("Transport = %q, want %q", s.Transport, TransportHTTP)
	}
	if s.URL != "https://mcp.example.com/mcp" {
		t.Errorf("URL = %q, want %q", s.URL, "https://mcp.example.com/mcp")
	}
	if s.Headers["Authorization"] != "Bearer ${API_TOKEN}" {
		t.Errorf("Headers[Authorization] = %q, want %q", s.Headers["Authorization"], "Bearer ${API_TOKEN}")
	}
	if s.Command != "" {
		t.Errorf("Command = %q, want empty", s.Command)
	}

	// Round-trip
	out, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s2 ServerDef
	if err := yaml.Unmarshal(out, &s2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if s2.URL != s.URL || s2.Transport != s.Transport {
		t.Errorf("round-trip mismatch: got %+v, want %+v", s2, s)
	}
}

func TestServerOverrideYAML(t *testing.T) {
	cmd := "node"
	input := ServerOverride{
		Command: &cmd,
		Args:    []string{"--flag"},
		Env:     map[string]string{"KEY": "val"},
	}

	out, err := yaml.Marshal(&input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ServerOverride
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Command == nil || *got.Command != "node" {
		t.Errorf("Command = %v, want 'node'", got.Command)
	}
	if len(got.Args) != 1 || got.Args[0] != "--flag" {
		t.Errorf("Args = %v, want [--flag]", got.Args)
	}
	if got.Env["KEY"] != "val" {
		t.Errorf("Env[KEY] = %q, want %q", got.Env["KEY"], "val")
	}
	if got.URL != nil {
		t.Errorf("URL = %v, want nil", got.URL)
	}
}
