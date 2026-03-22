package model

import (
	"maps"
	"slices"
)

// Transport represents the MCP server transport type.
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportSSE   Transport = "sse"
	TransportHTTP  Transport = "http"
)

// ServerDef is the canonical MCP server definition stored in the registry.
type ServerDef struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Transport   Transport         `yaml:"transport"`
	Command     string            `yaml:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	URL         string            `yaml:"url,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
}

func (s *ServerDef) ResourceName() string    { return s.Name }
func (s *ServerDef) SetResourceName(n string) { s.Name = n }

// Equal performs semantic comparison, ignoring Name and Description (registry-only metadata).
func (a ServerDef) Equal(b ServerDef) bool {
	return a.Transport == b.Transport &&
		a.Command == b.Command &&
		a.URL == b.URL &&
		slices.Equal(a.Args, b.Args) &&
		maps.Equal(a.Env, b.Env) &&
		maps.Equal(a.Headers, b.Headers)
}

// Target returns the transport-aware display field: URL for SSE/HTTP, Command for stdio.
func (s ServerDef) Target() string {
	switch s.Transport {
	case TransportSSE, TransportHTTP:
		return s.URL
	default:
		return s.Command
	}
}

// ServerOverride holds per-project field overrides for a server.
// Only non-nil/non-empty fields are applied during merge.
type ServerOverride struct {
	Command *string           `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	URL     *string           `yaml:"url,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}
