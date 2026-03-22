package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/registry"
)

func TestListCmd_EmptyRegistry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	reg := registry.New()
	if err := reg.Save(filepath.Join(tmp, "registry.yaml")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected header row with NAME")
	}
	if !strings.Contains(output, "TRANSPORT") {
		t.Error("expected header row with TRANSPORT")
	}
	if !strings.Contains(output, "COMMAND/URL") {
		t.Error("expected header row with COMMAND/URL")
	}
}

func TestListCmd_WithServers(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	reg := registry.New()
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "remote-api",
		Transport: model.TransportSSE,
		URL:       "https://mcp.example.com/sse",
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Save(filepath.Join(tmp, "registry.yaml")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "github") {
		t.Error("expected 'github' in output")
	}
	if !strings.Contains(output, "stdio") {
		t.Error("expected 'stdio' in output")
	}
	if !strings.Contains(output, "npx") {
		t.Error("expected 'npx' in output")
	}
	if !strings.Contains(output, "remote-api") {
		t.Error("expected 'remote-api' in output")
	}
	if !strings.Contains(output, "sse") {
		t.Error("expected 'sse' in output")
	}
	if !strings.Contains(output, "https://mcp.example.com/sse") {
		t.Error("expected URL in output")
	}
}

func TestListCmd_SortedByName(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	reg := registry.New()
	if err := reg.Servers.Add(model.ServerDef{Name: "zebra", Transport: model.TransportStdio, Command: "z"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Servers.Add(model.ServerDef{Name: "alpha", Transport: model.TransportStdio, Command: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Save(filepath.Join(tmp, "registry.yaml")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	alphaIdx := strings.Index(output, "alpha")
	zebraIdx := strings.Index(output, "zebra")
	if alphaIdx < 0 || zebraIdx < 0 {
		t.Fatal("expected both 'alpha' and 'zebra' in output")
	}
	if alphaIdx > zebraIdx {
		t.Error("expected 'alpha' before 'zebra' (sorted by name)")
	}
}

func TestListCmd_NoRegistryFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected header row even with no registry file")
	}
}

func TestCommandOrURL(t *testing.T) {
	tests := []struct {
		name   string
		server model.ServerDef
		want   string
	}{
		{
			name:   "stdio returns command",
			server: model.ServerDef{Transport: model.TransportStdio, Command: "npx"},
			want:   "npx",
		},
		{
			name:   "sse returns url",
			server: model.ServerDef{Transport: model.TransportSSE, URL: "https://example.com/sse"},
			want:   "https://example.com/sse",
		},
		{
			name:   "http returns url",
			server: model.ServerDef{Transport: model.TransportHTTP, URL: "https://example.com"},
			want:   "https://example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandOrURL(tt.server); got != tt.want {
				t.Errorf("commandOrURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
