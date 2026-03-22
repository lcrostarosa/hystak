package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTransport_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value Transport
		want  bool
	}{
		{"stdio", TransportStdio, true},
		{"sse", TransportSSE, true},
		{"http", TransportHTTP, true},
		{"empty", Transport(""), false},
		{"unknown", Transport("grpc"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("Transport(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestServerDef_ResourceName(t *testing.T) {
	s := &ServerDef{Name: "github"}
	if got := s.ResourceName(); got != "github" {
		t.Errorf("ResourceName() = %q, want %q", got, "github")
	}
	s.SetResourceName("gitlab")
	if got := s.ResourceName(); got != "gitlab" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "gitlab")
	}
}

func TestServerDef_Equal(t *testing.T) {
	tests := []struct {
		name string
		a, b ServerDef
		want bool
	}{
		{
			name: "identical stdio servers",
			a:    ServerDef{Transport: TransportStdio, Command: "npx", Args: []string{"-y", "server"}},
			b:    ServerDef{Transport: TransportStdio, Command: "npx", Args: []string{"-y", "server"}},
			want: true,
		},
		{
			name: "different names same fields",
			a:    ServerDef{Name: "a", Transport: TransportStdio, Command: "npx"},
			b:    ServerDef{Name: "b", Transport: TransportStdio, Command: "npx"},
			want: true, // Name is metadata, not compared
		},
		{
			name: "different descriptions same fields",
			a:    ServerDef{Transport: TransportStdio, Command: "npx", Description: "first"},
			b:    ServerDef{Transport: TransportStdio, Command: "npx", Description: "second"},
			want: true, // Description is metadata
		},
		{
			name: "different transport",
			a:    ServerDef{Transport: TransportStdio},
			b:    ServerDef{Transport: TransportSSE},
			want: false,
		},
		{
			name: "different command",
			a:    ServerDef{Transport: TransportStdio, Command: "npx"},
			b:    ServerDef{Transport: TransportStdio, Command: "node"},
			want: false,
		},
		{
			name: "different URL",
			a:    ServerDef{Transport: TransportSSE, URL: "http://a.com"},
			b:    ServerDef{Transport: TransportSSE, URL: "http://b.com"},
			want: false,
		},
		{
			name: "nil vs empty args",
			a:    ServerDef{Transport: TransportStdio, Args: nil},
			b:    ServerDef{Transport: TransportStdio, Args: []string{}},
			want: true,
		},
		{
			name: "nil vs empty env",
			a:    ServerDef{Transport: TransportStdio, Env: nil},
			b:    ServerDef{Transport: TransportStdio, Env: map[string]string{}},
			want: true,
		},
		{
			name: "nil vs empty headers",
			a:    ServerDef{Transport: TransportSSE, URL: "http://x.com", Headers: nil},
			b:    ServerDef{Transport: TransportSSE, URL: "http://x.com", Headers: map[string]string{}},
			want: true,
		},
		{
			name: "different args",
			a:    ServerDef{Transport: TransportStdio, Args: []string{"a"}},
			b:    ServerDef{Transport: TransportStdio, Args: []string{"b"}},
			want: false,
		},
		{
			name: "different env",
			a:    ServerDef{Transport: TransportStdio, Env: map[string]string{"K": "1"}},
			b:    ServerDef{Transport: TransportStdio, Env: map[string]string{"K": "2"}},
			want: false,
		},
		{
			name: "different headers",
			a:    ServerDef{Transport: TransportSSE, URL: "http://x.com", Headers: map[string]string{"A": "1"}},
			b:    ServerDef{Transport: TransportSSE, URL: "http://x.com", Headers: map[string]string{"A": "2"}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerDef_YAMLRoundTrip_Stdio(t *testing.T) {
	original := ServerDef{
		Name:      "github",
		Transport: TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@anthropic/mcp-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored ServerDef
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !original.Equal(restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
	if restored.Name != original.Name {
		t.Errorf("Name: got %q, want %q", restored.Name, original.Name)
	}
}

func TestServerDef_YAMLRoundTrip_HTTP(t *testing.T) {
	original := ServerDef{
		Name:      "remote",
		Transport: TransportHTTP,
		URL:       "https://mcp.example.com",
		Headers:   map[string]string{"Authorization": "Bearer ${TOKEN}"},
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored ServerDef
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !original.Equal(restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}

func TestServerOverride_Apply(t *testing.T) {
	base := ServerDef{
		Transport: TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "server"},
		Env:       map[string]string{"A": "1", "B": "2"},
		Headers:   map[string]string{"X": "1"},
	}

	t.Run("nil override returns copy", func(t *testing.T) {
		var o *ServerOverride
		result := o.Apply(base)
		if !result.Equal(base) {
			t.Errorf("nil override should return copy of base")
		}
	})

	t.Run("command override", func(t *testing.T) {
		cmd := "node"
		o := &ServerOverride{Command: &cmd}
		result := o.Apply(base)
		if result.Command != "node" {
			t.Errorf("Command = %q, want %q", result.Command, "node")
		}
	})

	t.Run("url override", func(t *testing.T) {
		url := "http://new.com"
		o := &ServerOverride{URL: &url}
		result := o.Apply(base)
		if result.URL != "http://new.com" {
			t.Errorf("URL = %q, want %q", result.URL, "http://new.com")
		}
	})

	t.Run("args replacement", func(t *testing.T) {
		o := &ServerOverride{Args: []string{"--verbose"}}
		result := o.Apply(base)
		if len(result.Args) != 1 || result.Args[0] != "--verbose" {
			t.Errorf("Args = %v, want [--verbose]", result.Args)
		}
	})

	t.Run("env map merge", func(t *testing.T) {
		o := &ServerOverride{Env: map[string]string{"B": "3", "C": "4"}}
		result := o.Apply(base)
		if result.Env["A"] != "1" {
			t.Errorf("Env[A] = %q, want %q", result.Env["A"], "1")
		}
		if result.Env["B"] != "3" {
			t.Errorf("Env[B] = %q, want %q (override should win)", result.Env["B"], "3")
		}
		if result.Env["C"] != "4" {
			t.Errorf("Env[C] = %q, want %q", result.Env["C"], "4")
		}
	})

	t.Run("headers map merge", func(t *testing.T) {
		o := &ServerOverride{Headers: map[string]string{"X": "2", "Y": "3"}}
		result := o.Apply(base)
		if result.Headers["X"] != "2" {
			t.Errorf("Headers[X] = %q, want %q (override should win)", result.Headers["X"], "2")
		}
		if result.Headers["Y"] != "3" {
			t.Errorf("Headers[Y] = %q, want %q", result.Headers["Y"], "3")
		}
	})
}
