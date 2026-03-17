package deploy

import (
	"fmt"

	"github.com/lcrostarosa/hystak/internal/model"
)

// ClaudeDesktopDeployer is a stub deployer for Claude Desktop.
// Not yet implemented in v1.
type ClaudeDesktopDeployer struct{}

func (d *ClaudeDesktopDeployer) ClientType() model.ClientType {
	return model.ClientClaudeDesktop
}

func (d *ClaudeDesktopDeployer) ConfigPath(_ string) string {
	return ""
}

func (d *ClaudeDesktopDeployer) ReadServers(_ string) (map[string]model.ServerDef, error) {
	return nil, fmt.Errorf("claude-desktop deployer is not yet implemented")
}

func (d *ClaudeDesktopDeployer) WriteServers(_ string, _ map[string]model.ServerDef) error {
	return fmt.Errorf("claude-desktop deployer is not yet implemented")
}

func (d *ClaudeDesktopDeployer) Bootstrap(_ string) error {
	return fmt.Errorf("claude-desktop deployer is not yet implemented")
}
