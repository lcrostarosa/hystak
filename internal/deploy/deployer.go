package deploy

import (
	"fmt"

	"github.com/lcrostarosa/hystak/internal/model"
)

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

// NewDeployer returns a Deployer for the given client type.
func NewDeployer(ct model.ClientType) (Deployer, error) {
	switch ct {
	case model.ClientClaudeCode:
		return &ClaudeCodeDeployer{}, nil
	case model.ClientClaudeDesktop:
		return nil, fmt.Errorf("claude-desktop deployer is not yet implemented")
	case model.ClientCursor:
		return nil, fmt.Errorf("cursor deployer is not yet implemented")
	default:
		return nil, fmt.Errorf("unknown client type: %s", ct)
	}
}
