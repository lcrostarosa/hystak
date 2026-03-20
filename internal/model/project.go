package model

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Project represents a registered project with its server assignments.
type Project struct {
	Name          string                   `yaml:"-"`
	Path          string                   `yaml:"path"`
	Clients       []ClientType             `yaml:"clients"`
	Tags          []string                 `yaml:"tags,omitempty"`
	MCPs          []MCPAssignment          `yaml:"mcps,omitempty"`
	Skills        []string                 `yaml:"skills,omitempty"`
	Hooks         []string                 `yaml:"hooks,omitempty"`
	Permissions   []string                 `yaml:"permissions,omitempty"`
	ClaudeMD      string                   `yaml:"claude_md,omitempty"`
	Profiles      map[string]ProjectProfile `yaml:"profiles,omitempty"`
	ActiveProfile string                   `yaml:"active_profile,omitempty"`
	Launched      bool                     `yaml:"launched,omitempty"`
	ManagedMCPs   []string                 `yaml:"managed_mcps,omitempty"` // server names deployed by hystak in last sync
}

// ProjectProfile is a profile stored inline in a project config.
type ProjectProfile struct {
	Description string            `yaml:"description,omitempty"`
	MCPs        []string          `yaml:"mcps,omitempty"`
	Skills      []string          `yaml:"skills,omitempty"`
	Hooks       []string          `yaml:"hooks,omitempty"`
	Permissions []string          `yaml:"permissions,omitempty"`
	EnvVars     map[string]string `yaml:"env,omitempty"`
	ClaudeMD    string            `yaml:"claude_md,omitempty"`
	Isolation   string            `yaml:"isolation,omitempty"`
}

// MCPAssignment represents a server assigned to a project,
// optionally with overrides.
//
// Supports dual YAML format:
//   - Bare string:  "- github"
//   - Map with overrides:  "- github: {overrides: {env: {KEY: val}}}"
type MCPAssignment struct {
	Name      string          `yaml:"-"`
	Overrides *ServerOverride `yaml:"overrides,omitempty"`
}

// mcpAssignmentValue is the inner value when MCPAssignment is serialized as a map.
type mcpAssignmentValue struct {
	Overrides *ServerOverride `yaml:"overrides,omitempty"`
}

// MarshalYAML implements yaml.Marshaler for MCPAssignment.
// Bare names serialize as a scalar string; entries with overrides serialize as a single-key map.
func (a MCPAssignment) MarshalYAML() (interface{}, error) {
	if a.Overrides == nil {
		return a.Name, nil
	}
	return map[string]mcpAssignmentValue{
		a.Name: {Overrides: a.Overrides},
	}, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for MCPAssignment.
// Accepts both a scalar string and a single-key map.
func (a *MCPAssignment) UnmarshalYAML(value *yaml.Node) error {
	// Bare string: "- github"
	if value.Kind == yaml.ScalarNode {
		a.Name = value.Value
		a.Overrides = nil
		return nil
	}

	// Map: "- github: {overrides: ...}"
	if value.Kind == yaml.MappingNode {
		if len(value.Content) != 2 {
			return fmt.Errorf("MCPAssignment map must have exactly one key, got %d", len(value.Content)/2)
		}
		a.Name = value.Content[0].Value
		var val mcpAssignmentValue
		if err := value.Content[1].Decode(&val); err != nil {
			return fmt.Errorf("decoding MCPAssignment value for %q: %w", a.Name, err)
		}
		a.Overrides = val.Overrides
		return nil
	}

	return fmt.Errorf("MCPAssignment: expected string or map, got %v", value.Kind)
}

// DriftStatus represents the sync state of a server in a project.
type DriftStatus string

const (
	DriftSynced    DriftStatus = "synced"
	DriftDrifted   DriftStatus = "drifted"
	DriftMissing   DriftStatus = "missing"
	DriftUnmanaged DriftStatus = "unmanaged"
)

// ServerDriftReport holds drift info for a single server in a project+client.
type ServerDriftReport struct {
	ServerName string
	Status     DriftStatus
	Expected   *ServerDef // nil if unmanaged
	Deployed   *ServerDef // nil if missing
}
