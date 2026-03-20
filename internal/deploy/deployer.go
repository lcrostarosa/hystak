package deploy

import (
	"fmt"

	"github.com/lcrostarosa/hystak/internal/model"
)

// PreflightConflict represents a resource that already exists in the project
// but was not placed by hystak.
type PreflightConflict struct {
	ResourceType string // "skill", "hook", "permission", "claude_md"
	Name         string
	ExistingPath string
}

// Deployer writes and reads MCP server configs for a specific client.
type Deployer interface {
	// ClientType returns which client this deployer handles.
	ClientType() model.ClientType

	// ConfigPath returns the config file path for this client+project.
	// projectPath is the project's filesystem path (empty or "~" for global).
	ConfigPath(projectPath string) string

	// ReadServers reads current MCP server configs from the client config.
	// Returns the canonical ServerDef format (translated from client format).
	ReadServers(projectPath string) (map[string]model.ServerDef, error)

	// WriteServers writes MCP servers to the client config, translating
	// to the client's expected format. Preserves all non-mcpServers keys.
	WriteServers(projectPath string, servers map[string]model.ServerDef) error

	// Bootstrap creates the config file if it doesn't exist.
	Bootstrap(projectPath string) error
}

var deployerFactories = map[model.ClientType]func() Deployer{
	model.ClientClaudeCode: func() Deployer { return &ClaudeCodeDeployer{} },
}

var unimplemented = map[model.ClientType]bool{
	model.ClientClaudeDesktop: true,
	model.ClientCursor:        true,
}

// NewDeployer returns a Deployer for the given client type.
func NewDeployer(ct model.ClientType) (Deployer, error) {
	if f, ok := deployerFactories[ct]; ok {
		return f(), nil
	}
	if unimplemented[ct] {
		return nil, fmt.Errorf("%s deployer is not yet implemented", ct)
	}
	return nil, fmt.Errorf("unknown client type: %s", ct)
}
