package deploy

import "github.com/hystak/hystak/internal/model"

// Deployer is the interface for client-specific MCP server deployment.
// Each MCP client (Claude Code, Cursor, etc.) has its own deployer.
type Deployer interface {
	// ClientType returns which client this deployer handles.
	ClientType() model.ClientType

	// ConfigPath returns the path to the client's config file for a project.
	ConfigPath(projectPath string) string

	// Bootstrap ensures the client config file exists with correct structure.
	Bootstrap(projectPath string) error

	// ReadServers reads the currently deployed MCP servers from the config.
	ReadServers(projectPath string) (map[string]model.ServerDef, error)

	// WriteServers writes the full server map to the config file.
	// Non-mcpServers keys in the existing file are preserved.
	WriteServers(projectPath string, servers map[string]model.ServerDef) error
}

// NewDeployer creates a deployer for the given client type.
func NewDeployer(ct model.ClientType) (Deployer, bool) {
	factory, ok := deployerFactories[ct]
	if !ok {
		return nil, false
	}
	return factory(), true
}

var deployerFactories = map[model.ClientType]func() Deployer{
	model.ClientClaudeCode: func() Deployer { return &ClaudeCodeDeployer{} },
}
