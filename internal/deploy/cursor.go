package deploy

import (
	"fmt"

	"github.com/rbbydotdev/hystak/internal/model"
)

// CursorDeployer is a stub deployer for Cursor.
// Not yet implemented in v1.
type CursorDeployer struct{}

func (d *CursorDeployer) ClientType() model.ClientType {
	return model.ClientCursor
}

func (d *CursorDeployer) ConfigPath(_ string) string {
	return ""
}

func (d *CursorDeployer) ReadServers(_ string) (map[string]model.ServerDef, error) {
	return nil, fmt.Errorf("cursor deployer is not yet implemented")
}

func (d *CursorDeployer) WriteServers(_ string, _ map[string]model.ServerDef) error {
	return fmt.Errorf("cursor deployer is not yet implemented")
}

func (d *CursorDeployer) Bootstrap(_ string) error {
	return fmt.Errorf("cursor deployer is not yet implemented")
}
