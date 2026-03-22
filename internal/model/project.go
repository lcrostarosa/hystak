package model

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ClientType identifies an MCP client application.
type ClientType string

const (
	ClientClaudeCode    ClientType = "claude-code"
	ClientClaudeDesktop ClientType = "claude-desktop"
	ClientCursor        ClientType = "cursor"
)

// Valid reports whether ct is a known client type.
func (ct ClientType) Valid() bool {
	switch ct {
	case ClientClaudeCode, ClientClaudeDesktop, ClientCursor:
		return true
	}
	return false
}

// IsolationStrategy controls how concurrent sessions are isolated.
type IsolationStrategy string

const (
	IsolationNone     IsolationStrategy = "none"
	IsolationWorktree IsolationStrategy = "worktree"
	IsolationLock     IsolationStrategy = "lock"
)

// Valid reports whether s is a known isolation strategy.
func (s IsolationStrategy) Valid() bool {
	switch s {
	case IsolationNone, IsolationWorktree, IsolationLock:
		return true
	}
	return false
}

// Project is a registered project directory with its configuration.
type Project struct {
	Name          string       `yaml:"name,omitempty"`
	Path          string       `yaml:"path"`
	ActiveProfile string       `yaml:"active_profile,omitempty"`
	ManagedMCPs   []string     `yaml:"managed_mcps,omitempty"`
	Clients       []ClientType `yaml:"clients,omitempty"`
}

func (p *Project) ResourceName() string     { return p.Name }
func (p *Project) SetResourceName(n string) { p.Name = n }

// ProjectProfile is a named loadout within a project.
type ProjectProfile struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Scope       string            `yaml:"scope,omitempty"`
	Project     string            `yaml:"project,omitempty"`
	MCPs        []MCPAssignment   `yaml:"mcps,omitempty"`
	Skills      []string          `yaml:"skills,omitempty"`
	Hooks       []string          `yaml:"hooks,omitempty"`
	Permissions []string          `yaml:"permissions,omitempty"`
	Template    string            `yaml:"template,omitempty"`
	Prompts     []string          `yaml:"prompts,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Isolation   IsolationStrategy `yaml:"isolation,omitempty"`
}

// MCPAssignment is a server assignment that supports two YAML formats:
//   - bare string: "github"
//   - map with overrides: github: {overrides: {env: {KEY: val}}}
type MCPAssignment struct {
	Name      string          `yaml:"-"`
	Overrides *ServerOverride `yaml:"overrides,omitempty"`
}

// mcpAssignmentValue is the internal structure for the map YAML format.
type mcpAssignmentValue struct {
	Overrides *ServerOverride `yaml:"overrides,omitempty"`
}

// MarshalYAML implements custom YAML marshaling for MCPAssignment.
// Bare string when no overrides, map when overrides are present.
func (a MCPAssignment) MarshalYAML() (interface{}, error) {
	if a.Overrides == nil {
		return a.Name, nil
	}
	return map[string]mcpAssignmentValue{
		a.Name: {Overrides: a.Overrides},
	}, nil
}

// UnmarshalYAML implements custom YAML unmarshaling for MCPAssignment.
// Accepts bare string or single-key map.
func (a *MCPAssignment) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		a.Name = value.Value
		return nil
	}
	if value.Kind == yaml.MappingNode {
		if len(value.Content) < 2 {
			return fmt.Errorf("MCP assignment map must have exactly one key")
		}
		a.Name = value.Content[0].Value
		var val mcpAssignmentValue
		if err := value.Content[1].Decode(&val); err != nil {
			return fmt.Errorf("decoding MCP assignment value for %q: %w", a.Name, err)
		}
		a.Overrides = val.Overrides
		return nil
	}
	return fmt.Errorf("MCP assignment: expected string or map, got YAML kind %d", value.Kind)
}
