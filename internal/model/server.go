package model

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

// ServerOverride holds per-project field overrides for a server.
// Only non-nil/non-empty fields are applied during merge.
type ServerOverride struct {
	Command *string           `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	URL     *string           `yaml:"url,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}
